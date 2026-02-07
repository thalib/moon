// Package server provides HTTP server setup and routing.
// It configures the HTTP router with all API endpoints following
// the AIP-136 custom actions pattern defined in SPEC.md.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/auth"
	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/handlers"
	"github.com/thalib/moon/cmd/moon/internal/middleware"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// Server represents the HTTP server
type Server struct {
	config         *config.AppConfig
	db             database.Driver
	registry       *registry.SchemaRegistry
	mux            *http.ServeMux
	server         *http.Server
	version        string
	rateLimiter    *middleware.RateLimitMiddleware
	authzMiddle    *middleware.AuthorizationMiddleware
	corsMiddle     *middleware.CORSMiddleware
	tokenService   *auth.TokenService
	tokenBlacklist *auth.TokenBlacklist
	apiKeyRepo     *auth.APIKeyRepository
}

// New creates a new server instance
func New(cfg *config.AppConfig, db database.Driver, reg *registry.SchemaRegistry, version string) *Server {
	mux := http.NewServeMux()

	// Create rate limiter with config values
	rateLimiterConfig := middleware.RateLimiterConfig{
		UserRPM:   cfg.Auth.RateLimit.UserRPM,
		APIKeyRPM: cfg.Auth.RateLimit.APIKeyRPM,
	}
	if rateLimiterConfig.UserRPM == 0 {
		rateLimiterConfig.UserRPM = config.Defaults.Auth.RateLimit.UserRPM
	}
	if rateLimiterConfig.APIKeyRPM == 0 {
		rateLimiterConfig.APIKeyRPM = config.Defaults.Auth.RateLimit.APIKeyRPM
	}

	// Create CORS middleware with config values
	corsConfig := middleware.CORSConfig{
		Enabled:          cfg.CORS.Enabled,
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}

	// Create token service for authentication
	accessExpiry := cfg.JWT.AccessExpiry
	if accessExpiry == 0 {
		accessExpiry = cfg.JWT.Expiry
	}
	tokenService := auth.NewTokenService(cfg.JWT.Secret, accessExpiry, cfg.JWT.RefreshExpiry)

	srv := &Server{
		config:         cfg,
		db:             db,
		registry:       reg,
		mux:            mux,
		version:        version,
		rateLimiter:    middleware.NewRateLimitMiddleware(rateLimiterConfig),
		authzMiddle:    middleware.NewAuthorizationMiddleware(),
		corsMiddle:     middleware.NewCORSMiddleware(corsConfig),
		tokenService:   tokenService,
		tokenBlacklist: auth.NewTokenBlacklist(db),
		apiKeyRepo:     auth.NewAPIKeyRepository(db),
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      mux,
			ReadTimeout:  constants.HTTPReadTimeout,
			WriteTimeout: constants.HTTPWriteTimeout,
			IdleTimeout:  constants.HTTPIdleTimeout,
		},
	}

	srv.setupRoutes()
	return srv
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Create collections handler
	collectionsHandler := handlers.NewCollectionsHandler(s.db, s.registry)

	// Create data handler
	dataHandler := handlers.NewDataHandler(s.db, s.registry)

	// Create aggregation handler
	aggregationHandler := handlers.NewAggregationHandler(s.db, s.registry)

	// Create documentation handler
	docHandler := handlers.NewDocHandler(s.registry, s.config, s.version)

	// Create auth handler with login rate limiting
	accessExpiry := s.config.JWT.AccessExpiry
	if accessExpiry == 0 {
		accessExpiry = s.config.JWT.Expiry // fallback to legacy config
	}
	refreshExpiry := s.config.JWT.RefreshExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 604800 // 7 days default
	}
	authHandler := handlers.NewAuthHandler(s.db, s.config.JWT.Secret, accessExpiry, refreshExpiry)

	// Create users handler (admin only endpoints)
	usersHandler := handlers.NewUsersHandler(s.db, s.config.JWT.Secret, accessExpiry, refreshExpiry)

	// Create API keys handler (admin only endpoints)
	apiKeysHandler := handlers.NewAPIKeysHandler(s.db, s.config.JWT.Secret, accessExpiry, refreshExpiry)

	// Get the prefix from config
	prefix := s.config.Server.Prefix

	// Middleware helper functions for cleaner route definitions
	// Public endpoints: Public CORS (Access-Control-Allow-Origin: *) + logging (PRD-052)
	publicCORS := func(h http.HandlerFunc) http.HandlerFunc {
		return s.corsMiddle.HandlePublic(s.loggingMiddleware(h))
	}

	// Public endpoints: Standard CORS + logging (for endpoints like root message)
	public := func(h http.HandlerFunc) http.HandlerFunc {
		return s.corsMiddle.Handle(s.loggingMiddleware(h))
	}

	// Auth endpoints: CORS + logging + auth (login/refresh don't need rate limit or authz)
	authNoLimit := func(h http.HandlerFunc) http.HandlerFunc {
		return s.corsMiddle.Handle(s.loggingMiddleware(h))
	}

	// Authenticated: CORS + logging + auth + rate limit (any authenticated entity)
	authenticated := func(h http.HandlerFunc) http.HandlerFunc {
		return s.corsMiddle.Handle(
			s.loggingMiddleware(
				s.authMiddleware(
					s.rateLimiter.RateLimit(
						s.authzMiddle.RequireAuthenticated(h)))))
	}

	// Admin only: CORS + logging + auth + rate limit + admin role
	adminOnly := func(h http.HandlerFunc) http.HandlerFunc {
		return s.corsMiddle.Handle(
			s.loggingMiddleware(
				s.authMiddleware(
					s.rateLimiter.RateLimit(
						s.authzMiddle.RequireAdmin(h)))))
	}

	// Write required: CORS + logging + auth + rate limit + write permission
	writeRequired := func(h http.HandlerFunc) http.HandlerFunc {
		return s.corsMiddle.Handle(
			s.loggingMiddleware(
				s.authMiddleware(
					s.rateLimiter.RateLimit(
						s.authzMiddle.RequireWrite(h)))))
	}

	// Root message endpoint (only for exact "/" path with no prefix)
	if prefix == "" {
		s.mux.HandleFunc("GET /{$}", public(s.rootMessageHandler))
	}

	// ==========================================
	// PUBLIC ENDPOINTS (No Auth)
	// ==========================================

	// Health check endpoint (always at /health, respects prefix) - PRD-052: Public CORS
	healthPath := prefix + "/health"
	s.mux.HandleFunc("GET "+healthPath, publicCORS(s.healthHandler))
	s.mux.HandleFunc("OPTIONS "+healthPath, publicCORS(s.healthHandler))

	// Documentation endpoints (public) - PRD-052: Public CORS
	s.mux.HandleFunc("GET "+prefix+"/doc/{$}", publicCORS(docHandler.HTML))
	s.mux.HandleFunc("OPTIONS "+prefix+"/doc/{$}", publicCORS(docHandler.HTML))
	s.mux.HandleFunc("GET "+prefix+"/doc/llms-full.txt", publicCORS(docHandler.Markdown))
	s.mux.HandleFunc("OPTIONS "+prefix+"/doc/llms-full.txt", publicCORS(docHandler.Markdown))

	// ==========================================
	// AUTH ENDPOINTS (No role check)
	// ==========================================

	// Login and refresh don't need auth/rate limit (they have their own rate limiting)
	s.mux.HandleFunc("POST "+prefix+"/auth:login", authNoLimit(authHandler.Login))
	s.mux.HandleFunc("OPTIONS "+prefix+"/auth:login", authNoLimit(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/auth:refresh", authNoLimit(authHandler.Refresh))
	s.mux.HandleFunc("OPTIONS "+prefix+"/auth:refresh", authNoLimit(s.corsPreflightHandler))

	// ==========================================
	// AUTHENTICATED ENDPOINTS (Any Role)
	// ==========================================

	// Logout requires authentication
	s.mux.HandleFunc("POST "+prefix+"/auth:logout", authenticated(authHandler.Logout))
	s.mux.HandleFunc("OPTIONS "+prefix+"/auth:logout", authenticated(s.corsPreflightHandler))

	// Me endpoints require authentication
	s.mux.HandleFunc("GET "+prefix+"/auth:me", authenticated(authHandler.GetMe))
	s.mux.HandleFunc("OPTIONS "+prefix+"/auth:me", authenticated(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/auth:me", authenticated(authHandler.UpdateMe))

	// Collections read endpoints (any authenticated user)
	s.mux.HandleFunc("GET "+prefix+"/collections:list", authenticated(collectionsHandler.List))
	s.mux.HandleFunc("OPTIONS "+prefix+"/collections:list", authenticated(s.corsPreflightHandler))
	s.mux.HandleFunc("GET "+prefix+"/collections:get", authenticated(collectionsHandler.Get))
	s.mux.HandleFunc("OPTIONS "+prefix+"/collections:get", authenticated(s.corsPreflightHandler))

	// Doc refresh requires authentication
	s.mux.HandleFunc("POST "+prefix+"/doc:refresh", authenticated(docHandler.RefreshCache))
	s.mux.HandleFunc("OPTIONS "+prefix+"/doc:refresh", authenticated(s.corsPreflightHandler))

	// ==========================================
	// ADMIN ONLY ENDPOINTS
	// ==========================================

	// User management endpoints (admin only)
	s.mux.HandleFunc("GET "+prefix+"/users:list", adminOnly(usersHandler.List))
	s.mux.HandleFunc("OPTIONS "+prefix+"/users:list", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("GET "+prefix+"/users:get", adminOnly(usersHandler.Get))
	s.mux.HandleFunc("OPTIONS "+prefix+"/users:get", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/users:create", adminOnly(usersHandler.Create))
	s.mux.HandleFunc("OPTIONS "+prefix+"/users:create", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/users:update", adminOnly(usersHandler.Update))
	s.mux.HandleFunc("OPTIONS "+prefix+"/users:update", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/users:destroy", adminOnly(usersHandler.Destroy))
	s.mux.HandleFunc("OPTIONS "+prefix+"/users:destroy", adminOnly(s.corsPreflightHandler))

	// API key management endpoints (admin only)
	s.mux.HandleFunc("GET "+prefix+"/apikeys:list", adminOnly(apiKeysHandler.List))
	s.mux.HandleFunc("OPTIONS "+prefix+"/apikeys:list", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("GET "+prefix+"/apikeys:get", adminOnly(apiKeysHandler.Get))
	s.mux.HandleFunc("OPTIONS "+prefix+"/apikeys:get", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/apikeys:create", adminOnly(apiKeysHandler.Create))
	s.mux.HandleFunc("OPTIONS "+prefix+"/apikeys:create", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/apikeys:update", adminOnly(apiKeysHandler.Update))
	s.mux.HandleFunc("OPTIONS "+prefix+"/apikeys:update", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/apikeys:destroy", adminOnly(apiKeysHandler.Destroy))
	s.mux.HandleFunc("OPTIONS "+prefix+"/apikeys:destroy", adminOnly(s.corsPreflightHandler))

	// Collections management endpoints (admin only)
	s.mux.HandleFunc("POST "+prefix+"/collections:create", adminOnly(collectionsHandler.Create))
	s.mux.HandleFunc("OPTIONS "+prefix+"/collections:create", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/collections:update", adminOnly(collectionsHandler.Update))
	s.mux.HandleFunc("OPTIONS "+prefix+"/collections:update", adminOnly(s.corsPreflightHandler))
	s.mux.HandleFunc("POST "+prefix+"/collections:destroy", adminOnly(collectionsHandler.Destroy))
	s.mux.HandleFunc("OPTIONS "+prefix+"/collections:destroy", adminOnly(s.corsPreflightHandler))

	// ==========================================
	// DYNAMIC DATA ENDPOINTS
	// ==========================================

	// Data access endpoints (dynamic collections with :action pattern)
	// This also serves as catch-all when prefix is empty
	if prefix == "" {
		s.mux.HandleFunc("/", s.loggingMiddleware(s.dynamicDataHandler(dataHandler, aggregationHandler, authenticated, writeRequired)))
	} else {
		s.mux.HandleFunc(prefix+"/", s.loggingMiddleware(s.dynamicDataHandler(dataHandler, aggregationHandler, authenticated, writeRequired)))
		// Catch-all for 404 when prefix is set
		s.mux.HandleFunc("/", s.loggingMiddleware(s.notFoundHandler))
	}
}

// loggingMiddleware logs HTTP requests and responses
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next(rw, r)

		// Log the request
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %s",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			duration,
		)
	}
}

// authMiddleware extracts and validates JWT or API key and sets the auth entity in context.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Try JWT authentication first (Authorization: Bearer <token>)
		authHeader := r.Header.Get(constants.HeaderAuthorization)
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == strings.ToLower(constants.AuthSchemeBearer) {
				token := strings.TrimSpace(parts[1])
				if token != "" {
					// Check if token is blacklisted
					blacklisted, err := s.tokenBlacklist.IsBlacklisted(ctx, token)
					if err != nil {
						log.Printf("Error checking token blacklist: %v", err)
						s.writeAuthError(w, http.StatusInternalServerError, "authentication error")
						return
					}
					if blacklisted {
						s.writeAuthError(w, http.StatusUnauthorized, "token has been revoked")
						return
					}

					claims, err := s.tokenService.ValidateAccessToken(token)
					if err == nil {
						// Valid JWT - create auth entity
						entity := &middleware.AuthEntity{
							ID:       claims.UserID,
							Type:     middleware.EntityTypeUser,
							Role:     claims.Role,
							CanWrite: claims.CanWrite,
							Username: claims.Username,
						}
						ctx = middleware.SetAuthEntity(ctx, entity)
						next(w, r.WithContext(ctx))
						return
					}
					// Invalid JWT token
					s.writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}
			}
		}

		// Try API key authentication (X-API-Key header)
		apiKey := r.Header.Get(s.config.APIKey.Header)
		if apiKey == "" && s.config.APIKey.Header != constants.HeaderAPIKey {
			apiKey = r.Header.Get(constants.HeaderAPIKey)
		}
		if apiKey != "" {
			keyHash := auth.HashAPIKey(apiKey)
			apiKeyObj, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
			if err == nil && apiKeyObj != nil {
				// Valid API key - create auth entity
				entity := &middleware.AuthEntity{
					ID:       apiKeyObj.ULID,
					Type:     middleware.EntityTypeAPIKey,
					Role:     apiKeyObj.Role,
					CanWrite: apiKeyObj.CanWrite,
				}
				ctx = middleware.SetAuthEntity(ctx, entity)

				// Update last used (non-blocking)
				go func(id int64) {
					if err := s.apiKeyRepo.UpdateLastUsed(context.Background(), id); err != nil {
						log.Printf("Failed to update API key last used: %v", err)
					}
				}(apiKeyObj.ID)

				next(w, r.WithContext(ctx))
				return
			}
			// Invalid API key
			s.writeAuthError(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		// No authentication provided
		s.writeAuthError(w, http.StatusUnauthorized, "authentication required")
	}
}

// writeAuthError writes an authentication error response.
func (s *Server) writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set(constants.HeaderContentType, constants.MIMEApplicationJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"error": message,
		"code":  statusCode,
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.server.Shutdown(ctx)
}

// Run starts the server and handles graceful shutdown
func (s *Server) Run() error {
	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- s.Start()
	}()

	// Listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Printf("Received signal: %v", sig)

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), constants.ShutdownTimeout)
		defer cancel()

		// Shutdown the server
		if err := s.Shutdown(ctx); err != nil {
			if err := s.server.Close(); err != nil {
				return fmt.Errorf("could not stop server gracefully: %w", err)
			}
		}
	}

	return nil
}

// Health check handler
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := "live"

	// Check database connection
	if err := s.db.Ping(ctx); err != nil {
		status = "down"
	}

	response := map[string]string{
		"status":  status,
		"name":    "moon",
		"version": s.version,
	}

	// Always return HTTP 200, even if service is down
	// Clients must check the "status" field to determine service health
	s.writeJSON(w, http.StatusOK, response)
}

// corsPreflightHandler handles CORS preflight OPTIONS requests.
// The CORS middleware wrapping this handler sets all necessary headers and returns 204.
// This handler exists solely to provide a route for OPTIONS requests.
func (s *Server) corsPreflightHandler(w http.ResponseWriter, r *http.Request) {
	// The CORS middleware handles everything for OPTIONS requests
	// This is just a no-op handler that allows the route to exist
}

// Root message handler - returns a friendly message at the root path
func (s *Server) rootMessageHandler(w http.ResponseWriter, r *http.Request) {
	// Only respond to exact root path
	if r.URL.Path != "/" {
		s.notFoundHandler(w, r)
		return
	}

	w.Header().Set(constants.HeaderContentType, constants.MIMETextPlain)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(config.RootMessage))
}

// Not found handler
func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "Endpoint not found")
}

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set(constants.HeaderContentType, constants.MIMEApplicationJSON)
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// writeError writes a JSON error response
func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	s.writeJSON(w, statusCode, map[string]any{
		"error": message,
		"code":  statusCode,
	})
}

// Data handler wrappers that extract collection name from URL path

func (s *Server) dynamicDataHandler(dataHandler *handlers.DataHandler, aggregationHandler *handlers.AggregationHandler, authenticated, writeRequired func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse path: {prefix}/{name}:{action}
		path := strings.TrimPrefix(r.URL.Path, s.config.Server.Prefix+"/")

		// Split by colon to get name and action
		parts := strings.SplitN(path, ":", 2)
		if len(parts) != 2 {
			s.writeError(w, http.StatusNotFound, "Endpoint not found")
			return
		}

		collectionName := parts[0]
		action := parts[1]

		// Prevent accessing collections endpoint
		if collectionName == "collections" {
			s.writeError(w, http.StatusNotFound, "Endpoint not found")
			return
		}

		// Skip reserved endpoints that are handled by other routes
		if collectionName == "auth" || collectionName == "users" || collectionName == "apikeys" || collectionName == "doc" {
			s.writeError(w, http.StatusNotFound, "Endpoint not found")
			return
		}

		// Route to appropriate handler based on action
		// Read operations: authenticated (any role)
		// Write operations: writeRequired (admin or user with can_write)
		switch action {
		case "list":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				dataHandler.List(w, r, collectionName)
			})(w, r)
		case "get":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				dataHandler.Get(w, r, collectionName)
			})(w, r)
		case "create":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			writeRequired(func(w http.ResponseWriter, r *http.Request) {
				dataHandler.Create(w, r, collectionName)
			})(w, r)
		case "update":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			writeRequired(func(w http.ResponseWriter, r *http.Request) {
				dataHandler.Update(w, r, collectionName)
			})(w, r)
		case "destroy":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			writeRequired(func(w http.ResponseWriter, r *http.Request) {
				dataHandler.Destroy(w, r, collectionName)
			})(w, r)
		case "count":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				aggregationHandler.Count(w, r, collectionName)
			})(w, r)
		case "sum":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				aggregationHandler.Sum(w, r, collectionName)
			})(w, r)
		case "avg":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				aggregationHandler.Avg(w, r, collectionName)
			})(w, r)
		case "min":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				aggregationHandler.Min(w, r, collectionName)
			})(w, r)
		case "max":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				aggregationHandler.Max(w, r, collectionName)
			})(w, r)
		case "schema":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			authenticated(func(w http.ResponseWriter, r *http.Request) {
				dataHandler.Schema(w, r, collectionName)
			})(w, r)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown action")
		}
	}
}
