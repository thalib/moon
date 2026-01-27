package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAPIKeyStore(t *testing.T) {
	store := NewAPIKeyStore()
	if store == nil {
		t.Fatal("NewAPIKeyStore returned nil")
	}
	if store.keys == nil {
		t.Fatal("Store keys map is nil")
	}
}

func TestAPIKeyStore_AddAndGet(t *testing.T) {
	store := NewAPIKeyStore()

	info := &APIKeyInfo{
		ID:   "key1",
		Name: "Test Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	}

	hashedKey := HashAPIKey("test-api-key-12345678901234567890")
	store.Add(hashedKey, info)

	retrieved, ok := store.Get(hashedKey)
	if !ok {
		t.Fatal("Failed to retrieve added key")
	}

	if retrieved.ID != info.ID {
		t.Errorf("Expected ID %s, got %s", info.ID, retrieved.ID)
	}

	if retrieved.Name != info.Name {
		t.Errorf("Expected Name %s, got %s", info.Name, retrieved.Name)
	}
}

func TestAPIKeyStore_Remove(t *testing.T) {
	store := NewAPIKeyStore()

	info := &APIKeyInfo{ID: "key1", Name: "Test Key"}
	hashedKey := HashAPIKey("test-api-key-12345678901234567890")
	store.Add(hashedKey, info)

	store.Remove(hashedKey)

	_, ok := store.Get(hashedKey)
	if ok {
		t.Error("Key should have been removed")
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "test-api-key-12345678901234567890"

	hash1 := HashAPIKey(key)
	hash2 := HashAPIKey(key)

	if hash1 != hash2 {
		t.Error("Same key should produce same hash")
	}

	if len(hash1) != 64 { // SHA-256 produces 64 hex characters
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}

	// Different key should produce different hash
	differentHash := HashAPIKey("different-key-12345678901234567890")
	if hash1 == differentHash {
		t.Error("Different keys should produce different hashes")
	}
}

func TestNewAPIKeyMiddleware_Defaults(t *testing.T) {
	config := APIKeyConfig{
		Enabled: true,
	}

	middleware := NewAPIKeyMiddleware(config)

	if middleware.config.HeaderName != "X-API-Key" {
		t.Errorf("Expected default header name 'X-API-Key', got '%s'", middleware.config.HeaderName)
	}

	if middleware.config.QueryParamName != "api_key" {
		t.Errorf("Expected default query param name 'api_key', got '%s'", middleware.config.QueryParamName)
	}

	if middleware.config.Store == nil {
		t.Error("Store should be initialized by default")
	}
}

func TestAPIKeyMiddleware_Disabled(t *testing.T) {
	config := APIKeyConfig{
		Enabled: false,
	}

	middleware := NewAPIKeyMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if !handlerCalled {
		t.Error("Handler should be called when API key auth is disabled")
	}
}

func TestAPIKeyMiddleware_UnprotectedPath(t *testing.T) {
	config := APIKeyConfig{
		Enabled:          true,
		UnprotectedPaths: []string{"/health", "/public/*"},
		ProtectByDefault: true,
	}

	middleware := NewAPIKeyMiddleware(config)

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handlerCalled = false
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			middleware.Authenticate(handler)(w, req)

			if !handlerCalled {
				t.Error("Handler should be called for unprotected path")
			}
		})
	}
}

func TestAPIKeyMiddleware_MissingAPIKey(t *testing.T) {
	store := NewAPIKeyStore()
	config := APIKeyConfig{
		Enabled:          true,
		Store:            store,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewAPIKeyMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if handlerCalled {
		t.Error("Handler should not be called without API key")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAPIKeyMiddleware_InvalidAPIKey(t *testing.T) {
	store := NewAPIKeyStore()
	config := APIKeyConfig{
		Enabled:          true,
		Store:            store,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewAPIKeyMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-API-Key", "invalid-key-that-does-not-exist")
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if handlerCalled {
		t.Error("Handler should not be called with invalid API key")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAPIKeyMiddleware_ValidAPIKey_Header(t *testing.T) {
	store := NewAPIKeyStore()
	apiKey := "valid-api-key-123456789012345678901"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "key1",
		Name: "Test Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	})

	config := APIKeyConfig{
		Enabled:          true,
		Store:            store,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewAPIKeyMiddleware(config)

	var capturedInfo *APIKeyInfo
	handler := func(w http.ResponseWriter, r *http.Request) {
		info, ok := GetAPIKeyInfo(r.Context())
		if ok {
			capturedInfo = info
		}
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if capturedInfo == nil {
		t.Fatal("API key info was not passed to handler")
	}

	if capturedInfo.ID != "key1" {
		t.Errorf("Expected key ID 'key1', got '%s'", capturedInfo.ID)
	}
}

func TestAPIKeyMiddleware_ValidAPIKey_QueryParam(t *testing.T) {
	store := NewAPIKeyStore()
	apiKey := "valid-api-key-123456789012345678901"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "key1",
		Name: "Test Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	})

	config := APIKeyConfig{
		Enabled:          true,
		Store:            store,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewAPIKeyMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test?api_key="+apiKey, nil)
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if !handlerCalled {
		t.Error("Handler should be called with valid API key in query param")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAPIKeyMiddleware_Permissions(t *testing.T) {
	store := NewAPIKeyStore()
	apiKey := "valid-api-key-123456789012345678901"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "key1",
		Name: "Read Only Key",
		Permissions: []Permission{
			{Resource: "/api/*", Actions: []string{"read"}},
		},
	})

	config := APIKeyConfig{
		Enabled:          true,
		Store:            store,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewAPIKeyMiddleware(config)

	t.Run("GET request with read permission", func(t *testing.T) {
		handlerCalled := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("X-API-Key", apiKey)
		w := httptest.NewRecorder()

		middleware.Authenticate(handler)(w, req)

		if !handlerCalled {
			t.Error("Handler should be called for GET with read permission")
		}
	})

	t.Run("POST request without write permission", func(t *testing.T) {
		handlerCalled := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		}

		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		req.Header.Set("X-API-Key", apiKey)
		w := httptest.NewRecorder()

		middleware.Authenticate(handler)(w, req)

		if handlerCalled {
			t.Error("Handler should not be called for POST without write permission")
		}

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})
}

func TestAPIKeyMiddleware_AdminPermission(t *testing.T) {
	store := NewAPIKeyStore()
	apiKey := "admin-api-key-123456789012345678901"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "admin1",
		Name: "Admin Key",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"admin"}},
		},
	})

	config := APIKeyConfig{
		Enabled:          true,
		Store:            store,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
	}

	middleware := NewAPIKeyMiddleware(config)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method+" with admin permission", func(t *testing.T) {
			handlerCalled := false
			handler := func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			}

			req := httptest.NewRequest(method, "/api/test", nil)
			req.Header.Set("X-API-Key", apiKey)
			w := httptest.NewRecorder()

			middleware.Authenticate(handler)(w, req)

			if !handlerCalled {
				t.Errorf("Handler should be called for %s with admin permission", method)
			}
		})
	}
}

func TestAPIKeyMiddleware_EndpointSpecificPerms(t *testing.T) {
	store := NewAPIKeyStore()
	apiKey := "limited-api-key-12345678901234567890"
	hashedKey := HashAPIKey(apiKey)
	store.Add(hashedKey, &APIKeyInfo{
		ID:   "limited1",
		Name: "Limited Key",
		Permissions: []Permission{
			{Resource: "/api/v1/special", Actions: []string{"special_action"}},
		},
	})

	config := APIKeyConfig{
		Enabled:          true,
		Store:            store,
		ProtectedPaths:   []string{"/api/*"},
		ProtectByDefault: false,
		EndpointPerms: map[string][]string{
			"/api/v1/special": {"special_action"},
		},
	}

	middleware := NewAPIKeyMiddleware(config)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/special", nil)
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()

	middleware.Authenticate(handler)(w, req)

	if !handlerCalled {
		t.Error("Handler should be called for endpoint with custom permission")
	}
}

func TestValidateAPIKeyFormat(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{"Valid key 40 chars", "abcdefghij1234567890ABCDEFGHIJ1234567890", true},
		{"Valid with dashes 40 chars", "abc-def-ghi-123-456-789-0AB-CD-EF-XYZ-123", true},
		{"Valid with underscores 40 chars", "abc_def_ghi_123_456_789_0AB_CD_EF_XYZ_123", true},
		{"Too short 32 chars", "abcdefghij1234567890ABCDEFGHIJ12", false},
		{"Too short", "short", false},
		{"Contains special chars", "abc!def@ghi#123$456%789^0AB&CD*EF!123456", false},
		{"Contains spaces", "abc def ghi 123 456 789 0AB CD EF XYZ 123", false},
		{"Empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateAPIKeyFormat(tc.key)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestCompareAPIKeys(t *testing.T) {
	key1 := "test-key-1234567890123456789012345"
	key2 := "test-key-1234567890123456789012345"
	key3 := "different-key-12345678901234567890"

	if !CompareAPIKeys(key1, key2) {
		t.Error("Same keys should match")
	}

	if CompareAPIKeys(key1, key3) {
		t.Error("Different keys should not match")
	}
}

func TestGetAPIKeyInfo(t *testing.T) {
	t.Run("Info present in context", func(t *testing.T) {
		info := &APIKeyInfo{
			ID:   "key1",
			Name: "Test Key",
		}

		ctx := context.WithValue(context.Background(), APIKeyContextKey, info)

		retrieved, ok := GetAPIKeyInfo(ctx)

		if !ok {
			t.Fatal("GetAPIKeyInfo returned false for context with info")
		}

		if retrieved.ID != "key1" {
			t.Errorf("Expected ID 'key1', got '%s'", retrieved.ID)
		}
	})

	t.Run("Info not present in context", func(t *testing.T) {
		ctx := context.Background()

		_, ok := GetAPIKeyInfo(ctx)

		if ok {
			t.Error("GetAPIKeyInfo returned true for context without info")
		}
	})
}

func TestAPIKeyMiddleware_WriteAuthError(t *testing.T) {
	middleware := NewAPIKeyMiddleware(APIKeyConfig{Enabled: true})

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
}

func TestAPIKeyMiddleware_HasPermissions(t *testing.T) {
	middleware := NewAPIKeyMiddleware(APIKeyConfig{Enabled: true})

	testCases := []struct {
		name          string
		keyInfo       *APIKeyInfo
		path          string
		requiredPerms []string
		expected      bool
	}{
		{
			name: "Has required permission",
			keyInfo: &APIKeyInfo{
				Permissions: []Permission{
					{Resource: "/api/*", Actions: []string{"read"}},
				},
			},
			path:          "/api/test",
			requiredPerms: []string{"read"},
			expected:      true,
		},
		{
			name: "Missing required permission",
			keyInfo: &APIKeyInfo{
				Permissions: []Permission{
					{Resource: "/api/*", Actions: []string{"read"}},
				},
			},
			path:          "/api/test",
			requiredPerms: []string{"write"},
			expected:      false,
		},
		{
			name: "Admin has all permissions",
			keyInfo: &APIKeyInfo{
				Permissions: []Permission{
					{Resource: "*", Actions: []string{"admin"}},
				},
			},
			path:          "/api/test",
			requiredPerms: []string{"write", "delete"},
			expected:      true,
		},
		{
			name: "Wildcard resource",
			keyInfo: &APIKeyInfo{
				Permissions: []Permission{
					{Resource: "*", Actions: []string{"read", "write"}},
				},
			},
			path:          "/anywhere",
			requiredPerms: []string{"read"},
			expected:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := middleware.hasPermissions(tc.keyInfo, tc.path, tc.requiredPerms)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestAPIKeyStore_Concurrent(t *testing.T) {
	store := NewAPIKeyStore()

	// Test concurrent access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := HashAPIKey("key" + string(rune(i)))
			store.Add(key, &APIKeyInfo{ID: "id" + string(rune(i))})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := HashAPIKey("key" + string(rune(i)))
			store.Get(key)
		}
		done <- true
	}()

	<-done
	<-done
}
