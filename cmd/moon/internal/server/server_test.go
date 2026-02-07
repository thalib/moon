package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

func setupTestServer(t *testing.T) *Server {
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Port: 6006,
			Host: "0.0.0.0",
		},
		Database: config.DatabaseConfig{
			Connection: "sqlite",
			Database:   ":memory:",
		},
		Logging: config.LoggingConfig{
			Path: "/tmp",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 3600,
		},
	}

	dbConfig := database.Config{
		ConnectionString: "sqlite://:memory:",
	}

	driver, err := database.NewDriver(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}

	if err := driver.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	reg := registry.NewSchemaRegistry()

	return New(cfg, driver, reg, "1-test")
}

func TestNew(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil {
		t.Fatal("New() returned nil")
	}

	if srv.config == nil {
		t.Error("Server config is nil")
	}

	if srv.db == nil {
		t.Error("Server db is nil")
	}

	if srv.registry == nil {
		t.Error("Server registry is nil")
	}

	if srv.mux == nil {
		t.Error("Server mux is nil")
	}

	if srv.server == nil {
		t.Error("Server server is nil")
	}
}

func TestHealthHandler(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "live" {
		t.Errorf("Expected status 'live', got '%v'", response["status"])
	}

	if response["name"] != "moon" {
		t.Errorf("Expected name 'moon', got '%v'", response["name"])
	}

	if response["version"] != "1-test" {
		t.Errorf("Expected version '1-test', got '%v'", response["version"])
	}

	// Ensure no other fields are present
	if len(response) != 3 {
		t.Errorf("Expected exactly 3 fields, got %d: %v", len(response), response)
	}
}

func TestNotFoundHandler(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.notFoundHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] == nil {
		t.Error("Expected error message in response")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	srv := setupTestServer(t)

	handlerCalled := false
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	wrapped := srv.loggingMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Write header
	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rw.statusCode)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("Expected underlying status code %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	srv := setupTestServer(t)

	w := httptest.NewRecorder()

	data := map[string]any{
		"key":   "value",
		"count": 42,
	}

	srv.writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["key"] != "value" {
		t.Errorf("Expected key 'value', got '%v'", response["key"])
	}

	if response["count"].(float64) != 42 {
		t.Errorf("Expected count 42, got %v", response["count"])
	}
}

func TestWriteError(t *testing.T) {
	srv := setupTestServer(t)

	w := httptest.NewRecorder()

	srv.writeError(w, http.StatusBadRequest, "Test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Test error message" {
		t.Errorf("Expected error 'Test error message', got '%v'", response["error"])
	}

	if response["code"].(float64) != float64(http.StatusBadRequest) {
		t.Errorf("Expected code %d, got %v", http.StatusBadRequest, response["code"])
	}
}

func TestServerRoutes(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Health check",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func setupTestServerWithPrefix(t *testing.T, prefix string) *Server {
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Port:   6006,
			Host:   "0.0.0.0",
			Prefix: prefix,
		},
		Database: config.DatabaseConfig{
			Connection: "sqlite",
			Database:   ":memory:",
		},
		Logging: config.LoggingConfig{
			Path: "/tmp",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 3600,
		},
	}

	dbConfig := database.Config{
		ConnectionString: "sqlite://:memory:",
	}

	driver, err := database.NewDriver(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}

	if err := driver.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	reg := registry.NewSchemaRegistry()

	return New(cfg, driver, reg, "1-test")
}

func TestServerRoutes_WithPrefix(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		requestPath    string
		expectedStatus int
	}{
		{
			name:           "Health check with /api/v1 prefix",
			prefix:         "/api/v1",
			requestPath:    "/api/v1/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Health check without prefix should fail",
			prefix:         "/api/v1",
			requestPath:    "/health",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Health check with /moon/api prefix",
			prefix:         "/moon/api",
			requestPath:    "/moon/api/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Health check with empty prefix",
			prefix:         "",
			requestPath:    "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Collections list with prefix",
			prefix:         "/api/v1",
			requestPath:    "/api/v1/collections:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "Collections list with empty prefix",
			prefix:         "",
			requestPath:    "/collections:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := setupTestServerWithPrefix(t, tt.prefix)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDynamicDataHandler_WithPrefix(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		requestPath    string
		expectedStatus int
	}{
		{
			name:           "Collection action with /api/v1 prefix",
			prefix:         "/api/v1",
			requestPath:    "/api/v1/customers:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "Collection action with empty prefix",
			prefix:         "",
			requestPath:    "/customers:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "Collection action with /moon/api/ prefix",
			prefix:         "/moon/api",
			requestPath:    "/moon/api/products:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "Invalid path should 404 with prefix",
			prefix:         "/api/v1",
			requestPath:    "/api/v1/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Prevent collections endpoint access",
			prefix:         "/api/v1",
			requestPath:    "/api/v1/collections:invalid",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := setupTestServerWithPrefix(t, tt.prefix)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRootMessageHandler(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.rootMessageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/plain; charset=utf-8', got '%s'", contentType)
	}

	expectedBody := "Moon is running."
	actualBody := w.Body.String()
	if actualBody != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, actualBody)
	}
}

func TestRootMessageHandler_NonRootPath(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/something", nil)
	w := httptest.NewRecorder()

	srv.rootMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d for non-root path, got %d", http.StatusNotFound, w.Code)
	}
}

func TestRootMessageHandler_Integration(t *testing.T) {
	tests := []struct {
		name                string
		prefix              string
		requestPath         string
		expectedStatus      int
		expectedBody        string
		expectedContentType string
	}{
		{
			name:                "Root message with no prefix",
			prefix:              "",
			requestPath:         "/",
			expectedStatus:      http.StatusOK,
			expectedBody:        "Moon is running.",
			expectedContentType: "text/plain; charset=utf-8",
		},
		{
			name:           "Root path should 404 with prefix",
			prefix:         "/api/v1",
			requestPath:    "/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Other routes should still work with no prefix",
			prefix:         "",
			requestPath:    "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Collections endpoint should work with no prefix",
			prefix:         "",
			requestPath:    "/collections:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := setupTestServerWithPrefix(t, tt.prefix)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" {
				actualBody := w.Body.String()
				if actualBody != tt.expectedBody {
					t.Errorf("Expected body '%s', got '%s'", tt.expectedBody, actualBody)
				}
			}

			if tt.expectedContentType != "" {
				contentType := w.Header().Get("Content-Type")
				if contentType != tt.expectedContentType {
					t.Errorf("Expected Content-Type '%s', got '%s'", tt.expectedContentType, contentType)
				}
			}
		})
	}
}

// TestDynamicCollectionEndpoints_RoutingFix tests that the root handler does not
// intercept dynamic collection endpoints (bug fix for GitHub issue)
func TestDynamicCollectionEndpoints_RoutingFix(t *testing.T) {
	srv := setupTestServer(t)

	// Register a test collection in registry so we can test routing
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "order_id", Type: registry.TypeString},
		},
	}
	srv.registry.Set(collection)

	// Test that dynamic endpoints reach the handler (not 404)
	// They may return other errors (like 500 if table doesn't exist), but NOT 404
	tests := []struct {
		name        string
		method      string
		path        string
		shouldRoute bool
		description string
	}{
		{
			name:        "Orders list should route",
			method:      http.MethodGet,
			path:        "/orders:list",
			shouldRoute: true,
			description: "Should reach dynamic handler, not return 404",
		},
		{
			name:        "Orders count should route",
			method:      http.MethodGet,
			path:        "/orders:count",
			shouldRoute: true,
			description: "Should reach aggregation handler, not return 404",
		},
		{
			name:        "Root still returns 200",
			method:      http.MethodGet,
			path:        "/",
			shouldRoute: true,
			description: "Root handler should work",
		},
		{
			name:        "Collections list still works",
			method:      http.MethodGet,
			path:        "/collections:list",
			shouldRoute: true,
			description: "Collections endpoint should still work",
		},
		{
			name:        "Invalid path returns 404",
			method:      http.MethodGet,
			path:        "/invalid",
			shouldRoute: false,
			description: "Invalid paths should return 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if tt.shouldRoute {
				if w.Code == http.StatusNotFound {
					t.Errorf("%s: got 404 (routing failed), expected handler to be reached. Body: %s",
						tt.description, w.Body.String())
				}
			} else {
				if w.Code != http.StatusNotFound {
					t.Errorf("%s: expected 404, got %d. Body: %s",
						tt.description, w.Code, w.Body.String())
				}
			}
		})
	}
}

// TestDynamicCollectionEndpoints_MultipleCollections tests that multiple
// collections can have their dynamic endpoints route correctly
func TestDynamicCollectionEndpoints_MultipleCollections(t *testing.T) {
	srv := setupTestServer(t)

	// Register multiple collections in the registry
	collections := []string{"products", "orders", "customers"}

	for _, collName := range collections {
		collection := &registry.Collection{
			Name: collName,
			Columns: []registry.Column{
				{Name: "name", Type: registry.TypeString},
			},
		}
		srv.registry.Set(collection)
	}

	// Test that all dynamic endpoints route correctly (not 404)
	for _, collName := range collections {
		t.Run(fmt.Sprintf("List %s", collName), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s:list", collName), nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("Got 404 for /%s:list (routing failed), expected handler to be reached. Body: %s",
					collName, w.Body.String())
			}
		})
	}
}

// TestPublicCORSHeaders tests PRD-052: Public CORS for Health and Docs Endpoints
func TestPublicCORSHeaders(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name           string
		path           string
		description    string
		expectCORS     bool
		expectWildcard bool
	}{
		{
			name:           "health_endpoint",
			path:           "/health",
			description:    "Health endpoint should have Access-Control-Allow-Origin: *",
			expectCORS:     true,
			expectWildcard: true,
		},
		{
			name:           "doc_html_endpoint",
			path:           "/doc/",
			description:    "Doc HTML endpoint should have Access-Control-Allow-Origin: *",
			expectCORS:     true,
			expectWildcard: true,
		},
		{
			name:           "doc_markdown_endpoint",
			path:           "/doc/llms-full.txt",
			description:    "Doc Markdown endpoint should have Access-Control-Allow-Origin: *",
			expectCORS:     true,
			expectWildcard: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			// Set an Origin header to simulate cross-origin request
			req.Header.Set("Origin", "https://example.com")
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			// Check for CORS header
			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS {
				if corsHeader == "" {
					t.Errorf("%s: Expected Access-Control-Allow-Origin header, got none", tt.description)
				}
				if tt.expectWildcard && corsHeader != "*" {
					t.Errorf("%s: Expected Access-Control-Allow-Origin: *, got %s", tt.description, corsHeader)
				}
			}
		})
	}
}

// TestPublicCORSPreflight tests CORS preflight OPTIONS requests for public endpoints
func TestPublicCORSPreflight(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name string
		path string
	}{
		{"health_preflight", "/health"},
		{"doc_html_preflight", "/doc/"},
		{"doc_llms_full_preflight", "/doc/llms-full.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", "GET")
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			// OPTIONS should return 204 No Content
			if w.Code != http.StatusNoContent {
				t.Errorf("Expected status %d for OPTIONS request, got %d", http.StatusNoContent, w.Code)
			}

			// Check CORS headers
			corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if corsOrigin != "*" {
				t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", corsOrigin)
			}

			corsMethods := w.Header().Get("Access-Control-Allow-Methods")
			if corsMethods == "" {
				t.Error("Expected Access-Control-Allow-Methods header, got none")
			}
		})
	}
}

// TestAuthEndpointsDoNotHaveWildcardCORS tests that auth endpoints do not have wildcard CORS
func TestAuthEndpointsDoNotHaveWildcardCORS(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name string
		path string
	}{
		{"auth_login", "/auth:login"},
		{"auth_refresh", "/auth:refresh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			req.Header.Set("Origin", "https://example.com")
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			// Auth endpoints should not have wildcard CORS by default
			// (they should use the standard CORS middleware which requires config)
			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			if corsHeader == "*" {
				t.Errorf("%s should not have Access-Control-Allow-Origin: *, got %s", tt.path, corsHeader)
			}
		})
	}
}

// TestAuthEndpointsCORSPreflight tests CORS preflight OPTIONS requests for auth endpoints
func TestAuthEndpointsCORSPreflight(t *testing.T) {
	// Setup server with CORS enabled
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Port: 6006,
			Host: "0.0.0.0",
		},
		Database: config.DatabaseConfig{
			Connection: "sqlite",
			Database:   ":memory:",
		},
		Logging: config.LoggingConfig{
			Path: "/tmp",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 3600,
		},
		CORS: config.CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"https://app.example.com"},
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           3600,
		},
	}

	dbConfig := database.Config{
		ConnectionString: "sqlite://:memory:",
	}

	driver, err := database.NewDriver(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}

	if err := driver.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	reg := registry.NewSchemaRegistry()
	srv := New(cfg, driver, reg, "1-test")

	tests := []struct {
		name         string
		path         string
		expectStatus int
	}{
		{"auth_login", "/auth:login", http.StatusNoContent},
		{"auth_refresh", "/auth:refresh", http.StatusNoContent},
		{"auth_logout", "/auth:logout", http.StatusNoContent},
		{"auth_me", "/auth:me", http.StatusNoContent},
		{"collections_list", "/collections:list", http.StatusNoContent},
		{"collections_get", "/collections:get", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			req.Header.Set("Origin", "https://app.example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")
			req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			// Should return 204 No Content for OPTIONS preflight
			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d for OPTIONS %s, got %d", tt.expectStatus, tt.path, w.Code)
			}

			// Check CORS headers
			corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if corsOrigin != "https://app.example.com" {
				t.Errorf("Expected Access-Control-Allow-Origin: https://app.example.com, got %s", corsOrigin)
			}

			corsMethods := w.Header().Get("Access-Control-Allow-Methods")
			if corsMethods == "" {
				t.Errorf("Expected Access-Control-Allow-Methods header to be set")
			}

			corsHeaders := w.Header().Get("Access-Control-Allow-Headers")
			if corsHeaders == "" {
				t.Errorf("Expected Access-Control-Allow-Headers header to be set")
			}

			corsMaxAge := w.Header().Get("Access-Control-Max-Age")
			if corsMaxAge == "" {
				t.Errorf("Expected Access-Control-Max-Age header to be set")
			}
		})
	}
}
