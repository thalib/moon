package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// createTestDB creates an in-memory SQLite database with test data
func createTestDB(t *testing.T) database.Driver {
	cfg := database.Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := database.NewDriver(cfg)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create test table
	_, err = driver.Exec(ctx, `CREATE TABLE orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ulid TEXT NOT NULL,
		total INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		status TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	testData := []struct {
		ulid     string
		total    int
		quantity int
		status   string
	}{
		{"01ARYZ6S41TSV4RRFFQ69G5FA1", 100, 2, "completed"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA2", 200, 3, "completed"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA3", 150, 1, "pending"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA4", 50, 4, "cancelled"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA5", 300, 2, "completed"},
	}

	for _, d := range testData {
		_, err = driver.Exec(ctx, "INSERT INTO orders (ulid, total, quantity, status) VALUES (?, ?, ?, ?)",
			d.ulid, d.total, d.quantity, d.status)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	return driver
}

// TestAggregationHandler_Count_Integration tests Count with real database
func TestAggregationHandler_Count_Integration(t *testing.T) {
	driver := createTestDB(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger},
			{Name: "quantity", Type: registry.TypeInteger},
			{Name: "status", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "count all",
			url:            "/orders:count",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "count with filter",
			url:            "/orders:count?status[eq]=completed",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "count with total gt filter",
			url:            "/orders:count?total[gt]=100",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.Count(w, req, "orders")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAggregationHandler_Sum_Integration tests Sum with real database
func TestAggregationHandler_Sum_Integration(t *testing.T) {
	driver := createTestDB(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger},
			{Name: "quantity", Type: registry.TypeInteger},
			{Name: "status", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "sum all totals",
			url:            "/orders:sum?field=total",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "sum with filter",
			url:            "/orders:sum?field=total&status[eq]=completed",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "sum quantity",
			url:            "/orders:sum?field=quantity",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.Sum(w, req, "orders")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAggregationHandler_Avg_Integration tests Avg with real database
func TestAggregationHandler_Avg_Integration(t *testing.T) {
	driver := createTestDB(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger},
			{Name: "quantity", Type: registry.TypeInteger},
			{Name: "status", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "avg all totals",
			url:            "/orders:avg?field=total",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "avg with filter",
			url:            "/orders:avg?field=total&status[eq]=completed",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.Avg(w, req, "orders")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAggregationHandler_Min_Integration tests Min with real database
func TestAggregationHandler_Min_Integration(t *testing.T) {
	driver := createTestDB(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger},
			{Name: "quantity", Type: registry.TypeInteger},
			{Name: "status", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "min all totals",
			url:            "/orders:min?field=total",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "min with filter",
			url:            "/orders:min?field=total&status[eq]=completed",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.Min(w, req, "orders")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAggregationHandler_Max_Integration tests Max with real database
func TestAggregationHandler_Max_Integration(t *testing.T) {
	driver := createTestDB(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger},
			{Name: "quantity", Type: registry.TypeInteger},
			{Name: "status", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	handler := NewAggregationHandler(driver, reg)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "max all totals",
			url:            "/orders:max?field=total",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "max with filter",
			url:            "/orders:max?field=total&status[eq]=completed",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.Max(w, req, "orders")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAggregationHandler_InvalidFieldNotFound tests error when field doesn't exist
func TestAggregationHandler_InvalidFieldNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger},
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
			name:    "sum unknown field",
			handler: handler.Sum,
			url:     "/orders:sum?field=unknown",
		},
		{
			name:    "avg unknown field",
			handler: handler.Avg,
			url:     "/orders:avg?field=unknown",
		},
		{
			name:    "min unknown field",
			handler: handler.Min,
			url:     "/orders:min?field=unknown",
		},
		{
			name:    "max unknown field",
			handler: handler.Max,
			url:     "/orders:max?field=unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req, "orders")

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}
