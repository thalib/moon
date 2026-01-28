// Package middleware provides HTTP middleware for authentication, authorization,
// request logging, and error handling. It supports both JWT and API key authentication
// as specified in SPEC.md configuration.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey type for context keys
type ContextKey string

const (
	// UserClaimsKey is the key for user claims in request context
	UserClaimsKey ContextKey = "user_claims"
)

// UserClaims represents the claims extracted from JWT token
type UserClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTConfig holds JWT middleware configuration
type JWTConfig struct {
	Secret            string
	Expiration        time.Duration
	ProtectedPaths    []string
	UnprotectedPaths  []string
	RequiredRoles     map[string][]string // path -> required roles
	ProtectByDefault  bool
}

// JWTMiddleware provides JWT authentication middleware
type JWTMiddleware struct {
	config JWTConfig
}

// NewJWTMiddleware creates a new JWT middleware instance
func NewJWTMiddleware(config JWTConfig) *JWTMiddleware {
	return &JWTMiddleware{
		config: config,
	}
}

// Authenticate is the main JWT authentication middleware
func (m *JWTMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

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

		// Extract token from Authorization header
		token, err := m.extractToken(r)
		if err != nil {
			m.logAuthFailure(r, "missing or invalid authorization header", err)
			m.writeAuthError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
			return
		}

		// Validate token
		claims, err := m.validateToken(token)
		if err != nil {
			m.logAuthFailure(r, "invalid token", err)
			m.writeAuthError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Check role-based permissions
		if requiredRoles, ok := m.config.RequiredRoles[path]; ok && len(requiredRoles) > 0 {
			if !m.hasRequiredRoles(claims.Roles, requiredRoles) {
				m.logAuthFailure(r, "insufficient permissions", nil)
				m.writeAuthError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// isUnprotected checks if a path is explicitly unprotected
func (m *JWTMiddleware) isUnprotected(path string) bool {
	for _, unprotected := range m.config.UnprotectedPaths {
		if m.pathMatches(path, unprotected) {
			return true
		}
	}
	return false
}

// requiresAuth checks if a path requires authentication
func (m *JWTMiddleware) requiresAuth(path string) bool {
	// If protect by default, require auth unless explicitly unprotected
	if m.config.ProtectByDefault {
		return !m.isUnprotected(path)
	}

	// Otherwise, only require auth for explicitly protected paths
	for _, protected := range m.config.ProtectedPaths {
		if m.pathMatches(path, protected) {
			return true
		}
	}
	return false
}

// pathMatches checks if a path matches a pattern (supports prefix matching with *)
func (m *JWTMiddleware) pathMatches(path, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return path == pattern
}

// extractToken extracts the JWT token from the Authorization header
func (m *JWTMiddleware) extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("authorization header must be in 'Bearer <token>' format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}

// validateToken validates the JWT token and returns the claims
func (m *JWTMiddleware) validateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// hasRequiredRoles checks if user has any of the required roles
func (m *JWTMiddleware) hasRequiredRoles(userRoles, requiredRoles []string) bool {
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
func (m *JWTMiddleware) logAuthFailure(r *http.Request, reason string, err error) {
	if err != nil {
		log.Printf("AUTH_FAILURE: %s %s - %s: %v", r.Method, r.URL.Path, reason, err)
	} else {
		log.Printf("AUTH_FAILURE: %s %s - %s", r.Method, r.URL.Path, reason)
	}
}

// writeAuthError writes an authentication error response
func (m *JWTMiddleware) writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"error": message,
		"code":  statusCode,
	})
}

// GenerateToken generates a new JWT token with the given claims
func GenerateToken(secret string, userID string, roles []string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := &UserClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)), // Allow 30s clock skew
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GetUserClaims extracts user claims from request context
func GetUserClaims(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*UserClaims)
	return claims, ok
}
