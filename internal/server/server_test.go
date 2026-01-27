package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/internal/config"
	"github.com/thalib/moon/internal/database"
	"github.com/thalib/moon/internal/registry"
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

	return New(cfg, driver, reg)
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

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if response["database"] != "sqlite" {
		t.Errorf("Expected database 'sqlite', got '%v'", response["database"])
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
