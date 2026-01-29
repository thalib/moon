package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// mockAggDriver implements database.Driver for testing
type mockAggDriver struct {
	dialect      database.DialectType
	queryFunc    func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	queryRowFunc func(ctx context.Context, query string, args ...any) *sql.Row
	rowScanner   func(dest ...any) error
}

func (m *mockAggDriver) Connect(ctx context.Context) error {
	return nil
}

func (m *mockAggDriver) Close() error {
	return nil
}

func (m *mockAggDriver) Ping(ctx context.Context) error {
	return nil
}

func (m *mockAggDriver) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockAggDriver) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	// Return a mock row - we need a real sql.Row but we can't easily construct one
	// For testing, we'll use a different approach
	return nil
}

func (m *mockAggDriver) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (m *mockAggDriver) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (m *mockAggDriver) Dialect() database.DialectType {
	return m.dialect
}

func (m *mockAggDriver) DB() *sql.DB {
	return nil
}

func TestValidateNumericField(t *testing.T) {
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeFloat},
			{Name: "quantity", Type: registry.TypeInteger},
			{Name: "status", Type: registry.TypeString},
		},
	}

	tests := []struct {
		name      string
		fieldName string
		wantError bool
	}{
		{
			name:      "valid float field",
			fieldName: "total",
			wantError: false,
		},
		{
			name:      "valid integer field",
			fieldName: "quantity",
			wantError: false,
		},
		{
			name:      "invalid non-numeric field",
			fieldName: "status",
			wantError: true,
		},
		{
			name:      "invalid non-existent field",
			fieldName: "unknown",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNumericField(collection, tt.fieldName)
			if (err != nil) != tt.wantError {
				t.Errorf("validateNumericField() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestAggregationHandler_CollectionNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	driver := &mockAggDriver{dialect: database.DialectSQLite}
	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request, collection string)
		url     string
	}{
		{
			name:    "count collection not found",
			handler: handler.Count,
			url:     "/api/v1/unknown:count",
		},
		{
			name:    "sum collection not found",
			handler: handler.Sum,
			url:     "/api/v1/unknown:sum?field=total",
		},
		{
			name:    "avg collection not found",
			handler: handler.Avg,
			url:     "/api/v1/unknown:avg?field=total",
		},
		{
			name:    "min collection not found",
			handler: handler.Min,
			url:     "/api/v1/unknown:min?field=total",
		},
		{
			name:    "max collection not found",
			handler: handler.Max,
			url:     "/api/v1/unknown:max?field=total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req, "unknown")

			if w.Code != http.StatusNotFound {
				t.Errorf("expected status 404, got %d", w.Code)
			}
		})
	}
}

func TestAggregationHandler_MissingField(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name:    "orders",
		Columns: []registry.Column{{Name: "total", Type: registry.TypeFloat}},
	}
	reg.Set(collection)

	driver := &mockAggDriver{dialect: database.DialectSQLite}
	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request, collection string)
		url     string
	}{
		{
			name:    "sum missing field",
			handler: handler.Sum,
			url:     "/api/v1/orders:sum",
		},
		{
			name:    "avg missing field",
			handler: handler.Avg,
			url:     "/api/v1/orders:avg",
		},
		{
			name:    "min missing field",
			handler: handler.Min,
			url:     "/api/v1/orders:min",
		},
		{
			name:    "max missing field",
			handler: handler.Max,
			url:     "/api/v1/orders:max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req, "orders")

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestAggregationHandler_NonNumericField(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "status", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockAggDriver{dialect: database.DialectSQLite}
	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request, collection string)
		url     string
	}{
		{
			name:    "sum non-numeric field",
			handler: handler.Sum,
			url:     "/api/v1/orders:sum?field=status",
		},
		{
			name:    "avg non-numeric field",
			handler: handler.Avg,
			url:     "/api/v1/orders:avg?field=status",
		},
		{
			name:    "min non-numeric field",
			handler: handler.Min,
			url:     "/api/v1/orders:min?field=status",
		},
		{
			name:    "max non-numeric field",
			handler: handler.Max,
			url:     "/api/v1/orders:max?field=status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req, "orders")

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			body := w.Body.String()
			if !strings.Contains(body, "not numeric") {
				t.Error("expected error message about non-numeric field")
			}
		})
	}
}

