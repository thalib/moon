package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/query"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// mockDriver is a mock implementation of database.Driver for testing
type mockDataDriver struct {
	dialect      database.DialectType
	execFunc     func(ctx context.Context, query string, args ...any) (sql.Result, error)
	queryFunc    func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	queryRowFunc func(ctx context.Context, query string, args ...any) *sql.Row
	pingFunc     func(ctx context.Context) error
}

func (m *mockDataDriver) Connect(ctx context.Context) error            { return nil }
func (m *mockDataDriver) Close() error                                 { return nil }
func (m *mockDataDriver) Dialect() database.DialectType                { return m.dialect }
func (m *mockDataDriver) DB() *sql.DB                                  { return nil }
func (m *mockDataDriver) BeginTx(ctx context.Context) (*sql.Tx, error) { return nil, nil }
func (m *mockDataDriver) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func (m *mockDataDriver) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return mockResult{lastInsertID: 1, rowsAffected: 1}, nil
}

func (m *mockDataDriver) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDataDriver) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return nil
}

func (m *mockDataDriver) ListTables(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockDataDriver) GetTableInfo(ctx context.Context, tableName string) (*database.TableInfo, error) {
	return nil, nil
}

func (m *mockDataDriver) TableExists(ctx context.Context, tableName string) (bool, error) {
	return false, nil
}

// mockResult implements sql.Result
type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m mockResult) LastInsertId() (int64, error) { return m.lastInsertID, nil }
func (m mockResult) RowsAffected() (int64, error) { return m.rowsAffected, nil }

func TestDataHandler_Create_CollectionNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := CreateDataRequest{
		Data: map[string]any{
			"name": "Test",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDataHandler_Create_Success(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeFloat, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{
		dialect: database.DialectSQLite,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return mockResult{lastInsertID: 42, rowsAffected: 1}, nil
		},
	}
	handler := NewDataHandler(driver, reg)

	reqBody := CreateDataRequest{
		Data: map[string]any{
			"name":  "Test Product",
			"price": 19.99,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response CreateDataResponse
	json.NewDecoder(w.Body).Decode(&response)

	// ID is now a ULID string
	idStr, ok := response.Data["id"].(string)
	if !ok {
		t.Fatalf("expected id to be string, got %T", response.Data["id"])
	}
	if len(idStr) != 26 {
		t.Errorf("expected ULID (26 chars), got %d chars: %v", len(idStr), idStr)
	}
}

func TestDataHandler_Create_MissingRequiredField(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeFloat, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := CreateDataRequest{
		Data: map[string]any{
			"name": "Test Product",
			// missing required field "price"
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDataHandler_Create_InvalidFieldType(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeFloat, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := CreateDataRequest{
		Data: map[string]any{
			"name":  "Test Product",
			"price": "not a number", // wrong type
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDataHandler_Update_Success(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeFloat, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{
		dialect: database.DialectSQLite,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return mockResult{rowsAffected: 1}, nil
		},
	}
	handler := NewDataHandler(driver, reg)

	reqBody := UpdateDataRequest{
		ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Data: map[string]any{
			"name":  "Updated Product",
			"price": 29.99,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDataHandler_Update_NotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{
		dialect: database.DialectSQLite,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return mockResult{rowsAffected: 0}, nil
		},
	}
	handler := NewDataHandler(driver, reg)

	reqBody := UpdateDataRequest{
		ID: "01ARZ3NDEKTSV4RRFFQ69G5FBX",
		Data: map[string]any{
			"name": "Updated Product",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDataHandler_Destroy_Success(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name:    "products",
		Columns: []registry.Column{},
	}
	reg.Set(collection)

	driver := &mockDataDriver{
		dialect: database.DialectSQLite,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return mockResult{rowsAffected: 1}, nil
		},
	}
	handler := NewDataHandler(driver, reg)

	reqBody := DestroyDataRequest{
		ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Destroy(w, req, "products")

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDataHandler_Destroy_NotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name:    "products",
		Columns: []registry.Column{},
	}
	reg.Set(collection)

	driver := &mockDataDriver{
		dialect: database.DialectSQLite,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return mockResult{rowsAffected: 0}, nil
		},
	}
	handler := NewDataHandler(driver, reg)

	reqBody := DestroyDataRequest{
		ID: "01ARZ3NDEKTSV4RRFFQ69G5FBX",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Destroy(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestValidateFields_UnknownField(t *testing.T) {
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}

	data := map[string]any{
		"name":          "Test",
		"unknown_field": "value",
	}

	err := validateFields(data, collection)
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestValidateFields_ValidData(t *testing.T) {
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
			{Name: "price", Type: registry.TypeFloat},
			{Name: "available", Type: registry.TypeBoolean},
		},
	}

	data := map[string]any{
		"name":      "Test",
		"price":     19.99,
		"available": true,
	}

	err := validateFields(data, collection)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFieldType(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		value       any
		columnType  registry.ColumnType
		expectError bool
	}{
		{
			name:        "valid string",
			fieldName:   "name",
			value:       "test",
			columnType:  registry.TypeString,
			expectError: false,
		},
		{
			name:        "invalid string",
			fieldName:   "name",
			value:       123,
			columnType:  registry.TypeString,
			expectError: true,
		},
		{
			name:        "valid integer",
			fieldName:   "count",
			value:       42,
			columnType:  registry.TypeInteger,
			expectError: false,
		},
		{
			name:        "valid integer as float64 (from JSON)",
			fieldName:   "count",
			value:       float64(42),
			columnType:  registry.TypeInteger,
			expectError: false,
		},
		{
			name:        "invalid integer",
			fieldName:   "count",
			value:       "not a number",
			columnType:  registry.TypeInteger,
			expectError: true,
		},
		{
			name:        "valid float",
			fieldName:   "price",
			value:       19.99,
			columnType:  registry.TypeFloat,
			expectError: false,
		},
		{
			name:        "invalid float",
			fieldName:   "price",
			value:       "not a number",
			columnType:  registry.TypeFloat,
			expectError: true,
		},
		{
			name:        "valid boolean",
			fieldName:   "active",
			value:       true,
			columnType:  registry.TypeBoolean,
			expectError: false,
		},
		{
			name:        "invalid boolean",
			fieldName:   "active",
			value:       "yes",
			columnType:  registry.TypeBoolean,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldType(tt.fieldName, tt.value, tt.columnType)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDataHandler_PostgresDialect(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	queryCalled := false
	driver := &mockDataDriver{
		dialect: database.DialectPostgres,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			queryCalled = true
			// Verify PostgreSQL style placeholders
			if !bytes.Contains([]byte(query), []byte("$")) {
				t.Errorf("expected PostgreSQL style placeholder ($), got query: %s", query)
			}
			return mockResult{lastInsertID: 1, rowsAffected: 1}, nil
		},
	}
	handler := NewDataHandler(driver, reg)

	reqBody := CreateDataRequest{
		Data: map[string]any{
			"name": "Test",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if !queryCalled {
		t.Error("expected exec to be called")
	}
}

func TestDataHandler_Create_UnknownField(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := CreateDataRequest{
		Data: map[string]any{
			"name":          "Test",
			"unknown_field": "value",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Integration test with real SQLite database
func TestDataHandler_Integration_SQLite(t *testing.T) {
	// Create in-memory SQLite database
	dbConfig := database.Config{
		ConnectionString: ":memory:",
		MaxOpenConns:     5,
		MaxIdleConns:     2,
	}

	driver, err := database.NewDriver(dbConfig)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer driver.Close()

	// Create a test table
	createTableSQL := `
		CREATE TABLE products (
			ulid TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			price REAL NOT NULL
		)
	`
	if _, err := driver.Exec(ctx, createTableSQL); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Setup registry
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeFloat, Nullable: false},
		},
	}
	reg.Set(collection)

	handler := NewDataHandler(driver, reg)

	var createdULID string

	// Test Create
	t.Run("Create", func(t *testing.T) {
		reqBody := CreateDataRequest{
			Data: map[string]any{
				"name":  "Test Product",
				"price": 19.99,
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Create(w, req, "products")

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var response CreateDataResponse
		json.NewDecoder(w.Body).Decode(&response)
		if idStr, ok := response.Data["id"].(string); ok {
			createdULID = idStr
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:list", nil)
		w := httptest.NewRecorder()

		handler.List(w, req, "products")

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response DataListResponse
		json.NewDecoder(w.Body).Decode(&response)

		if len(response.Data) == 0 {
			t.Error("expected at least one record")
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:get?id="+createdULID, nil)
		w := httptest.NewRecorder()

		handler.Get(w, req, "products")

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		reqBody := UpdateDataRequest{
			ID: createdULID,
			Data: map[string]any{
				"name":  "Updated Product",
				"price": 29.99,
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Update(w, req, "products")

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	// Test Destroy
	t.Run("Destroy", func(t *testing.T) {
		reqBody := DestroyDataRequest{
			ID: createdULID,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Destroy(w, req, "products")

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})
}

// PRD 021: Advanced Filtering Tests

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedCount int
		expectedOps   map[string]string
	}{
		{
			name:          "No filters",
			url:           "/products:list",
			expectedCount: 0,
			expectedOps:   map[string]string{},
		},
		{
			name:          "Single eq filter",
			url:           "/products:list?status[eq]=active",
			expectedCount: 1,
			expectedOps:   map[string]string{"status": "eq"},
		},
		{
			name:          "Multiple filters",
			url:           "/products:list?price[gt]=100&category[eq]=electronics",
			expectedCount: 2,
			expectedOps:   map[string]string{"price": "gt", "category": "eq"},
		},
		{
			name:          "All operator types",
			url:           "/products:list?a[eq]=1&b[ne]=2&c[gt]=3&d[lt]=4&e[gte]=5&f[lte]=6&g[like]=test&h[in]=1,2,3",
			expectedCount: 8,
			expectedOps:   map[string]string{"a": "eq", "b": "ne", "c": "gt", "d": "lt", "e": "gte", "f": "lte", "g": "like", "h": "in"},
		},
		{
			name:          "Filter with standard params",
			url:           "/products:list?limit=10&price[gt]=100&after=abc",
			expectedCount: 1,
			expectedOps:   map[string]string{"price": "gt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			filters, err := parseFilters(req)
			if err != nil {
				t.Fatalf("parseFilters() error = %v", err)
			}

			if len(filters) != tt.expectedCount {
				t.Errorf("expected %d filters, got %d", tt.expectedCount, len(filters))
			}

			for _, filter := range filters {
				if expectedOp, exists := tt.expectedOps[filter.column]; exists {
					if filter.operator != expectedOp {
						t.Errorf("column %s: expected operator %s, got %s", filter.column, expectedOp, filter.operator)
					}
				}
			}
		})
	}
}

func TestMapOperatorToSQL(t *testing.T) {
	tests := []struct {
		shortOp string
		sqlOp   string
	}{
		{"eq", "="},
		{"ne", "!="},
		{"gt", ">"},
		{"lt", "<"},
		{"gte", ">="},
		{"lte", "<="},
		{"like", "LIKE"},
		{"in", "IN"},
	}

	for _, tt := range tests {
		t.Run(tt.shortOp, func(t *testing.T) {
			result := mapOperatorToSQL(tt.shortOp)
			if result != tt.sqlOp {
				t.Errorf("mapOperatorToSQL(%s) = %s, want %s", tt.shortOp, result, tt.sqlOp)
			}
		})
	}
}

func TestBuildConditions(t *testing.T) {
	// Create test collection
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
			{Name: "price", Type: registry.TypeFloat},
			{Name: "stock", Type: registry.TypeInteger},
			{Name: "active", Type: registry.TypeBoolean},
		},
	}

	tests := []struct {
		name        string
		filters     []filterParam
		wantErr     bool
		errContains string
		checkFunc   func(*testing.T, []any)
	}{
		{
			name: "Valid string filter",
			filters: []filterParam{
				{column: "name", operator: "eq", value: "Product A"},
			},
			wantErr: false,
		},
		{
			name: "Valid integer filter",
			filters: []filterParam{
				{column: "stock", operator: "gt", value: "100"},
			},
			wantErr: false,
		},
		{
			name: "Valid float filter",
			filters: []filterParam{
				{column: "price", operator: "lte", value: "99.99"},
			},
			wantErr: false,
		},
		{
			name: "Valid boolean filter",
			filters: []filterParam{
				{column: "active", operator: "eq", value: "true"},
			},
			wantErr: false,
		},
		{
			name: "Invalid column",
			filters: []filterParam{
				{column: "nonexistent", operator: "eq", value: "test"},
			},
			wantErr:     true,
			errContains: "invalid filter column",
		},
		{
			name: "Invalid integer value",
			filters: []filterParam{
				{column: "stock", operator: "gt", value: "not_a_number"},
			},
			wantErr:     true,
			errContains: "invalid value",
		},
		{
			name: "LIKE operator",
			filters: []filterParam{
				{column: "name", operator: "like", value: "moon"},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, conditions []any) {
				// LIKE should wrap value with %
				// This will be checked in integration tests
			},
		},
		{
			name: "IN operator",
			filters: []filterParam{
				{column: "name", operator: "in", value: "a,b,c"},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, conditions []any) {
				// IN should parse comma-separated values
				// This will be checked in integration tests
			},
		},
		{
			name: "Multiple filters",
			filters: []filterParam{
				{column: "price", operator: "gt", value: "50"},
				{column: "stock", operator: "lt", value: "1000"},
				{column: "name", operator: "like", value: "product"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, err := buildConditions(tt.filters, collection)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(conditions) != len(tt.filters) {
				t.Errorf("expected %d conditions, got %d", len(tt.filters), len(conditions))
			}

			if tt.checkFunc != nil {
				vals := make([]any, len(conditions))
				for i, c := range conditions {
					vals[i] = c.Value
				}
				tt.checkFunc(t, vals)
			}
		})
	}
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		colType  registry.ColumnType
		expected any
		wantErr  bool
	}{
		{
			name:     "String type",
			value:    "test",
			colType:  registry.TypeString,
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "Integer type - valid",
			value:    "123",
			colType:  registry.TypeInteger,
			expected: int64(123),
			wantErr:  false,
		},
		{
			name:    "Integer type - invalid",
			value:   "not_int",
			colType: registry.TypeInteger,
			wantErr: true,
		},
		{
			name:     "Float type - valid",
			value:    "123.45",
			colType:  registry.TypeFloat,
			expected: float64(123.45),
			wantErr:  false,
		},
		{
			name:    "Float type - invalid",
			value:   "not_float",
			colType: registry.TypeFloat,
			wantErr: true,
		},
		{
			name:     "Boolean type - true",
			value:    "true",
			colType:  registry.TypeBoolean,
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Boolean type - false",
			value:    "false",
			colType:  registry.TypeBoolean,
			expected: false,
			wantErr:  false,
		},
		{
			name:    "Boolean type - invalid",
			value:   "not_bool",
			colType: registry.TypeBoolean,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValue(tt.value, tt.colType)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// PRD 022: Sorting Support Tests

func TestParseSort(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedCount int
		expectedSorts []sortField
	}{
		{
			name:          "No sort parameter",
			url:           "/products:list",
			expectedCount: 0,
			expectedSorts: []sortField{},
		},
		{
			name:          "Single field ascending",
			url:           "/products:list?sort=price",
			expectedCount: 1,
			expectedSorts: []sortField{{column: "price", direction: "ASC"}},
		},
		{
			name:          "Single field descending",
			url:           "/products:list?sort=-price",
			expectedCount: 1,
			expectedSorts: []sortField{{column: "price", direction: "DESC"}},
		},
		{
			name:          "Single field explicit ascending",
			url:           "/products:list?sort=+price",
			expectedCount: 1,
			expectedSorts: []sortField{{column: "price", direction: "ASC"}},
		},
		{
			name:          "Multiple fields",
			url:           "/products:list?sort=-created_at,name",
			expectedCount: 2,
			expectedSorts: []sortField{
				{column: "created_at", direction: "DESC"},
				{column: "name", direction: "ASC"},
			},
		},
		{
			name:          "Multiple fields mixed directions",
			url:           "/products:list?sort=category,-price,+name",
			expectedCount: 3,
			expectedSorts: []sortField{
				{column: "category", direction: "ASC"},
				{column: "price", direction: "DESC"},
				{column: "name", direction: "ASC"},
			},
		},
		{
			name:          "Sort with spaces",
			url:           "/products:list?sort=-price,name",
			expectedCount: 2,
			expectedSorts: []sortField{
				{column: "price", direction: "DESC"},
				{column: "name", direction: "ASC"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			sorts, err := parseSort(req)
			if err != nil {
				t.Fatalf("parseSort() error = %v", err)
			}

			if len(sorts) != tt.expectedCount {
				t.Errorf("expected %d sort fields, got %d", tt.expectedCount, len(sorts))
			}

			for i, expected := range tt.expectedSorts {
				if i >= len(sorts) {
					t.Errorf("missing sort field at index %d", i)
					continue
				}
				if sorts[i].column != expected.column {
					t.Errorf("sort[%d].column: expected %s, got %s", i, expected.column, sorts[i].column)
				}
				if sorts[i].direction != expected.direction {
					t.Errorf("sort[%d].direction: expected %s, got %s", i, expected.direction, sorts[i].direction)
				}
			}
		})
	}
}

func TestBuildOrderBy(t *testing.T) {
	// Create test collection
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
			{Name: "price", Type: registry.TypeFloat},
			{Name: "created_at", Type: registry.TypeDatetime},
		},
	}

	tests := []struct {
		name        string
		sorts       []sortField
		dialect     database.DialectType
		expected    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "No sorts - default",
			sorts:    []sortField{},
			dialect:  database.DialectSQLite,
			expected: "ulid ASC",
			wantErr:  false,
		},
		{
			name: "Single field ascending - SQLite",
			sorts: []sortField{
				{column: "price", direction: "ASC"},
			},
			dialect:  database.DialectSQLite,
			expected: "price ASC",
			wantErr:  false,
		},
		{
			name: "Single field descending - SQLite",
			sorts: []sortField{
				{column: "price", direction: "DESC"},
			},
			dialect:  database.DialectSQLite,
			expected: "price DESC",
			wantErr:  false,
		},
		{
			name: "Single field - Postgres",
			sorts: []sortField{
				{column: "price", direction: "ASC"},
			},
			dialect:  database.DialectPostgres,
			expected: `"price" ASC`,
			wantErr:  false,
		},
		{
			name: "Single field - MySQL",
			sorts: []sortField{
				{column: "price", direction: "DESC"},
			},
			dialect:  database.DialectMySQL,
			expected: "`price` DESC",
			wantErr:  false,
		},
		{
			name: "Multiple fields",
			sorts: []sortField{
				{column: "created_at", direction: "DESC"},
				{column: "name", direction: "ASC"},
			},
			dialect:  database.DialectSQLite,
			expected: "created_at DESC, name ASC",
			wantErr:  false,
		},
		{
			name: "Sort by ulid",
			sorts: []sortField{
				{column: "ulid", direction: "DESC"},
			},
			dialect:  database.DialectSQLite,
			expected: "ulid DESC",
			wantErr:  false,
		},
		{
			name: "Invalid column",
			sorts: []sortField{
				{column: "nonexistent", direction: "ASC"},
			},
			dialect:     database.DialectSQLite,
			wantErr:     true,
			errContains: "invalid sort column",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := query.NewBuilder(tt.dialect)
			orderBy, err := buildOrderBy(tt.sorts, collection, builder)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if orderBy != tt.expected {
				t.Errorf("expected ORDER BY %q, got %q", tt.expected, orderBy)
			}
		})
	}
}

// PRD 023: Full-Text Search Tests

func TestBuildSearchConditions_Basic(t *testing.T) {
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
			{Name: "description", Type: registry.TypeText},
		},
	}

	sql, args := buildSearchConditions("laptop", collection, database.DialectSQLite)

	if sql == "" {
		t.Error("expected non-empty SQL")
	}

	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}

	for _, arg := range args {
		str := arg.(string)
		if str != "%laptop%" {
			t.Errorf("expected %%laptop%%, got %s", str)
		}
	}

	if !contains(sql, " OR ") {
		t.Error("expected OR operator")
	}
}

func TestBuildSearchConditions_NoTextColumns(t *testing.T) {
	collection := &registry.Collection{
		Name: "numbers",
		Columns: []registry.Column{
			{Name: "price", Type: registry.TypeFloat},
		},
	}

	sql, _ := buildSearchConditions("test", collection, database.DialectSQLite)

	if sql != "" {
		t.Errorf("expected empty SQL for non-text columns, got %s", sql)
	}
}

// PRD 024: Field Selection Tests

func TestParseFields(t *testing.T) {
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
			{Name: "price", Type: registry.TypeFloat},
			{Name: "stock", Type: registry.TypeInteger},
		},
	}

	tests := []struct {
		name        string
		url         string
		wantErr     bool
		errContains string
		checkFields func(*testing.T, []string)
	}{
		{
			name:    "No fields parameter - select all",
			url:     "/products:list",
			wantErr: false,
			checkFields: func(t *testing.T, fields []string) {
				if fields != nil {
					t.Error("expected nil fields for select all")
				}
			},
		},
		{
			name:    "Single field",
			url:     "/products:list?fields=name",
			wantErr: false,
			checkFields: func(t *testing.T, fields []string) {
				if len(fields) != 2 { // name + ulid
					t.Errorf("expected 2 fields (name + ulid), got %d", len(fields))
				}
			},
		},
		{
			name:    "Multiple fields",
			url:     "/products:list?fields=name,price",
			wantErr: false,
			checkFields: func(t *testing.T, fields []string) {
				if len(fields) != 3 { // name, price + ulid
					t.Errorf("expected 3 fields, got %d", len(fields))
				}
			},
		},
		{
			name:    "Field with spaces",
			url:     "/products:list?fields=name,price,stock",
			wantErr: false,
			checkFields: func(t *testing.T, fields []string) {
				if len(fields) != 4 { // name, price, stock + ulid
					t.Errorf("expected 4 fields, got %d", len(fields))
				}
			},
		},
		{
			name:    "Always includes ulid",
			url:     "/products:list?fields=name",
			wantErr: false,
			checkFields: func(t *testing.T, fields []string) {
				hasUlid := false
				for _, f := range fields {
					if f == "ulid" {
						hasUlid = true
						break
					}
				}
				if !hasUlid {
					t.Error("ulid should always be included")
				}
			},
		},
		{
			name:        "Invalid field",
			url:         "/products:list?fields=nonexistent",
			wantErr:     true,
			errContains: "invalid field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			fields, err := parseFields(req, collection)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkFields != nil {
				tt.checkFields(t, fields)
			}
		})
	}
}

// PRD 038: Cursor Pagination Tests - Fix skip bug
func TestDataHandler_CursorPagination(t *testing.T) {
	// Create in-memory SQLite database
	dbConfig := database.Config{
		ConnectionString: ":memory:",
		MaxOpenConns:     5,
		MaxIdleConns:     2,
	}

	driver, err := database.NewDriver(dbConfig)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer driver.Close()

	// Create a test table
	createTableSQL := `
CREATE TABLE test_pagination (
ulid TEXT PRIMARY KEY,
name TEXT NOT NULL
)
`
	if _, err := driver.Exec(ctx, createTableSQL); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Setup registry
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "test_pagination",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	handler := NewDataHandler(driver, reg)

	// Insert 5 test records
	recordIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		reqBody := CreateDataRequest{
			Data: map[string]any{
				"name": fmt.Sprintf("Record %d", i+1),
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/test_pagination:create", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Create(w, req, "test_pagination")

		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create record %d: status %d, body: %s", i+1, w.Code, w.Body.String())
		}

		var response CreateDataResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode create response: %v", err)
		}
		if idStr, ok := response.Data["id"].(string); ok {
			recordIDs[i] = idStr
		} else {
			t.Fatalf("failed to get ID from create response")
		}
	}

	t.Run("Paginate with limit=1 through all 5 records", func(t *testing.T) {
		retrievedRecords := make([]map[string]any, 0, 5)
		var cursor *string

		// Fetch all records using pagination with limit=1
		for pageNum := 1; pageNum <= 5; pageNum++ {
			url := "/test_pagination:list?limit=1"
			if cursor != nil {
				url += "&after=" + *cursor
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.List(w, req, "test_pagination")

			if w.Code != http.StatusOK {
				t.Fatalf("page %d: expected status %d, got %d. Body: %s", pageNum, http.StatusOK, w.Code, w.Body.String())
			}

			var response DataListResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("page %d: failed to decode response: %v", pageNum, err)
			}

			// Verify we got exactly 1 record per page
			if len(response.Data) != 1 {
				t.Fatalf("page %d: expected 1 record, got %d", pageNum, len(response.Data))
			}

			// Store the retrieved record
			retrievedRecords = append(retrievedRecords, response.Data[0])

			// Verify cursor behavior
			if pageNum < 5 {
				// Not the last page - should have a cursor
				if response.NextCursor == nil {
					t.Fatalf("page %d: expected next_cursor, got nil", pageNum)
				}
				// Verify cursor points to the ID of the record we just retrieved
				if recordID, ok := response.Data[0]["id"].(string); ok {
					if *response.NextCursor != recordID {
						t.Errorf("page %d: cursor should be ID of last returned record (%s), got %s",
							pageNum, recordID, *response.NextCursor)
					}
				}
				cursor = response.NextCursor
			} else {
				// Last page - should NOT have a cursor
				if response.NextCursor != nil {
					t.Errorf("page %d (last page): expected nil cursor, got %s", pageNum, *response.NextCursor)
				}
			}
		}

		// Verify we retrieved exactly 5 records
		if len(retrievedRecords) != 5 {
			t.Errorf("expected to retrieve 5 records total, got %d", len(retrievedRecords))
		}

		// Verify no duplicates (check all IDs are unique)
		seenIDs := make(map[string]bool)
		for i, record := range retrievedRecords {
			if idStr, ok := record["id"].(string); ok {
				if seenIDs[idStr] {
					t.Errorf("duplicate record found at position %d: ID %s", i, idStr)
				}
				seenIDs[idStr] = true
			}
		}

		// Verify all created records were retrieved
		for _, expectedID := range recordIDs {
			if !seenIDs[expectedID] {
				t.Errorf("record with ID %s was not retrieved during pagination", expectedID)
			}
		}
	})

	t.Run("Edge case: Single record with limit=1", func(t *testing.T) {
		// Create new table for this test
		createSQL := `
CREATE TABLE test_single (
ulid TEXT PRIMARY KEY,
name TEXT NOT NULL
)
`
		if _, err := driver.Exec(ctx, createSQL); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}

		singleCollection := &registry.Collection{
			Name: "test_single",
			Columns: []registry.Column{
				{Name: "name", Type: registry.TypeString, Nullable: false},
			},
		}
		reg.Set(singleCollection)

		// Insert one record
		reqBody := CreateDataRequest{
			Data: map[string]any{
				"name": "Single Record",
			},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/test_single:create", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Create(w, req, "test_single")

		// List with limit=1
		req = httptest.NewRequest(http.MethodGet, "/test_single:list?limit=1", nil)
		w = httptest.NewRecorder()
		handler.List(w, req, "test_single")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response DataListResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(response.Data) != 1 {
			t.Errorf("expected 1 record, got %d", len(response.Data))
		}

		// Should have null cursor (no more records)
		if response.NextCursor != nil {
			t.Errorf("expected nil cursor for single record, got %s", *response.NextCursor)
		}
	})

	t.Run("Edge case: Empty collection", func(t *testing.T) {
		// Create new empty table
		createSQL := `
CREATE TABLE test_empty (
ulid TEXT PRIMARY KEY,
name TEXT NOT NULL
)
`
		if _, err := driver.Exec(ctx, createSQL); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}

		emptyCollection := &registry.Collection{
			Name: "test_empty",
			Columns: []registry.Column{
				{Name: "name", Type: registry.TypeString, Nullable: false},
			},
		}
		reg.Set(emptyCollection)

		// List empty collection
		req := httptest.NewRequest(http.MethodGet, "/test_empty:list?limit=1", nil)
		w := httptest.NewRecorder()
		handler.List(w, req, "test_empty")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response DataListResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(response.Data) != 0 {
			t.Errorf("expected 0 records, got %d", len(response.Data))
		}

		// Should have null cursor
		if response.NextCursor != nil {
			t.Errorf("expected nil cursor for empty collection, got %s", *response.NextCursor)
		}
	})

	t.Run("Edge case: Exactly limit records", func(t *testing.T) {
		// Create new table for this test
		createSQL := `
CREATE TABLE test_exact (
ulid TEXT PRIMARY KEY,
name TEXT NOT NULL
)
`
		if _, err := driver.Exec(ctx, createSQL); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}

		exactCollection := &registry.Collection{
			Name: "test_exact",
			Columns: []registry.Column{
				{Name: "name", Type: registry.TypeString, Nullable: false},
			},
		}
		reg.Set(exactCollection)

		// Insert exactly 2 records
		for i := 0; i < 2; i++ {
			reqBody := CreateDataRequest{
				Data: map[string]any{
					"name": fmt.Sprintf("Exact Record %d", i+1),
				},
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/test_exact:create", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handler.Create(w, req, "test_exact")

			if w.Code != http.StatusCreated {
				t.Fatalf("failed to create record: %v", w.Body.String())
			}
		}

		// List with limit=2 (exactly the number of records)
		req := httptest.NewRequest(http.MethodGet, "/test_exact:list?limit=2", nil)
		w := httptest.NewRecorder()
		handler.List(w, req, "test_exact")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response DataListResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(response.Data) != 2 {
			t.Errorf("expected 2 records, got %d", len(response.Data))
		}

		// Should have null cursor (no more records)
		if response.NextCursor != nil {
			t.Errorf("expected nil cursor when fetching exact number of records, got %s", *response.NextCursor)
		}
	})

	t.Run("Paginate with limit=2 through 5 records", func(t *testing.T) {
		retrievedRecords := make([]map[string]any, 0, 5)
		var cursor *string
		pageNum := 0

		// Fetch all 5 records using pagination with limit=2
		for {
			pageNum++
			url := "/test_pagination:list?limit=2"
			if cursor != nil {
				url += "&after=" + *cursor
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.List(w, req, "test_pagination")

			if w.Code != http.StatusOK {
				t.Fatalf("page %d: expected status %d, got %d", pageNum, http.StatusOK, w.Code)
			}

			var response DataListResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("page %d: failed to decode response: %v", pageNum, err)
			}

			if len(response.Data) == 0 {
				break
			}

			retrievedRecords = append(retrievedRecords, response.Data...)

			if response.NextCursor == nil {
				break
			}
			cursor = response.NextCursor
		}

		// Verify we retrieved exactly 5 records
		if len(retrievedRecords) != 5 {
			t.Errorf("expected to retrieve 5 records total, got %d", len(retrievedRecords))
		}

		// Verify no duplicates
		seenIDs := make(map[string]bool)
		for _, record := range retrievedRecords {
			if idStr, ok := record["id"].(string); ok {
				if seenIDs[idStr] {
					t.Errorf("duplicate record found: ID %s", idStr)
				}
				seenIDs[idStr] = true
			}
		}
	})
}
