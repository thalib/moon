package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	// APIKeyContextKey is the key for API key info in request context
	APIKeyContextKey ContextKey = "api_key_info"
)

// Permission represents allowed actions on resources
type Permission struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"` // "read", "write", "delete", "admin"
}

// APIKeyInfo holds information about an API key
type APIKeyInfo struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
	RateLimit   int          `json:"rate_limit"` // requests per minute (0 = unlimited)
}

// APIKeyStore manages API keys (in-memory implementation)
type APIKeyStore struct {
	keys map[string]*APIKeyInfo // hashed key -> info
	mu   sync.RWMutex
}

// NewAPIKeyStore creates a new API key store
func NewAPIKeyStore() *APIKeyStore {
	return &APIKeyStore{
		keys: make(map[string]*APIKeyInfo),
	}
}

// Add adds a new API key to the store (key should be hashed)
func (s *APIKeyStore) Add(hashedKey string, info *APIKeyInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[hashedKey] = info
}

// Get retrieves API key info by hashed key
func (s *APIKeyStore) Get(hashedKey string) (*APIKeyInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.keys[hashedKey]
	return info, ok
}

// Remove removes an API key from the store
func (s *APIKeyStore) Remove(hashedKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, hashedKey)
}

// HashAPIKey hashes an API key using SHA-256
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// APIKeyConfig holds API key middleware configuration
type APIKeyConfig struct {
	Enabled          bool
	HeaderName       string // Default: X-API-Key
	QueryParamName   string // Default: api_key
	Store            *APIKeyStore
	ProtectedPaths   []string
	UnprotectedPaths []string
	ProtectByDefault bool
	EndpointPerms    map[string][]string // path -> required permissions (e.g., "read", "write")
}

// APIKeyMiddleware provides API key authentication middleware
type APIKeyMiddleware struct {
	config APIKeyConfig
}

// NewAPIKeyMiddleware creates a new API key middleware instance
func NewAPIKeyMiddleware(config APIKeyConfig) *APIKeyMiddleware {
	if config.HeaderName == "" {
		config.HeaderName = "X-API-Key"
	}
	if config.QueryParamName == "" {
		config.QueryParamName = "api_key"
	}
	if config.Store == nil {
		config.Store = NewAPIKeyStore()
	}
	return &APIKeyMiddleware{
		config: config,
	}
}

// Authenticate is the main API key authentication middleware
func (m *APIKeyMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip if API key auth is disabled
		if !m.config.Enabled {
			next(w, r)
			return
		}

		path := r.URL.Path
		method := r.Method

		// Check if path is explicitly unprotected
		if m.isUnprotected(path) {
			next(w, r)
			return
		}

		// Check if path requires protection
		if !m.requiresAuth(path) {
			next(w, r)
			return
		}

		// Extract API key from header or query parameter
		apiKey := m.extractAPIKey(r)
		if apiKey == "" {
			m.logAuthFailure(r, "missing API key")
			m.writeAuthError(w, http.StatusUnauthorized, "API key is required")
			return
		}

		// Hash the key and look up in store
		hashedKey := HashAPIKey(apiKey)
		keyInfo, ok := m.config.Store.Get(hashedKey)
		if !ok {
			m.logAuthFailure(r, "invalid API key")
			m.writeAuthError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		// Check endpoint-specific permissions
		requiredPerms := m.getRequiredPermissions(path, method)
		if len(requiredPerms) > 0 {
			if !m.hasPermissions(keyInfo, path, requiredPerms) {
				m.logAuthFailure(r, "insufficient permissions")
				m.writeAuthError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}
		}

		// Log successful API key usage for audit trail
		m.logAPIKeyUsage(r, keyInfo)

		// Add key info to context
		ctx := context.WithValue(r.Context(), APIKeyContextKey, keyInfo)
		next(w, r.WithContext(ctx))
	}
}

// isUnprotected checks if a path is explicitly unprotected
func (m *APIKeyMiddleware) isUnprotected(path string) bool {
	for _, unprotected := range m.config.UnprotectedPaths {
		if m.pathMatches(path, unprotected) {
			return true
		}
	}
	return false
}

// requiresAuth checks if a path requires authentication
func (m *APIKeyMiddleware) requiresAuth(path string) bool {
	if m.config.ProtectByDefault {
		return !m.isUnprotected(path)
	}

	for _, protected := range m.config.ProtectedPaths {
		if m.pathMatches(path, protected) {
			return true
		}
	}
	return false
}

// pathMatches checks if a path matches a pattern (supports prefix matching with *)
func (m *APIKeyMiddleware) pathMatches(path, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return path == pattern
}

// extractAPIKey extracts the API key from header or query parameter
func (m *APIKeyMiddleware) extractAPIKey(r *http.Request) string {
	// First, try header
	apiKey := r.Header.Get(m.config.HeaderName)
	if apiKey != "" {
		return apiKey
	}

	// Then, try query parameter
	apiKey = r.URL.Query().Get(m.config.QueryParamName)
	return apiKey
}

// getRequiredPermissions returns the required permissions for a path and method
func (m *APIKeyMiddleware) getRequiredPermissions(path, method string) []string {
	// Check explicit endpoint permissions first
	if perms, ok := m.config.EndpointPerms[path]; ok {
		return perms
	}

	// Default permission mapping based on HTTP method
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return []string{"read"}
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return []string{"write"}
	case http.MethodDelete:
		return []string{"delete"}
	default:
		return nil
	}
}

// hasPermissions checks if an API key has the required permissions for a resource
func (m *APIKeyMiddleware) hasPermissions(keyInfo *APIKeyInfo, path string, requiredPerms []string) bool {
	// Check for admin permission which grants all access
	for _, perm := range keyInfo.Permissions {
		for _, action := range perm.Actions {
			if action == "admin" {
				return true
			}
		}
	}

	// Check each required permission
	for _, required := range requiredPerms {
		hasPermission := false
		for _, perm := range keyInfo.Permissions {
			// Check if permission applies to this resource/path
			if m.permissionMatchesPath(perm.Resource, path) {
				for _, action := range perm.Actions {
					if action == required || action == "admin" {
						hasPermission = true
						break
					}
				}
			}
			if hasPermission {
				break
			}
		}
		if !hasPermission {
			return false
		}
	}
	return true
}

// permissionMatchesPath checks if a permission resource matches a path
func (m *APIKeyMiddleware) permissionMatchesPath(resource, path string) bool {
	// Wildcard matches all
	if resource == "*" {
		return true
	}
	return m.pathMatches(path, resource)
}

// logAuthFailure logs authentication failures for security monitoring
func (m *APIKeyMiddleware) logAuthFailure(r *http.Request, reason string) {
	log.Printf("APIKEY_AUTH_FAILURE: %s %s - %s", r.Method, r.URL.Path, reason)
}

// logAPIKeyUsage logs API key usage for audit trail
func (m *APIKeyMiddleware) logAPIKeyUsage(r *http.Request, keyInfo *APIKeyInfo) {
	log.Printf("APIKEY_USAGE: %s %s - key_id=%s key_name=%s", r.Method, r.URL.Path, keyInfo.ID, keyInfo.Name)
}

// writeAuthError writes an authentication error response
func (m *APIKeyMiddleware) writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"error": message,
		"code":  statusCode,
	})
}

// GetAPIKeyInfo extracts API key info from request context
func GetAPIKeyInfo(ctx context.Context) (*APIKeyInfo, bool) {
	info, ok := ctx.Value(APIKeyContextKey).(*APIKeyInfo)
	return info, ok
}

// ValidateAPIKeyFormat validates that an API key has a proper format
// Keys should be at least 40 characters (recommended: 64) for sufficient entropy
func ValidateAPIKeyFormat(key string) bool {
	// API keys should be at least 40 characters for strong security
	// (provides ~240 bits of entropy with base62 encoding)
	if len(key) < 40 {
		return false
	}
	for _, c := range key {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// CompareAPIKeys compares two API keys in constant time to prevent timing attacks
func CompareAPIKeys(key1, key2 string) bool {
	return subtle.ConstantTimeCompare([]byte(key1), []byte(key2)) == 1
}
