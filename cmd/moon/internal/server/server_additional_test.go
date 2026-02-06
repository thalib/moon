package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// TestDynamicDataHandler_AllActions tests all supported action types
func TestDynamicDataHandler_AllActions(t *testing.T) {
	srv := setupTestServer(t)

	// Register a test collection
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger},
			{Name: "status", Type: registry.TypeString},
		},
	}
	srv.registry.Set(collection)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		// GET actions - Now require authentication (401)
		{
			name:           "list action with GET",
			method:         http.MethodGet,
			path:           "/orders:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "get action with GET",
			method:         http.MethodGet,
			path:           "/orders:get?id=01ARYZ6S41TSV4RRFFQ69G5FAV",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "count action with GET",
			method:         http.MethodGet,
			path:           "/orders:count",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "sum action with GET",
			method:         http.MethodGet,
			path:           "/orders:sum?field=total",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "avg action with GET",
			method:         http.MethodGet,
			path:           "/orders:avg?field=total",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "min action with GET",
			method:         http.MethodGet,
			path:           "/orders:min?field=total",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "max action with GET",
			method:         http.MethodGet,
			path:           "/orders:max?field=total",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		// Wrong methods for GET actions
		{
			name:           "list with POST should fail",
			method:         http.MethodPost,
			path:           "/orders:list",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "get with POST should fail",
			method:         http.MethodPost,
			path:           "/orders:get",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "count with POST should fail",
			method:         http.MethodPost,
			path:           "/orders:count",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "sum with POST should fail",
			method:         http.MethodPost,
			path:           "/orders:sum?field=total",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "avg with POST should fail",
			method:         http.MethodPost,
			path:           "/orders:avg?field=total",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "min with POST should fail",
			method:         http.MethodPost,
			path:           "/orders:min?field=total",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "max with POST should fail",
			method:         http.MethodPost,
			path:           "/orders:max?field=total",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		// Wrong methods for POST actions
		{
			name:           "create with GET should fail",
			method:         http.MethodGet,
			path:           "/orders:create",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "update with GET should fail",
			method:         http.MethodGet,
			path:           "/orders:update",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "destroy with GET should fail",
			method:         http.MethodGet,
			path:           "/orders:destroy",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		// Unknown action
		{
			name:           "unknown action",
			method:         http.MethodGet,
			path:           "/orders:unknown",
			expectedStatus: http.StatusNotFound,
		},
		// Invalid path (no colon)
		{
			name:           "invalid path without colon",
			method:         http.MethodGet,
			path:           "/orders",
			expectedStatus: http.StatusNotFound,
		},
		// Collections endpoint protection
		{
			name:           "collections should be protected",
			method:         http.MethodGet,
			path:           "/collections:unknown",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHealthHandler_DatabaseDown tests health when database is down
func TestHealthHandler_DatabaseDown(t *testing.T) {
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Port: 6006,
			Host: "0.0.0.0",
		},
	}

	// Create a mock driver that fails on ping
	mockDriver := &mockFailingPingDriver{}

	reg := registry.NewSchemaRegistry()

	srv := New(cfg, mockDriver, reg, "1.0")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "down" {
		t.Errorf("Expected status 'down' when db fails, got '%v'", response["status"])
	}
}

// mockFailingPingDriver is a mock driver that fails on ping
type mockFailingPingDriver struct{}

func (m *mockFailingPingDriver) Connect(ctx context.Context) error                { return nil }
func (m *mockFailingPingDriver) Close() error                                     { return nil }
func (m *mockFailingPingDriver) Ping(ctx context.Context) error                   { return context.DeadlineExceeded }
func (m *mockFailingPingDriver) Dialect() database.DialectType                    { return database.DialectSQLite }
func (m *mockFailingPingDriver) DB() *sql.DB                                      { return nil }
func (m *mockFailingPingDriver) ListTables(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockFailingPingDriver) GetTableInfo(ctx context.Context, tableName string) (*database.TableInfo, error) {
	return nil, nil
}
func (m *mockFailingPingDriver) TableExists(ctx context.Context, tableName string) (bool, error) {
	return false, nil
}
func (m *mockFailingPingDriver) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}
func (m *mockFailingPingDriver) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}
func (m *mockFailingPingDriver) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}
func (m *mockFailingPingDriver) BeginTx(ctx context.Context) (*sql.Tx, error) { return nil, nil }

// TestServerRoutes_DocumentationEndpoints tests documentation endpoints
func TestServerRoutes_DocumentationEndpoints(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "HTML documentation",
			method:         http.MethodGet,
			path:           "/doc/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Markdown documentation",
			method:         http.MethodGet,
			path:           "/doc/llms-full.txt",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Old markdown endpoint should return 404",
			method:         http.MethodGet,
			path:           "/doc/md",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Refresh documentation cache",
			method:         http.MethodPost,
			path:           "/doc:refresh",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestServerRoutes_CollectionsEndpoints tests collections management endpoints
func TestServerRoutes_CollectionsEndpoints(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Collections list",
			method:         http.MethodGet,
			path:           "/collections:list",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
		{
			name:           "Collections get",
			method:         http.MethodGet,
			path:           "/collections:get?name=test",
			expectedStatus: http.StatusUnauthorized, // Requires authentication
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			srv.mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestServerRoutes_WithDifferentPrefixes tests routes work with various prefixes
func TestServerRoutes_WithDifferentPrefixes(t *testing.T) {
	prefixes := []struct {
		prefix     string
		healthPath string
		docsPath   string
	}{
		{prefix: "", healthPath: "/health", docsPath: "/doc/"},
		{prefix: "/api/v1", healthPath: "/api/v1/health", docsPath: "/api/v1/doc/"},
		{prefix: "/moon", healthPath: "/moon/health", docsPath: "/moon/doc/"},
		{prefix: "/api/v2/backend", healthPath: "/api/v2/backend/health", docsPath: "/api/v2/backend/doc/"},
	}

	for _, tt := range prefixes {
		t.Run("prefix="+tt.prefix, func(t *testing.T) {
			srv := setupTestServerWithPrefix(t, tt.prefix)

			// Test health endpoint
			req := httptest.NewRequest(http.MethodGet, tt.healthPath, nil)
			w := httptest.NewRecorder()
			srv.mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Health at %s: expected status 200, got %d", tt.healthPath, w.Code)
			}

			// Test docs endpoint
			req = httptest.NewRequest(http.MethodGet, tt.docsPath, nil)
			w = httptest.NewRecorder()
			srv.mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Docs at %s: expected status 200, got %d", tt.docsPath, w.Code)
			}
		})
	}
}

// TestServer_WriteJSONWithDifferentTypes tests writeJSON with various data types
func TestServer_WriteJSONWithDifferentTypes(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name string
		data any
	}{
		{
			name: "map",
			data: map[string]any{"key": "value"},
		},
		{
			name: "slice",
			data: []string{"a", "b", "c"},
		},
		{
			name: "struct",
			data: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}{Name: "test", Value: 42},
		},
		{
			name: "nested map",
			data: map[string]any{
				"outer": map[string]any{
					"inner": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			srv.writeJSON(w, http.StatusOK, tt.data)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			// Verify it's valid JSON
			var result any
			if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
				t.Errorf("Failed to decode JSON: %v", err)
			}
		})
	}
}
