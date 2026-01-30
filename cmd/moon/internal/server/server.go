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

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/handlers"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// Server represents the HTTP server
type Server struct {
	config   *config.AppConfig
	db       database.Driver
	registry *registry.SchemaRegistry
	mux      *http.ServeMux
	server   *http.Server
	version  string
}

// New creates a new server instance
func New(cfg *config.AppConfig, db database.Driver, reg *registry.SchemaRegistry, version string) *Server {
	mux := http.NewServeMux()

	srv := &Server{
		config:   cfg,
		db:       db,
		registry: reg,
		mux:      mux,
		version:  version,
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

	// Get the prefix from config
	prefix := s.config.Server.Prefix

	// Health check endpoint (always at /health, respects prefix)
	healthPath := prefix + "/health"
	s.mux.HandleFunc("GET "+healthPath, s.loggingMiddleware(s.healthHandler))

	// Schema management endpoints (collections)
	s.mux.HandleFunc("GET "+prefix+"/collections:list", s.loggingMiddleware(collectionsHandler.List))
	s.mux.HandleFunc("GET "+prefix+"/collections:get", s.loggingMiddleware(collectionsHandler.Get))
	s.mux.HandleFunc("POST "+prefix+"/collections:create", s.loggingMiddleware(collectionsHandler.Create))
	s.mux.HandleFunc("POST "+prefix+"/collections:update", s.loggingMiddleware(collectionsHandler.Update))
	s.mux.HandleFunc("POST "+prefix+"/collections:destroy", s.loggingMiddleware(collectionsHandler.Destroy))

	// Data access endpoints (dynamic collections with :action pattern)
	// This also serves as catch-all when prefix is empty
	if prefix == "" {
		s.mux.HandleFunc("/", s.loggingMiddleware(s.dynamicDataHandler(dataHandler, aggregationHandler)))
	} else {
		s.mux.HandleFunc(prefix+"/", s.loggingMiddleware(s.dynamicDataHandler(dataHandler, aggregationHandler)))
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

func (s *Server) dynamicDataHandler(dataHandler *handlers.DataHandler, aggregationHandler *handlers.AggregationHandler) http.HandlerFunc {
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

		// Route to appropriate handler based on action
		switch action {
		case "list":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			dataHandler.List(w, r, collectionName)
		case "get":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			dataHandler.Get(w, r, collectionName)
		case "create":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			dataHandler.Create(w, r, collectionName)
		case "update":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			dataHandler.Update(w, r, collectionName)
		case "destroy":
			if r.Method != http.MethodPost {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			dataHandler.Destroy(w, r, collectionName)
		case "count":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			aggregationHandler.Count(w, r, collectionName)
		case "sum":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			aggregationHandler.Sum(w, r, collectionName)
		case "avg":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			aggregationHandler.Avg(w, r, collectionName)
		case "min":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			aggregationHandler.Min(w, r, collectionName)
		case "max":
			if r.Method != http.MethodGet {
				s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			aggregationHandler.Max(w, r, collectionName)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown action")
		}
	}
}
