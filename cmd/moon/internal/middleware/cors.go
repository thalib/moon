package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string // Headers exposed to browser (PRD-049)
	AllowCredentials bool
	MaxAge           int
	Endpoints        []CORSEndpointConfig // Endpoint-specific CORS registration (PRD-058)
}

// CORSEndpointConfig represents a single CORS endpoint registration
type CORSEndpointConfig struct {
	Path             string
	PatternType      string
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	BypassAuth       bool
}

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS)
type CORSMiddleware struct {
	config CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware instance
func NewCORSMiddleware(config CORSConfig) *CORSMiddleware {
	// Set default exposed headers if not specified (PRD-049)
	if len(config.ExposedHeaders) == 0 {
		config.ExposedHeaders = []string{
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"X-Request-ID",
		}
	}
	return &CORSMiddleware{config: config}
}

// Handle adds CORS headers to HTTP responses
func (m *CORSMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If CORS is disabled, pass through without adding headers
		if !m.config.Enabled {
			next(w, r)
			return
		}

		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && m.isOriginAllowed(origin) {
			// Set Access-Control-Allow-Origin
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// Set Access-Control-Allow-Credentials
			if m.config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Set Access-Control-Expose-Headers (PRD-049)
			if len(m.config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(m.config.ExposedHeaders, ", "))
			}

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				// Set Access-Control-Allow-Methods
				if len(m.config.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.config.AllowedMethods, ", "))
				}

				// Set Access-Control-Allow-Headers
				if len(m.config.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowedHeaders, ", "))
				}

				// Set Access-Control-Max-Age
				if m.config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", m.config.MaxAge))
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next(w, r)
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func (m *CORSMiddleware) isOriginAllowed(origin string) bool {
	// If no origins specified, allow none
	if len(m.config.AllowedOrigins) == 0 {
		return false
	}

	// Check for wildcard
	for _, allowed := range m.config.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}

	return false
}

// HandlePublic adds public CORS headers (Access-Control-Allow-Origin: *) for public endpoints (PRD-052)
// This is used for health checks, documentation, and other non-sensitive endpoints
func (m *CORSMiddleware) HandlePublic(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Always set Access-Control-Allow-Origin to * for public endpoints
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Set Access-Control-Allow-Methods for GET requests
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			// Set Access-Control-Allow-Headers
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			// Set Access-Control-Max-Age
			if m.config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", m.config.MaxAge))
			} else {
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// MatchEndpoint checks if a request path matches any registered CORS endpoint
// Returns the matching endpoint configuration, or nil if no match
func (m *CORSMiddleware) MatchEndpoint(path string) *CORSEndpointConfig {
	var bestMatch *CORSEndpointConfig
	var bestMatchScore int

	for i := range m.config.Endpoints {
		endpoint := &m.config.Endpoints[i]
		if matches, score := endpoint.Matches(path); matches {
			if score > bestMatchScore {
				bestMatch = endpoint
				bestMatchScore = score
			}
		}
	}

	if bestMatch != nil {
		log.Printf("DEBUG: CORS endpoint match: path=%s, pattern=%s (%s), score=%d, bypass_auth=%t",
			path, bestMatch.Path, bestMatch.PatternType, bestMatchScore, bestMatch.BypassAuth)
	}

	return bestMatch
}

// Matches checks if this endpoint config matches the given path
// Returns (matched bool, score int) where higher scores indicate more specific matches
func (e *CORSEndpointConfig) Matches(path string) (bool, int) {
	switch e.PatternType {
	case "exact":
		if path == e.Path {
			return true, 1000 + len(e.Path) // Highest priority
		}
	case "prefix":
		cleanPattern := strings.TrimSuffix(e.Path, "/*")
		if strings.HasPrefix(path, cleanPattern) {
			return true, 500 + len(cleanPattern) // Priority by prefix length
		}
	case "suffix":
		cleanPattern := strings.TrimPrefix(e.Path, "*")
		if strings.HasSuffix(path, cleanPattern) {
			return true, 300 + len(cleanPattern) // Priority by suffix length
		}
	case "contains":
		cleanPattern := strings.Trim(e.Path, "*")
		if strings.Contains(path, cleanPattern) {
			return true, 100 + len(cleanPattern) // Lowest priority
		}
	}
	return false, 0
}

// HandleDynamic applies CORS based on endpoint registration or falls back to global config (PRD-058)
func (m *CORSMiddleware) HandleDynamic(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if request path matches a registered CORS endpoint
		endpointConfig := m.MatchEndpoint(r.URL.Path)

		if endpointConfig != nil {
			// Apply endpoint-specific CORS
			m.applyCORSHeaders(w, r, endpointConfig.AllowedOrigins,
				endpointConfig.AllowedMethods,
				endpointConfig.AllowedHeaders,
				endpointConfig.AllowCredentials)

			// Handle OPTIONS preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next(w, r)
			return
		}

		// Fall back to standard CORS handler
		m.Handle(next)(w, r)
	}
}

// applyCORSHeaders is a helper to apply CORS headers based on configuration
func (m *CORSMiddleware) applyCORSHeaders(w http.ResponseWriter, r *http.Request,
	origins, methods, headers []string,
	allowCredentials bool) {
	origin := r.Header.Get("Origin")

	// Check if origin is allowed
	if origin != "" && m.isOriginAllowedFor(origin, origins) {
		// For wildcard, use "*" directly
		if len(origins) == 1 && origins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if allowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if len(methods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		}

		if len(headers) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
		}

		if m.config.MaxAge > 0 {
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", m.config.MaxAge))
		}

		// Set exposed headers (PRD-049)
		if len(m.config.ExposedHeaders) > 0 {
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(m.config.ExposedHeaders, ", "))
		}
	}
}

// isOriginAllowedFor checks if origin is in the specific allowed list
func (m *CORSMiddleware) isOriginAllowedFor(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	return false
}
