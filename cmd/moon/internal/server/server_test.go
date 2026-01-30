package server

import (
	"context"
	"encoding/json"
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
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Collections list with empty prefix",
			prefix:         "",
			requestPath:    "/collections:list",
			expectedStatus: http.StatusOK,
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
			requestPath:    "/api/v1/users:list",
			expectedStatus: http.StatusNotFound, // Collection doesn't exist yet
		},
		{
			name:           "Collection action with empty prefix",
			prefix:         "",
			requestPath:    "/users:list",
			expectedStatus: http.StatusNotFound, // Collection doesn't exist yet
		},
		{
			name:           "Collection action with /moon/api/ prefix",
			prefix:         "/moon/api",
			requestPath:    "/moon/api/products:list",
			expectedStatus: http.StatusNotFound, // Collection doesn't exist yet
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

	expectedBody := "Darling, the Moon is still the Moon in all of its phases."
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
			expectedBody:        "Darling, the Moon is still the Moon in all of its phases.",
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
			expectedStatus: http.StatusOK,
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
