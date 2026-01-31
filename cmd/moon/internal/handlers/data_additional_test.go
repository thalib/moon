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
	"github.com/thalib/moon/cmd/moon/internal/query"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// TestBuildSearchQueryWithFields_Extended tests the field selection and search query builder
func TestBuildSearchQueryWithFields_Extended(t *testing.T) {
	tests := []struct {
		name       string
		tableName  string
		fields     []string
		filters    []query.Condition
		searchSQL  string
		searchArgs []any
		orderBy    string
		limit      int
		dialect    database.DialectType
		wantSQL    string
	}{
		{
			name:       "select all with no filters",
			tableName:  "products",
			fields:     []string{},
			filters:    []query.Condition{},
			searchSQL:  "(name LIKE ?)",
			searchArgs: []any{"%test%"},
			orderBy:    "",
			limit:      10,
			dialect:    database.DialectSQLite,
			wantSQL:    "SELECT * FROM products WHERE (name LIKE ?) LIMIT ?",
		},
		{
			name:       "select specific fields",
			tableName:  "products",
			fields:     []string{"id", "name", "price"},
			filters:    []query.Condition{},
			searchSQL:  "(name LIKE ?)",
			searchArgs: []any{"%test%"},
			orderBy:    "",
			limit:      10,
			dialect:    database.DialectSQLite,
			wantSQL:    "SELECT id, name, price FROM products WHERE (name LIKE ?) LIMIT ?",
		},
		{
			name:      "with filters",
			tableName: "products",
			fields:    []string{},
			filters: []query.Condition{
				{Column: "price", Operator: ">", Value: 100},
			},
			searchSQL:  "(name LIKE ?)",
			searchArgs: []any{"%test%"},
			orderBy:    "",
			limit:      10,
			dialect:    database.DialectSQLite,
			wantSQL:    "SELECT * FROM products WHERE (name LIKE ?) AND price > ? LIMIT ?",
		},
		{
			name:       "with order by",
			tableName:  "products",
			fields:     []string{},
			filters:    []query.Condition{},
			searchSQL:  "(name LIKE ?)",
			searchArgs: []any{"%test%"},
			orderBy:    "price DESC",
			limit:      10,
			dialect:    database.DialectSQLite,
			wantSQL:    "SELECT * FROM products WHERE (name LIKE ?) ORDER BY price DESC LIMIT ?",
		},
		{
			name:       "postgres dialect with fields",
			tableName:  "products",
			fields:     []string{"id", "name"},
			filters:    []query.Condition{},
			searchSQL:  `("name" ILIKE $1)`,
			searchArgs: []any{"%test%"},
			orderBy:    "",
			limit:      10,
			dialect:    database.DialectPostgres,
			wantSQL:    `SELECT "id", "name" FROM "products" WHERE ("name" ILIKE $1) LIMIT $2`,
		},
		{
			name:       "mysql dialect with fields",
			tableName:  "products",
			fields:     []string{"id", "name"},
			filters:    []query.Condition{},
			searchSQL:  "(`name` LIKE ?)",
			searchArgs: []any{"%test%"},
			orderBy:    "",
			limit:      10,
			dialect:    database.DialectMySQL,
			wantSQL:    "SELECT `id`, `name` FROM `products` WHERE (`name` LIKE ?) LIMIT ?",
		},
		{
			name:      "with IN operator filter",
			tableName: "products",
			fields:    []string{},
			filters: []query.Condition{
				{Column: "status", Operator: query.OpIn, Value: []any{"active", "pending"}},
			},
			searchSQL:  "(name LIKE ?)",
			searchArgs: []any{"%test%"},
			orderBy:    "",
			limit:      10,
			dialect:    database.DialectSQLite,
			wantSQL:    "SELECT * FROM products WHERE (name LIKE ?) AND status IN (?, ?) LIMIT ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := buildSearchQueryWithFields(
				tt.tableName,
				tt.fields,
				tt.filters,
				tt.searchSQL,
				tt.searchArgs,
				tt.orderBy,
				tt.limit,
				tt.dialect,
			)

			if gotSQL != tt.wantSQL {
				t.Errorf("buildSearchQueryWithFields() SQL = %q, want %q", gotSQL, tt.wantSQL)
			}

			// Verify args count
			expectedArgCount := len(tt.searchArgs) + len(tt.filters)
			for _, f := range tt.filters {
				if f.Operator == query.OpIn {
					if values, ok := f.Value.([]any); ok {
						expectedArgCount += len(values) - 1 // subtract 1 because we already counted once
					}
				}
			}
			if tt.limit > 0 {
				expectedArgCount++
			}

			if len(gotArgs) != expectedArgCount {
				t.Errorf("buildSearchQueryWithFields() args count = %d, want %d", len(gotArgs), expectedArgCount)
			}
		})
	}
}

// TestDataHandler_Update_CollectionNotFound tests update when collection doesn't exist
func TestDataHandler_Update_CollectionNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := UpdateDataRequest{
		ID:   "01ARYZ6S41TSV4RRFFQ69G5FAV",
		Data: map[string]any{"name": "Test"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestDataHandler_Destroy_CollectionNotFound tests destroy when collection doesn't exist
func TestDataHandler_Destroy_CollectionNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := DestroyDataRequest{
		ID: "01ARYZ6S41TSV4RRFFQ69G5FAV",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Destroy(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestDataHandler_Get_CollectionNotFound tests get when collection doesn't exist
func TestDataHandler_Get_CollectionNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	req := httptest.NewRequest(http.MethodGet, "/products:get?id=01ARYZ6S41TSV4RRFFQ69G5FAV", nil)
	w := httptest.NewRecorder()

	handler.Get(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestDataHandler_List_CollectionNotFound tests list when collection doesn't exist
func TestDataHandler_List_CollectionNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	req := httptest.NewRequest(http.MethodGet, "/products:list", nil)
	w := httptest.NewRecorder()

	handler.List(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestDataHandler_Get_MissingID tests get when ID is missing
func TestDataHandler_Get_MissingID(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	req := httptest.NewRequest(http.MethodGet, "/products:get", nil)
	w := httptest.NewRecorder()

	handler.Get(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestDataHandler_Update_MissingID tests update when ID is missing
func TestDataHandler_Update_MissingID(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := UpdateDataRequest{
		ID:   "", // Missing ID
		Data: map[string]any{"name": "Test"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestDataHandler_Destroy_MissingID tests destroy when ID is missing
func TestDataHandler_Destroy_MissingID(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	reqBody := DestroyDataRequest{
		ID: "", // Missing ID
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Destroy(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestDataHandler_Create_InvalidJSON tests create with invalid JSON body
func TestDataHandler_Create_InvalidJSON(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestDataHandler_Update_InvalidJSON tests update with invalid JSON body
func TestDataHandler_Update_InvalidJSON(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	req := httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestDataHandler_Destroy_InvalidJSON tests destroy with invalid JSON body
func TestDataHandler_Destroy_InvalidJSON(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg)

	req := httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.Destroy(w, req, "products")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Mock for testing database errors
type mockDBErrorDriver struct {
	mockDataDriver
}

func (m *mockDBErrorDriver) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, sql.ErrConnDone
}

func (m *mockDBErrorDriver) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, sql.ErrConnDone
}

// TestDataHandler_Create_DatabaseError tests create when database returns an error
func TestDataHandler_Create_DatabaseError(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}
	reg.Set(collection)

	driver := &mockDBErrorDriver{
		mockDataDriver: mockDataDriver{dialect: database.DialectSQLite},
	}
	handler := NewDataHandler(driver, reg)

	reqBody := CreateDataRequest{
		Data: map[string]any{"name": "Test Product"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
