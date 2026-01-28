package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// mockDriver is a mock implementation of database.Driver for testing
type mockDataDriver struct {
	dialect       database.DialectType
	execFunc      func(ctx context.Context, query string, args ...any) (sql.Result, error)
	queryFunc     func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	queryRowFunc  func(ctx context.Context, query string, args ...any) *sql.Row
	pingFunc      func(ctx context.Context) error
}

func (m *mockDataDriver) Connect(ctx context.Context) error         { return nil }
func (m *mockDataDriver) Close() error                               { return nil }
func (m *mockDataDriver) Dialect() database.DialectType              { return m.dialect }
func (m *mockDataDriver) DB() *sql.DB                                { return nil }
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

	// ID comes as float64 from JSON decoder
	idFloat, ok := response.Data["id"].(float64)
	if !ok {
		t.Fatalf("expected id to be float64, got %T", response.Data["id"])
	}
	if int64(idFloat) != int64(42) {
		t.Errorf("expected id 42, got %v", idFloat)
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
		ID: 42,
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
		ID: 999,
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
		ID: 42,
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
		ID: 999,
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
		"name":         "Test",
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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
		req := httptest.NewRequest(http.MethodGet, "/products:get?id=1", nil)
		w := httptest.NewRecorder()

		handler.Get(w, req, "products")

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		reqBody := UpdateDataRequest{
			ID: 1,
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
			ID: 1,
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
