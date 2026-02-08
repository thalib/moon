package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/constants"
)

// UnifiedAuthConfig holds configuration for unified authentication middleware
type UnifiedAuthConfig struct {
	JWTMiddleware    *JWTMiddleware
	APIKeyMiddleware *APIKeyMiddleware
	ProtectedPaths   []string
	UnprotectedPaths []string
	ProtectByDefault bool
	CORSMiddleware   *CORSMiddleware // Used to check for auth bypass (PRD-058)
}

// UnifiedAuthMiddleware provides unified authentication supporting both JWT and API keys
// via the Authorization: Bearer header
type UnifiedAuthMiddleware struct {
	config UnifiedAuthConfig
}

// NewUnifiedAuthMiddleware creates a new unified authentication middleware
func NewUnifiedAuthMiddleware(config UnifiedAuthConfig) *UnifiedAuthMiddleware {
	return &UnifiedAuthMiddleware{
		config: config,
	}
}

// ShouldBypassAuth checks if the request path should bypass authentication (PRD-058)
func (m *UnifiedAuthMiddleware) ShouldBypassAuth(path string) bool {
	// Check CORS endpoint registration
	if m.config.CORSMiddleware != nil {
		endpointConfig := m.config.CORSMiddleware.MatchEndpoint(path)
		if endpointConfig != nil && endpointConfig.BypassAuth {
			log.Printf("INFO: Authentication bypassed for %s (CORS endpoint configuration)", path)
			return true
		}
	}
	return false
}

// Authenticate is the main unified authentication middleware
func (m *UnifiedAuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check if this endpoint should bypass authentication (PRD-058)
		if m.ShouldBypassAuth(path) {
			next(w, r)
			return
		}

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

		// Extract token from Authorization: Bearer header
		token, err := m.extractBearerToken(r)

		// If no token, return authentication required error
		if err != nil || token == "" {
			m.writeAuthError(w, http.StatusUnauthorized, "authentication_required",
				"Authorization header required. Use: Authorization: Bearer <token>")
			return
		}

		// Detect token type and route to appropriate handler
		if m.isAPIKey(token) {
			// Route to API key authentication
			if m.config.APIKeyMiddleware == nil {
				m.writeAuthError(w, http.StatusUnauthorized, "invalid_credentials",
					"API key authentication is not enabled")
				return
			}

			// Validate API key
			hashedKey := HashAPIKey(token)
			keyInfo, ok := m.config.APIKeyMiddleware.config.Store.Get(hashedKey)
			if !ok {
				m.logAuthFailure(r, "invalid API key", nil)
				m.writeAuthError(w, http.StatusUnauthorized, "invalid_credentials",
					"Token is invalid, expired, or has been revoked")
				return
			}

			// Check permissions
			method := r.Method
			requiredPerms := m.getRequiredPermissions(path, method)
			if len(requiredPerms) > 0 {
				if !m.hasPermissions(keyInfo, path, requiredPerms) {
					m.logAuthFailure(r, "insufficient permissions", nil)
					m.writeAuthError(w, http.StatusForbidden, "forbidden",
						"Insufficient permissions for this operation")
					return
				}
			}

			// Log API key usage
			log.Printf("APIKEY_USAGE: %s %s - key_id=%s key_name=%s", r.Method, r.URL.Path, keyInfo.ID, keyInfo.Name)

			// Add key info to context
			ctx := context.WithValue(r.Context(), APIKeyContextKey, keyInfo)
			next(w, r.WithContext(ctx))

		} else if m.isJWT(token) {
			// Route to JWT authentication
			if m.config.JWTMiddleware == nil {
				m.writeAuthError(w, http.StatusUnauthorized, "invalid_credentials",
					"JWT authentication is not enabled")
				return
			}

			// Check if token is blacklisted
			if m.config.JWTMiddleware.config.TokenBlacklist != nil {
				blacklisted, err := m.config.JWTMiddleware.config.TokenBlacklist.IsBlacklisted(r.Context(), token)
				if err != nil {
					m.logAuthFailure(r, "error checking token blacklist", err)
					m.writeAuthError(w, http.StatusInternalServerError, "authentication_error",
						"Authentication error")
					return
				}
				if blacklisted {
					m.logAuthFailure(r, "token has been revoked", nil)
					m.writeAuthError(w, http.StatusUnauthorized, "invalid_credentials",
						"Token is invalid, expired, or has been revoked")
					return
				}
			}

			// Validate JWT
			claims, err := m.config.JWTMiddleware.validateToken(token)
			if err != nil {
				m.logAuthFailure(r, "invalid token", err)
				m.writeAuthError(w, http.StatusUnauthorized, "invalid_credentials",
					"Token is invalid, expired, or has been revoked")
				return
			}

			// Check role-based permissions
			if requiredRoles, ok := m.config.JWTMiddleware.config.RequiredRoles[path]; ok && len(requiredRoles) > 0 {
				if !m.hasRequiredRoles(claims.Roles, requiredRoles) {
					m.logAuthFailure(r, "insufficient permissions", nil)
					m.writeAuthError(w, http.StatusForbidden, "forbidden",
						"Insufficient permissions for this operation")
					return
				}
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next(w, r.WithContext(ctx))

		} else {
			// Invalid token format
			m.logAuthFailure(r, "invalid token format", nil)
			m.writeAuthError(w, http.StatusUnauthorized, "invalid_token_format",
				"Token must be a valid JWT or API key (starting with moon_live_)")
			return
		}
	}
}

// extractBearerToken extracts the token from Authorization: Bearer header
func (m *UnifiedAuthMiddleware) extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get(constants.HeaderAuthorization)
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != strings.ToLower(constants.AuthSchemeBearer) {
		return "", fmt.Errorf("authorization header must be in '%s <token>' format", constants.AuthSchemeBearer)
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}

// isAPIKey checks if a token is an API key based on prefix
func (m *UnifiedAuthMiddleware) isAPIKey(token string) bool {
	return strings.HasPrefix(token, constants.APIKeyPrefix)
}

// isJWT checks if a token matches JWT format (3 base64 segments separated by dots)
func (m *UnifiedAuthMiddleware) isJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3
}

// isUnprotected checks if a path is explicitly unprotected
func (m *UnifiedAuthMiddleware) isUnprotected(path string) bool {
	for _, unprotected := range m.config.UnprotectedPaths {
		if m.pathMatches(path, unprotected) {
			return true
		}
	}
	return false
}

// requiresAuth checks if a path requires authentication
func (m *UnifiedAuthMiddleware) requiresAuth(path string) bool {
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
func (m *UnifiedAuthMiddleware) pathMatches(path, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return path == pattern
}

// getRequiredPermissions returns the required permissions for a path and method
func (m *UnifiedAuthMiddleware) getRequiredPermissions(path, method string) []string {
	if m.config.APIKeyMiddleware == nil {
		return nil
	}

	// Check explicit endpoint permissions first
	if perms, ok := m.config.APIKeyMiddleware.config.EndpointPerms[path]; ok {
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
func (m *UnifiedAuthMiddleware) hasPermissions(keyInfo *APIKeyInfo, path string, requiredPerms []string) bool {
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
func (m *UnifiedAuthMiddleware) permissionMatchesPath(resource, path string) bool {
	// Wildcard matches all
	if resource == "*" {
		return true
	}
	return m.pathMatches(path, resource)
}

// hasRequiredRoles checks if user has any of the required roles
func (m *UnifiedAuthMiddleware) hasRequiredRoles(userRoles, requiredRoles []string) bool {
	roleSet := make(map[string]bool)
	for _, role := range userRoles {
		roleSet[role] = true
	}

	for _, required := range requiredRoles {
		if roleSet[required] {
			return true
		}
	}
	return false
}

// logAuthFailure logs authentication failures for security monitoring
func (m *UnifiedAuthMiddleware) logAuthFailure(r *http.Request, reason string, err error) {
	if err != nil {
		log.Printf("AUTH_FAILURE: %s %s - %s: %v", r.Method, r.URL.Path, reason, err)
	} else {
		log.Printf("AUTH_FAILURE: %s %s - %s", r.Method, r.URL.Path, reason)
	}
}

// writeAuthError writes a standardized authentication error response
func (m *UnifiedAuthMiddleware) writeAuthError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set(constants.HeaderContentType, constants.MIMEApplicationJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"error":   errorType,
		"message": message,
		"status":  statusCode,
	})
}
