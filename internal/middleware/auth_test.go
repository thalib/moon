package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-testing"

func TestNewJWTMiddleware(t *testing.T) {
	config := JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	}

	middleware := NewJWTMiddleware(config)

	if middleware == nil {
		t.Fatal("NewJWTMiddleware returned nil")
	}

	if middleware.config.Secret != testSecret {
		t.Errorf("Expected secret %s, got %s", testSecret, middleware.config.Secret)
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(testSecret, "user123", []string{"admin", "user"}, time.Hour)

	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}
}

func TestValidateToken_ValidToken(t *testing.T) {
	config := JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	}

	middleware := NewJWTMiddleware(config)

	// Generate a valid token
	token, err := GenerateToken(testSecret, "user123", []string{"admin"}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := middleware.validateToken(token)

	if err != nil {
		t.Fatalf("validateToken failed for valid token: %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", claims.UserID)
	}

	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Errorf("Expected roles ['admin'], got %v", claims.Roles)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	}

	middleware := NewJWTMiddleware(config)

	// Generate an expired token
	token, err := GenerateToken(testSecret, "user123", []string{"admin"}, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	_, err = middleware.validateToken(token)

	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	config := JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	}

	middleware := NewJWTMiddleware(config)

	// Generate a token with a different secret
	token, err := GenerateToken("different-secret", "user123", []string{"admin"}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	_, err = middleware.validateToken(token)

	if err == nil {
		t.Error("Expected error for invalid signature, got nil")
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	config := JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	}

	middleware := NewJWTMiddleware(config)

	_, err := middleware.validateToken("not.a.valid.jwt.token")

	if err == nil {
		t.Error("Expected error for invalid token format, got nil")
	}
}

func TestExtractToken_ValidHeader(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{Secret: testSecret})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")

	token, err := middleware.extractToken(req)

	if err != nil {
		t.Fatalf("extractToken failed: %v", err)
	}

	if token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", token)
	}
}

func TestExtractToken_MissingHeader(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{Secret: testSecret})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := middleware.extractToken(req)

	if err == nil {
		t.Error("Expected error for missing header, got nil")
	}
}

func TestExtractToken_InvalidFormat(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{Secret: testSecret})

	testCases := []struct {
		name   string
		header string
	}{
		{"No bearer prefix", "test-token-123"},
		{"Wrong prefix", "Basic test-token-123"},
		{"Empty token", "Bearer "},
		{"Bearer only", "Bearer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tc.header)

			_, err := middleware.extractToken(req)

			if err == nil {
				t.Errorf("Expected error for '%s', got nil", tc.header)
			}
		})
	}
}

func TestAuthenticate_UnprotectedPath(t *testing.T) {
	config := JWTConfig{
		Secret:           testSecret,
		UnprotectedPaths: []string{"/health", "/public/*"},
		ProtectByDefault: true,
	}

	middleware := NewJWTMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	testCases := []struct {
		name string
		path string
	}{
		{"Exact match", "/health"},
		{"Prefix match", "/public/page"},
		{"Prefix match nested", "/public/dir/page"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handlerCalled = false
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			middleware.Authenticate(handler)(w, req)

			if !handlerCalled {
				t.Error("Handler was not called for unprotected path")
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestAuthenticate_ProtectedPath_NoToken(t *testing.T) {
	config := JWTConfig{
		Secret:           testSecret,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewJWTMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if handlerCalled {
		t.Error("Handler should not be called without token")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthenticate_ProtectedPath_ValidToken(t *testing.T) {
	config := JWTConfig{
		Secret:           testSecret,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewJWTMiddleware(config)

	token, _ := GenerateToken(testSecret, "user123", []string{"admin"}, time.Hour)

	var capturedClaims *UserClaims
	handler := func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetUserClaims(r.Context())
		if ok {
			capturedClaims = claims
		}
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if capturedClaims == nil {
		t.Fatal("Claims were not passed to handler")
	}

	if capturedClaims.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", capturedClaims.UserID)
	}
}

func TestAuthenticate_ProtectedPath_InvalidToken(t *testing.T) {
	config := JWTConfig{
		Secret:           testSecret,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewJWTMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if handlerCalled {
		t.Error("Handler should not be called with invalid token")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthenticate_RoleBasedAccess(t *testing.T) {
	config := JWTConfig{
		Secret:           testSecret,
		ProtectedPaths:   []string{"/admin/*"},
		ProtectByDefault: false,
		RequiredRoles: map[string][]string{
			"/admin/users": {"admin", "superadmin"},
		},
	}

	middleware := NewJWTMiddleware(config)

	t.Run("User with required role", func(t *testing.T) {
		token, _ := GenerateToken(testSecret, "user123", []string{"admin"}, time.Hour)

		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}

		middleware.Authenticate(handler)(w, req)

		if !handlerCalled {
			t.Error("Handler should be called for user with required role")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("User without required role", func(t *testing.T) {
		token, _ := GenerateToken(testSecret, "user123", []string{"user"}, time.Hour)

		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		}

		middleware.Authenticate(handler)(w, req)

		if handlerCalled {
			t.Error("Handler should not be called for user without required role")
		}

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})
}

func TestHasRequiredRoles(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{Secret: testSecret})

	testCases := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expected      bool
	}{
		{
			name:          "Has one required role",
			userRoles:     []string{"admin", "user"},
			requiredRoles: []string{"admin"},
			expected:      true,
		},
		{
			name:          "Has multiple required roles",
			userRoles:     []string{"admin"},
			requiredRoles: []string{"admin", "superadmin"},
			expected:      true,
		},
		{
			name:          "Missing required role",
			userRoles:     []string{"user"},
			requiredRoles: []string{"admin"},
			expected:      false,
		},
		{
			name:          "Empty user roles",
			userRoles:     []string{},
			requiredRoles: []string{"admin"},
			expected:      false,
		},
		{
			name:          "Empty required roles",
			userRoles:     []string{"admin"},
			requiredRoles: []string{},
			expected:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := middleware.hasRequiredRoles(tc.userRoles, tc.requiredRoles)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestPathMatches(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{Secret: testSecret})

	testCases := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "Exact match",
			path:     "/health",
			pattern:  "/health",
			expected: true,
		},
		{
			name:     "Prefix match with wildcard",
			path:     "/api/users",
			pattern:  "/api/*",
			expected: true,
		},
		{
			name:     "No match",
			path:     "/health",
			pattern:  "/api/*",
			expected: false,
		},
		{
			name:     "Nested prefix match",
			path:     "/api/v1/users/123",
			pattern:  "/api/*",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := middleware.pathMatches(tc.path, tc.pattern)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestGetUserClaims(t *testing.T) {
	t.Run("Claims present in context", func(t *testing.T) {
		claims := &UserClaims{
			UserID: "user123",
			Roles:  []string{"admin"},
		}

		ctx := context.WithValue(context.Background(), UserClaimsKey, claims)

		retrieved, ok := GetUserClaims(ctx)

		if !ok {
			t.Fatal("GetUserClaims returned false for context with claims")
		}

		if retrieved.UserID != "user123" {
			t.Errorf("Expected user ID 'user123', got '%s'", retrieved.UserID)
		}
	})

	t.Run("Claims not present in context", func(t *testing.T) {
		ctx := context.Background()

		_, ok := GetUserClaims(ctx)

		if ok {
			t.Error("GetUserClaims returned true for context without claims")
		}
	})
}

func TestWriteAuthError(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{Secret: testSecret})

	w := httptest.NewRecorder()
	middleware.writeAuthError(w, http.StatusUnauthorized, "Test error")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Test error" {
		t.Errorf("Expected error 'Test error', got '%v'", response["error"])
	}

	if response["code"].(float64) != float64(http.StatusUnauthorized) {
		t.Errorf("Expected code %d, got %v", http.StatusUnauthorized, response["code"])
	}
}

func TestUserClaims_JWTClaims(t *testing.T) {
	// Test that UserClaims properly implements jwt.Claims interface
	now := time.Now()
	claims := &UserClaims{
		UserID: "user123",
		Roles:  []string{"admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create a token with these claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))

	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// Parse the token back
	parsedToken, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	parsedClaims, ok := parsedToken.Claims.(*UserClaims)
	if !ok {
		t.Fatal("Failed to cast claims to UserClaims")
	}

	if parsedClaims.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", parsedClaims.UserID)
	}

	if len(parsedClaims.Roles) != 1 || parsedClaims.Roles[0] != "admin" {
		t.Errorf("Expected roles ['admin'], got %v", parsedClaims.Roles)
	}
}
