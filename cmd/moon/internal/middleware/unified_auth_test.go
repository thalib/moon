package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/constants"
)

func TestUnifiedAuth_JWTViaAuthorizationBearer(t *testing.T) {
	// Setup JWT middleware
	jwtMiddleware := NewJWTMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	// Setup unified auth middleware
	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		JWTMiddleware:    jwtMiddleware,
		ProtectByDefault: true,
		UnprotectedPaths: []string{"/health"},
	})

	// Generate a valid JWT token
	token, err := GenerateToken(testSecret, "user123", []string{"admin"}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create test handler
	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create request with Authorization: Bearer header
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer "+token)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestUnifiedAuth_APIKeyViaAuthorizationBearer(t *testing.T) {
	// Setup API key store
	store := NewAPIKeyStore()
	apiKey := constants.APIKeyPrefix + "test1234567890123456789012345678901234567890123456789012345678"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "key1",
		Name: "Test Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read", "write"}},
		},
	})

	// Setup API key middleware
	apiKeyMiddleware := NewAPIKeyMiddleware(APIKeyConfig{
		Enabled: true,
		Store:   store,
	})

	// Setup unified auth middleware
	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		APIKeyMiddleware: apiKeyMiddleware,
		ProtectByDefault: true,
		UnprotectedPaths: []string{"/health"},
	})

	// Create test handler
	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create request with Authorization: Bearer header and API key
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer "+apiKey)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestUnifiedAuth_LegacyAPIKeyHeader(t *testing.T) {
	// Setup API key store
	store := NewAPIKeyStore()
	apiKey := constants.APIKeyPrefix + "test1234567890123456789012345678901234567890123456789012345678"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "key1",
		Name: "Test Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	})

	// Setup API key middleware
	apiKeyMiddleware := NewAPIKeyMiddleware(APIKeyConfig{
		Enabled: true,
		Store:   store,
	})

	// Setup unified auth middleware with legacy support enabled
	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		APIKeyMiddleware:       apiKeyMiddleware,
		LegacyHeaderSupport:    true,
		LegacyHeaderSunsetDate: "2026-05-08T00:00:00Z",
		ProtectByDefault:       true,
		UnprotectedPaths:       []string{"/health"},
	})

	// Create test handler
	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create request with legacy X-API-Key header
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set(constants.HeaderAPIKey, apiKey)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Check for deprecation headers
	if w.Header().Get(constants.HeaderDeprecation) != "true" {
		t.Error("Expected Deprecation header to be 'true'")
	}

	if w.Header().Get(constants.HeaderSunset) == "" {
		t.Error("Expected Sunset header to be present")
	}

	if w.Header().Get(constants.HeaderLink) == "" {
		t.Error("Expected Link header to be present")
	}
}

func TestUnifiedAuth_BothHeadersPresent_BearerTakesPrecedence(t *testing.T) {
	// Setup stores
	store := NewAPIKeyStore()

	// Create two different API keys
	apiKey1 := constants.APIKeyPrefix + "key1_12345678901234567890123456789012345678901234567890123456"
	apiKey2 := constants.APIKeyPrefix + "key2_12345678901234567890123456789012345678901234567890123456"

	hashedKey1 := HashAPIKey(apiKey1)
	hashedKey2 := HashAPIKey(apiKey2)

	store.Add(hashedKey1, &APIKeyInfo{
		ID:   "key1",
		Name: "Bearer Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	})

	store.Add(hashedKey2, &APIKeyInfo{
		ID:   "key2",
		Name: "Legacy Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	})

	// Setup API key middleware
	apiKeyMiddleware := NewAPIKeyMiddleware(APIKeyConfig{
		Enabled: true,
		Store:   store,
	})

	// Setup unified auth middleware with legacy support
	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		APIKeyMiddleware:    apiKeyMiddleware,
		LegacyHeaderSupport: true,
		ProtectByDefault:    true,
	})

	// Create test handler that checks which key was used
	var usedKeyID string
	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		keyInfo, ok := GetAPIKeyInfo(r.Context())
		if ok {
			usedKeyID = keyInfo.ID
		}
		w.WriteHeader(http.StatusOK)
	})

	// Create request with BOTH headers
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer "+apiKey1)
	req.Header.Set(constants.HeaderAPIKey, apiKey2)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify that Bearer token (key1) was used, not legacy X-API-Key (key2)
	if usedKeyID != "key1" {
		t.Errorf("Expected Bearer token (key1) to be used, but got %s", usedKeyID)
	}

	// Verify NO deprecation headers (since we used Bearer)
	if w.Header().Get(constants.HeaderDeprecation) != "" {
		t.Error("Expected NO Deprecation header when using Bearer token")
	}
}

func TestUnifiedAuth_MissingAuthorizationHeader(t *testing.T) {
	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		ProtectByDefault: true,
	})

	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for missing auth")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "authentication_required" {
		t.Errorf("Expected error 'authentication_required', got %v", resp["error"])
	}
}

func TestUnifiedAuth_InvalidTokenFormat(t *testing.T) {
	jwtMiddleware := NewJWTMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		JWTMiddleware:    jwtMiddleware,
		ProtectByDefault: true,
	})

	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for invalid token")
	})

	// Token that doesn't match JWT format or API key prefix
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer invalid_token_format")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "invalid_token_format" {
		t.Errorf("Expected error 'invalid_token_format', got %v", resp["error"])
	}
}

func TestUnifiedAuth_ExpiredJWT(t *testing.T) {
	jwtMiddleware := NewJWTMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		JWTMiddleware:    jwtMiddleware,
		ProtectByDefault: true,
	})

	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for expired token")
	})

	// Generate an expired token
	token, _ := GenerateToken(testSecret, "user123", []string{"admin"}, -time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer "+token)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "invalid_credentials" {
		t.Errorf("Expected error 'invalid_credentials', got %v", resp["error"])
	}
}

func TestUnifiedAuth_InvalidAPIKey(t *testing.T) {
	// Setup API key middleware with empty store
	apiKeyMiddleware := NewAPIKeyMiddleware(APIKeyConfig{
		Enabled: true,
		Store:   NewAPIKeyStore(),
	})

	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		APIKeyMiddleware: apiKeyMiddleware,
		ProtectByDefault: true,
	})

	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for invalid API key")
	})

	// API key with correct format but not in store
	apiKey := constants.APIKeyPrefix + "invalid1234567890123456789012345678901234567890123456789012"
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer "+apiKey)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "invalid_credentials" {
		t.Errorf("Expected error 'invalid_credentials', got %v", resp["error"])
	}
}

func TestUnifiedAuth_InsufficientPermissions(t *testing.T) {
	// Setup API key with read-only permissions
	store := NewAPIKeyStore()
	apiKey := constants.APIKeyPrefix + "readonly123456789012345678901234567890123456789012345678901"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "key1",
		Name: "Read-Only Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	})

	apiKeyMiddleware := NewAPIKeyMiddleware(APIKeyConfig{
		Enabled: true,
		Store:   store,
	})

	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{
		APIKeyMiddleware: apiKeyMiddleware,
		ProtectByDefault: true,
	})

	handler := unified.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for insufficient permissions")
	})

	// Try to POST (write operation) with read-only key
	req := httptest.NewRequest(http.MethodPost, "/data", nil)
	req.Header.Set(constants.HeaderAuthorization, "Bearer "+apiKey)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "forbidden" {
		t.Errorf("Expected error 'forbidden', got %v", resp["error"])
	}
}

func TestUnifiedAuth_TokenTypeDetection(t *testing.T) {
	unified := NewUnifiedAuthMiddleware(UnifiedAuthConfig{})

	tests := []struct {
		name         string
		token        string
		expectJWT    bool
		expectAPIKey bool
	}{
		{
			name:         "Valid JWT format",
			token:        "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			expectJWT:    true,
			expectAPIKey: false,
		},
		{
			name:         "API key with prefix",
			token:        "moon_live_abc123def456",
			expectJWT:    false,
			expectAPIKey: true,
		},
		{
			name:         "Invalid format - too few segments",
			token:        "not.jwt",
			expectJWT:    false,
			expectAPIKey: false,
		},
		{
			name:         "Invalid format - random string",
			token:        "randomstring",
			expectJWT:    false,
			expectAPIKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isJWT := unified.isJWT(tt.token)
			isAPIKey := unified.isAPIKey(tt.token)

			if isJWT != tt.expectJWT {
				t.Errorf("isJWT() = %v, want %v", isJWT, tt.expectJWT)
			}

			if isAPIKey != tt.expectAPIKey {
				t.Errorf("isAPIKey() = %v, want %v", isAPIKey, tt.expectAPIKey)
			}
		})
	}
}
