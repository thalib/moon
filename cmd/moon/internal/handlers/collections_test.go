package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

func setupTestHandler(t *testing.T) (*CollectionsHandler, database.Driver) {
	config := database.Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := database.NewDriver(config)
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	return handler, driver
}

func TestNewCollectionsHandler(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	if handler == nil {
		t.Fatal("NewCollectionsHandler returned nil")
	}
}

func TestList_Empty(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections:list", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response ListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 0 {
		t.Errorf("Expected count 0, got %d", response.Count)
	}
}

// TestList_WithCollectionsAndRecords tests List with collections containing records (PRD-065)
func TestList_WithCollectionsAndRecords(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "customers",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Now list collections
	req = httptest.NewRequest(http.MethodGet, "/api/v1/collections:list", nil)
	w = httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response ListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	if response.Count != 1 {
		t.Errorf("Expected count 1, got %d", response.Count)
	}

	if len(response.Collections) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(response.Collections))
	}

	// Verify collection item structure
	collection := response.Collections[0]
	if collection.Name != "customers" {
		t.Errorf("Expected collection name 'customers', got '%s'", collection.Name)
	}

	// Records should be 0 as we haven't inserted any
	if collection.Records != 0 {
		t.Errorf("Expected 0 records, got %d", collection.Records)
	}
}

func TestGet_NotFound(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections:get?name=nonexistent", nil)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGet_MissingName(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections:get", nil)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreate_Success(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	createReq := CreateRequest{
		Name: "customers",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false, Unique: true},
			{Name: "age", Type: registry.TypeInteger, Nullable: true},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	var response CreateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Collection.Name != "customers" {
		t.Errorf("Expected collection name 'customers', got '%s'", response.Collection.Name)
	}

	if len(response.Collection.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(response.Collection.Columns))
	}
}

func TestCreate_InvalidName(t *testing.T) {
	tests := []struct {
		name        string
		collName    string
		expectError bool
	}{
		{"empty name", "", true},
		{"reserved word", "select", true},
		{"invalid chars", "test-table", true},
		{"starts with number", "1users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new handler for each test case to avoid conflicts
			handler, driver := setupTestHandler(t)
			defer driver.Close()

			createReq := CreateRequest{
				Name: tt.collName,
				Columns: []registry.Column{
					{Name: "id", Type: registry.TypeInteger},
				},
			}

			body, _ := json.Marshal(createReq)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if tt.expectError && w.Code == http.StatusCreated {
				t.Errorf("Expected error for '%s', but got success", tt.collName)
			}

			if !tt.expectError && w.Code != http.StatusCreated {
				t.Errorf("Expected success for '%s', but got error: %d", tt.collName, w.Code)
			}
		})
	}
}

func TestCreate_NoColumns(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	createReq := CreateRequest{
		Name:    "empty_table",
		Columns: []registry.Column{},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreate_InvalidColumnType(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	createReq := CreateRequest{
		Name: "test_table",
		Columns: []registry.Column{
			{Name: "field1", Type: "invalid_type"},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreate_AlreadyExists(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create first time
	createReq := CreateRequest{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Try to create again
	body, _ = json.Marshal(createReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status code %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestUpdate_AddColumns(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create collection first
	createReq := CreateRequest{
		Name: "orders",
		Columns: []registry.Column{
			{Name: "customer", Type: registry.TypeString},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Update to add columns
	updateReq := UpdateRequest{
		Name: "orders",
		AddColumns: []registry.Column{
			{Name: "total", Type: registry.TypeInteger, Nullable: false},
			{Name: "notes", Type: registry.TypeString, Nullable: true},
		},
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response UpdateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Collection.Columns) != 3 {
		t.Errorf("Expected 3 columns after update, got %d", len(response.Collection.Columns))
	}
}

func TestUpdate_NotFound(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	updateReq := UpdateRequest{
		Name: "nonexistent",
		AddColumns: []registry.Column{
			{Name: "field", Type: registry.TypeString},
		},
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDestroy_Success(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create collection first
	createReq := CreateRequest{
		Name: "temp_table",
		Columns: []registry.Column{
			{Name: "data", Type: registry.TypeString},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Destroy it
	destroyReq := DestroyRequest{
		Name: "temp_table",
	}

	body, _ = json.Marshal(destroyReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:destroy", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Destroy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Verify it's gone from registry
	if handler.registry.Exists("temp_table") {
		t.Error("Collection should not exist after destroy")
	}
}

func TestDestroy_NotFound(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	destroyReq := DestroyRequest{
		Name: "nonexistent",
	}

	body, _ := json.Marshal(destroyReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:destroy", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestValidateCollectionName tests collection name validation (PRD-047, PRD-048)
func TestValidateCollectionName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errContains string // substring expected in error
	}{
		// Valid cases
		{"valid name", "products", false, ""},
		{"valid with underscore", "user_profiles", false, ""},
		{"valid with numbers", "table123", false, ""},
		{"valid min length", "ab", false, ""},

		// Invalid: empty
		{"empty", "", true, "empty"},

		// Invalid: length
		{"too short (1 char)", "a", true, "at least 2"},

		// Invalid: reserved endpoints
		{"reserved collections", "collections", true, "reserved"},
		{"reserved auth", "auth", true, "reserved"},
		{"reserved users", "users", true, "reserved"},
		{"reserved apikeys", "apikeys", true, "reserved"},

		// Invalid: pattern
		{"starts with number", "123table", true, "start with a letter"},
		{"with dash", "user-profiles", true, "start with a letter"},
		{"with space", "user profiles", true, "start with a letter"},

		// Invalid: reserved SQL keywords
		{"reserved word select", "select", true, "reserved keyword"},
		{"reserved word table", "table", true, "reserved keyword"},
		{"reserved word insert", "insert", true, "reserved keyword"},

		// Invalid: system prefix
		{"moon prefix", "moon_data", true, "moon_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCollectionName(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for '%s'", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect error for '%s', got: %v", tt.input, err)
			}
			if tt.expectError && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// TestValidateColumnName tests column name validation (PRD-048)
func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errContains string
	}{
		// Valid cases
		{"valid name", "name", false, ""},
		{"valid with underscore", "user_name", false, ""},
		{"valid with number", "address_line_2", false, ""},

		// Invalid: empty
		{"empty name", "", true, "empty"},

		// Invalid: length
		{"too short", "ab", true, "at least 3"},

		// Invalid: system columns (note: these are also too short, so length error comes first)
		// Using a longer name that's still a system column concept
		// Actually, id and ulid are both only 2-4 chars, so we need to test system column check separately

		// Invalid: uppercase
		{"starts uppercase", "Name", true, "lowercase"},
		{"contains uppercase", "userName", true, "lowercase"},

		// Invalid: pattern
		{"starts with number", "123name", true, "lowercase"},
		{"starts with underscore", "_name", true, "lowercase"},

		// Invalid: SQL keywords
		{"select keyword", "select", true, "reserved keyword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateColumnName(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for '%s'", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect error for '%s', got: %v", tt.input, err)
			}
			if tt.expectError && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// TestValidateColumnName_SystemColumns tests system column validation specifically
func TestValidateColumnName_SystemColumns(t *testing.T) {
	// "id" and "ulid" are system columns
	// However, they are also too short (2 and 4 chars, min is 3)
	// So the length check fails first
	err := validateColumnName("id")
	if err == nil {
		t.Error("Expected error for system column 'id'")
	}

	err = validateColumnName("ulid")
	if err == nil {
		t.Error("Expected error for system column 'ulid'")
	}
	if err != nil && !strings.Contains(err.Error(), "system column") {
		t.Errorf("Expected system column error for 'ulid', got: %v", err)
	}
}

// TestValidateColumnType tests column type validation (PRD-048)
func TestValidateColumnType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errContains string
	}{
		// Valid types
		{"valid string", "string", false, ""},
		{"valid integer", "integer", false, ""},
		{"valid decimal", "decimal", false, ""},
		{"valid boolean", "boolean", false, ""},
		{"valid datetime", "datetime", false, ""},
		{"valid json", "json", false, ""},

		// Deprecated types
		{"deprecated text", "text", true, "deprecated"},
		{"deprecated float", "float", true, "deprecated"},

		// Invalid types
		{"invalid varchar", "varchar", true, "invalid column type"},
		{"invalid unknown", "unknown", true, "invalid column type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateColumnType(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for '%s'", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect error for '%s', got: %v", tt.input, err)
			}
			if tt.expectError && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestGenerateDDL(t *testing.T) {
	columns := []registry.Column{
		{Name: "name", Type: registry.TypeString, Nullable: false},
		{Name: "age", Type: registry.TypeInteger, Nullable: true},
	}

	// Test SQLite DDL
	ddl := generateCreateTableDDL("test", columns, database.DialectSQLite)
	if ddl == "" {
		t.Error("Expected non-empty DDL")
	}
	if !bytes.Contains([]byte(ddl), []byte("CREATE TABLE test")) {
		t.Error("DDL should contain CREATE TABLE statement")
	}

	// Test PostgreSQL DDL
	ddl = generateCreateTableDDL("test", columns, database.DialectPostgres)
	if !bytes.Contains([]byte(ddl), []byte("SERIAL PRIMARY KEY")) {
		t.Error("PostgreSQL DDL should use SERIAL")
	}

	// Test MySQL DDL
	ddl = generateCreateTableDDL("test", columns, database.DialectMySQL)
	if !bytes.Contains([]byte(ddl), []byte("AUTO_INCREMENT PRIMARY KEY")) {
		t.Error("MySQL DDL should use AUTO_INCREMENT")
	}
}

// Tests for Remove Columns functionality
func TestUpdate_RemoveColumns_Success(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection first
	createReq := CreateRequest{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
			{Name: "old_field", Type: registry.TypeString, Nullable: true},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Remove a column
	updateReq := UpdateRequest{
		Name:          "products",
		RemoveColumns: []string{"old_field"},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UpdateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify column was removed
	for _, col := range response.Collection.Columns {
		if col.Name == "old_field" {
			t.Error("Column 'old_field' should have been removed")
		}
	}
}

func TestUpdate_RemoveColumns_SystemColumn(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection first
	createReq := CreateRequest{
		Name: "test_table",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Try to remove system column
	tests := []struct {
		name       string
		columnName string
	}{
		{"remove id", "id"},
		{"remove ulid", "ulid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateReq := UpdateRequest{
				Name:          "test_table",
				RemoveColumns: []string{tt.columnName},
			}
			body, _ = json.Marshal(updateReq)
			req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
			w = httptest.NewRecorder()

			handler.Update(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status code %d for removing system column, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestUpdate_RemoveColumns_NonExistent(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "test_table",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Try to remove non-existent column
	updateReq := UpdateRequest{
		Name:          "test_table",
		RemoveColumns: []string{"nonexistent"},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Tests for Rename Columns functionality
func TestUpdate_RenameColumns_Success(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "customers",
		Columns: []registry.Column{
			{Name: "user_name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Rename column
	updateReq := UpdateRequest{
		Name: "customers",
		RenameColumns: []RenameColumn{
			{OldName: "user_name", NewName: "username"},
		},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UpdateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify column was renamed
	found := false
	for _, col := range response.Collection.Columns {
		if col.Name == "username" {
			found = true
		}
		if col.Name == "user_name" {
			t.Error("Old column name 'user_name' should not exist after rename")
		}
	}
	if !found {
		t.Error("New column name 'username' not found after rename")
	}
}

func TestUpdate_RenameColumns_SystemColumn(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "test_table",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Try to rename system column
	updateReq := UpdateRequest{
		Name: "test_table",
		RenameColumns: []RenameColumn{
			{OldName: "id", NewName: "new_id"},
		},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdate_RenameColumns_Conflict(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "test_table",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Try to rename to existing column name
	updateReq := UpdateRequest{
		Name: "test_table",
		RenameColumns: []RenameColumn{
			{OldName: "name", NewName: "email"},
		},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Tests for Modify Columns functionality
func TestUpdate_ModifyColumns_Success(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "description", Type: registry.TypeString, Nullable: true},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Modify column type
	updateReq := UpdateRequest{
		Name: "products",
		ModifyColumns: []ModifyColumn{
			{Name: "description", Type: registry.TypeString},
		},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UpdateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify column type was modified
	found := false
	for _, col := range response.Collection.Columns {
		if col.Name == "description" && col.Type == registry.TypeString {
			found = true
			break
		}
	}
	if !found {
		t.Error("Column 'description' should have been modified to string type")
	}
}

func TestUpdate_ModifyColumns_SystemColumn(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "test_table",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Try to modify system column
	updateReq := UpdateRequest{
		Name: "test_table",
		ModifyColumns: []ModifyColumn{
			{Name: "id", Type: registry.TypeString},
		},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Tests for Combined Operations
func TestUpdate_CombinedOperations(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "complex_table",
		Columns: []registry.Column{
			{Name: "old_name", Type: registry.TypeString, Nullable: false},
			{Name: "to_remove", Type: registry.TypeString, Nullable: true},
			{Name: "to_modify", Type: registry.TypeString, Nullable: true},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Perform multiple operations
	updateReq := UpdateRequest{
		Name: "complex_table",
		RenameColumns: []RenameColumn{
			{OldName: "old_name", NewName: "new_name"},
		},
		ModifyColumns: []ModifyColumn{
			{Name: "to_modify", Type: registry.TypeString},
		},
		AddColumns: []registry.Column{
			{Name: "new_field", Type: registry.TypeInteger, Nullable: true},
		},
		RemoveColumns: []string{"to_remove"},
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UpdateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify all operations
	hasNewName := false
	hasNewField := false
	hasToModify := false
	for _, col := range response.Collection.Columns {
		if col.Name == "new_name" {
			hasNewName = true
		}
		if col.Name == "new_field" {
			hasNewField = true
		}
		if col.Name == "to_modify" && col.Type == registry.TypeString {
			hasToModify = true
		}
		if col.Name == "old_name" {
			t.Error("Column 'old_name' should have been renamed")
		}
		if col.Name == "to_remove" {
			t.Error("Column 'to_remove' should have been removed")
		}
	}

	if !hasNewName {
		t.Error("Renamed column 'new_name' not found")
	}
	if !hasNewField {
		t.Error("Added column 'new_field' not found")
	}
	if !hasToModify {
		t.Error("Modified column 'to_modify' not found or not changed to string type")
	}
}

func TestUpdate_NoOperations(t *testing.T) {
	handler, driver := setupTestHandler(t)
	defer driver.Close()

	// Create a collection
	createReq := CreateRequest{
		Name: "test_table",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Try to update without any operations
	updateReq := UpdateRequest{
		Name: "test_table",
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Tests for DDL generation
func TestGenerateDropColumnDDL(t *testing.T) {
	ddl := generateDropColumnDDL("test_table", "old_column", database.DialectSQLite)
	if ddl == "" {
		t.Error("Expected non-empty DDL")
	}
	if !bytes.Contains([]byte(ddl), []byte("DROP COLUMN")) {
		t.Error("DDL should contain DROP COLUMN statement")
	}
}

func TestGenerateRenameColumnDDL(t *testing.T) {
	tests := []struct {
		dialect database.DialectType
		want    string
	}{
		{database.DialectPostgres, "RENAME COLUMN"},
		{database.DialectMySQL, "RENAME COLUMN"},
		{database.DialectSQLite, "RENAME COLUMN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dialect), func(t *testing.T) {
			ddl := generateRenameColumnDDL("test_table", "old_name", "new_name", tt.dialect)
			if ddl == "" {
				t.Error("Expected non-empty DDL")
			}
			if !bytes.Contains([]byte(ddl), []byte(tt.want)) {
				t.Errorf("DDL should contain '%s', got: %s", tt.want, ddl)
			}
		})
	}
}

func TestGenerateModifyColumnDDL(t *testing.T) {
	modify := ModifyColumn{
		Name: "test_column",
		Type: registry.TypeString,
	}

	tests := []struct {
		dialect database.DialectType
		want    string
	}{
		{database.DialectPostgres, "ALTER COLUMN"},
		{database.DialectMySQL, "MODIFY COLUMN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dialect), func(t *testing.T) {
			ddl := generateModifyColumnDDL("test_table", modify, tt.dialect)
			if ddl == "" {
				t.Error("Expected non-empty DDL")
			}
			if !bytes.Contains([]byte(ddl), []byte(tt.want)) {
				t.Errorf("DDL should contain '%s', got: %s", tt.want, ddl)
			}
		})
	}
}
