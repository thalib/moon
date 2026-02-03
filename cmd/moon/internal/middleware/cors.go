package middleware

import (
	"fmt"
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
