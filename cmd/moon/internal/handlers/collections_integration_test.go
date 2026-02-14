package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
	ulidpkg "github.com/thalib/moon/cmd/moon/internal/ulid"
)

// createTestDBForCollections creates an in-memory SQLite database for collections testing
func createTestDBForCollections(t *testing.T) database.Driver {
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

	return driver
}

// TestCollectionsHandler_Create_Integration tests Create with real database
func TestCollectionsHandler_Create_Integration(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	reqBody := map[string]any{
		"name": "products",
		"columns": []map[string]any{
			{"name": "name", "type": "string", "nullable": false},
			{"name": "price", "type": "integer", "nullable": false},
			{"name": "active", "type": "boolean", "nullable": true},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify collection exists in registry
	collection, exists := reg.Get("products")
	if !exists {
		t.Error("Expected collection to exist in registry")
	}

	if collection != nil && len(collection.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(collection.Columns))
	}
}

// TestCollectionsHandler_Update_Integration tests Update with real database
func TestCollectionsHandler_Update_Integration(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	// First create a collection
	createBody := map[string]any{
		"name": "products",
		"columns": []map[string]any{
			{"name": "name", "type": "string", "nullable": false},
		},
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create collection: %s", w.Body.String())
	}

	// Now update it by adding a column
	updateBody := map[string]any{
		"name": "products",
		"add_columns": []map[string]any{
			{"name": "price", "type": "integer", "nullable": true},
		},
	}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify column was added
	collection, _ := reg.Get("products")
	if collection == nil {
		t.Fatal("Collection not found")
	}

	hasPrice := false
	for _, col := range collection.Columns {
		if col.Name == "price" {
			hasPrice = true
			break
		}
	}
	if !hasPrice {
		t.Error("Expected 'price' column to be added")
	}
}

// TestCollectionsHandler_Destroy_Integration tests Destroy with real database
func TestCollectionsHandler_Destroy_Integration(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	// First create a collection
	createBody := map[string]any{
		"name": "products",
		"columns": []map[string]any{
			{"name": "name", "type": "string"},
		},
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create collection: %s", w.Body.String())
	}

	// Now destroy it
	destroyBody := map[string]any{
		"name": "products",
	}
	body, _ = json.Marshal(destroyBody)
	req = httptest.NewRequest(http.MethodPost, "/collections:destroy", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Destroy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify collection is removed from registry
	_, exists := reg.Get("products")
	if exists {
		t.Error("Expected collection to be removed from registry")
	}
}

// TestCollectionsHandler_List_WithData tests List when there are collections
func TestCollectionsHandler_List_WithData(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	// Create some collections
	collections := []map[string]any{
		{
			"name": "customers",
			"columns": []map[string]any{
				{"name": "name", "type": "string"},
			},
		},
		{
			"name": "products",
			"columns": []map[string]any{
				{"name": "title", "type": "string"},
			},
		},
	}

	for _, c := range collections {
		body, _ := json.Marshal(c)
		req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Create(w, req)
	}

	// Now list them
	req := httptest.NewRequest(http.MethodGet, "/collections:list", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]any
	json.NewDecoder(w.Body).Decode(&response)

	if count, ok := response["count"].(float64); ok {
		if int(count) != 2 {
			t.Errorf("Expected count 2, got %v", count)
		}
	}
}

// TestCollectionsHandler_Create_WithAllTypes tests creating collection with all column types
func TestCollectionsHandler_Create_WithAllTypes(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	reqBody := map[string]any{
		"name": "test_all_types",
		"columns": []map[string]any{
			{"name": "text_col", "type": "string", "nullable": false},
			{"name": "int_col", "type": "integer", "nullable": false},
			{"name": "bool_col", "type": "boolean", "nullable": true},
			{"name": "datetime_col", "type": "datetime", "nullable": true},
			{"name": "json_col", "type": "json", "nullable": true},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify collection has all columns
	collection, exists := reg.Get("test_all_types")
	if !exists {
		t.Fatal("Expected collection to exist")
	}

	if len(collection.Columns) != 5 {
		t.Errorf("Expected 5 columns, got %d", len(collection.Columns))
	}

	// Check each column type
	expectedTypes := map[string]registry.ColumnType{
		"text_col":     registry.TypeString,
		"int_col":      registry.TypeInteger,
		"bool_col":     registry.TypeBoolean,
		"datetime_col": registry.TypeDatetime,
		"json_col":     registry.TypeJSON,
	}

	for _, col := range collection.Columns {
		expected, exists := expectedTypes[col.Name]
		if exists && col.Type != expected {
			t.Errorf("Column %s: expected type %v, got %v", col.Name, expected, col.Type)
		}
	}
}

// TestCollectionsHandler_Update_RenameColumn tests renaming a column
func TestCollectionsHandler_Update_RenameColumn(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	// Create collection
	createBody := map[string]any{
		"name": "products",
		"columns": []map[string]any{
			{"name": "name", "type": "string"},
			{"name": "price", "type": "integer"},
		},
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create collection: %s", w.Body.String())
	}

	// Rename column
	updateBody := map[string]any{
		"name": "products",
		"rename_columns": []map[string]any{
			{"old_name": "name", "new_name": "title"},
		},
	}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify column was renamed
	collection, _ := reg.Get("products")
	hasTitle := false
	hasName := false
	for _, col := range collection.Columns {
		if col.Name == "title" {
			hasTitle = true
		}
		if col.Name == "name" {
			hasName = true
		}
	}
	if !hasTitle || hasName {
		t.Error("Expected 'name' column to be renamed to 'title'")
	}
}

// TestCollectionsHandler_Update_RemoveColumn tests removing a column
func TestCollectionsHandler_Update_RemoveColumn(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	// Create collection
	createBody := map[string]any{
		"name": "products",
		"columns": []map[string]any{
			{"name": "name", "type": "string"},
			{"name": "price", "type": "integer"},
			{"name": "description", "type": "string"},
		},
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create collection: %s", w.Body.String())
	}

	// Remove column
	updateBody := map[string]any{
		"name":           "products",
		"remove_columns": []string{"description"},
	}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify column was removed
	collection, _ := reg.Get("products")
	for _, col := range collection.Columns {
		if col.Name == "description" {
			t.Error("Expected 'description' column to be removed")
		}
	}
}

// TestCollectionsHandler_List_DetailedResponse tests List with detailed response including record counts (PRD-065)
func TestCollectionsHandler_List_DetailedResponse(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	// Create multiple collections
	collectionsData := []struct {
		name    string
		records int // number of records to insert
	}{
		{"customers", 5},
		{"products", 10},
		{"orders", 3},
	}

	ctx := context.Background()
	for _, cd := range collectionsData {
		// Create collection
		createBody := map[string]any{
			"name": cd.name,
			"columns": []map[string]any{
				{"name": "data", "type": "string"},
			},
		}
		body, _ := json.Marshal(createBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Create(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create collection %s: %s", cd.name, w.Body.String())
		}

		// Insert records
		for i := 0; i < cd.records; i++ {
			insertSQL := fmt.Sprintf("INSERT INTO %s (id, data) VALUES (?, ?)", cd.name)
			if driver.Dialect() == database.DialectPostgres {
				insertSQL = fmt.Sprintf("INSERT INTO %s (id, data) VALUES ($1, $2)", cd.name)
			}
			_, err := driver.Exec(ctx, insertSQL, ulidpkg.Generate(), fmt.Sprintf("record_%d", i))
			if err != nil {
				t.Fatalf("Failed to insert record into %s: %v", cd.name, err)
			}
		}
	}

	// Now list collections and verify counts
	req := httptest.NewRequest(http.MethodGet, "/collections:list", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response ListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify count
	if response.Count != 3 {
		t.Errorf("Expected count 3, got %d", response.Count)
	}

	// Verify collections array
	if len(response.Collections) != 3 {
		t.Fatalf("Expected 3 collections, got %d", len(response.Collections))
	}

	// Verify each collection has correct name and record count
	expectedCounts := map[string]int{
		"customers": 5,
		"products":  10,
		"orders":    3,
	}

	for _, col := range response.Collections {
		expectedCount, exists := expectedCounts[col.Name]
		if !exists {
			t.Errorf("Unexpected collection: %s", col.Name)
			continue
		}
		if col.Records != expectedCount {
			t.Errorf("Collection %s: expected %d records, got %d", col.Name, expectedCount, col.Records)
		}
	}
}

// TestSystemColumnsProtection_Integration tests that system columns (pkid, id) cannot be modified, deleted, or renamed
func TestSystemColumnsProtection_Integration(t *testing.T) {
	driver := createTestDBForCollections(t)
	defer driver.Close()

	reg := registry.NewSchemaRegistry()
	handler := NewCollectionsHandler(driver, reg)

	// Create a test collection
	createBody := map[string]any{
		"name": "products",
		"columns": []map[string]any{
			{"name": "name", "type": "string", "nullable": false},
			{"name": "price", "type": "integer", "nullable": false},
		},
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/collections:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create collection: %s", w.Body.String())
	}

	// Test 1: Cannot add 'pkid' as a column
	t.Run("CannotAddPkid", func(t *testing.T) {
		updateBody := map[string]any{
			"name": "products",
			"add_columns": []map[string]any{
				{"name": "pkid", "type": "integer", "nullable": false},
			},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for adding pkid, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}

		var errResp map[string]any
		json.NewDecoder(w.Body).Decode(&errResp)
		if errStr, ok := errResp["error"].(string); ok {
			if !strings.Contains(errStr, "system column") {
				t.Errorf("Expected error to mention 'system column', got: %s", errStr)
			}
		}
	})

	// Test 2: Cannot add 'id' as a column
	t.Run("CannotAddId", func(t *testing.T) {
		updateBody := map[string]any{
			"name": "products",
			"add_columns": []map[string]any{
				{"name": "id", "type": "string", "nullable": false},
			},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for adding id, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	// Test 3: Cannot remove 'pkid'
	t.Run("CannotRemovePkid", func(t *testing.T) {
		updateBody := map[string]any{
			"name":           "products",
			"remove_columns": []string{"pkid"},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for removing pkid, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}

		var errResp map[string]any
		json.NewDecoder(w.Body).Decode(&errResp)
		if errStr, ok := errResp["error"].(string); ok {
			if !strings.Contains(errStr, "system column") {
				t.Errorf("Expected error to mention 'system column', got: %s", errStr)
			}
		}
	})

	// Test 4: Cannot remove 'id'
	t.Run("CannotRemoveId", func(t *testing.T) {
		updateBody := map[string]any{
			"name":           "products",
			"remove_columns": []string{"id"},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for removing id, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}

		var errResp map[string]any
		json.NewDecoder(w.Body).Decode(&errResp)
		if errStr, ok := errResp["error"].(string); ok {
			if !strings.Contains(errStr, "system column") {
				t.Errorf("Expected error to mention 'system column', got: %s", errStr)
			}
		}
	})

	// Test 5: Cannot rename 'pkid'
	t.Run("CannotRenamePkid", func(t *testing.T) {
		updateBody := map[string]any{
			"name": "products",
			"rename_columns": []map[string]any{
				{"old_name": "pkid", "new_name": "primary_key"},
			},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for renaming pkid, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}

		var errResp map[string]any
		json.NewDecoder(w.Body).Decode(&errResp)
		if errStr, ok := errResp["error"].(string); ok {
			if !strings.Contains(errStr, "system column") {
				t.Errorf("Expected error to mention 'system column', got: %s", errStr)
			}
		}
	})

	// Test 6: Cannot rename 'id'
	t.Run("CannotRenameId", func(t *testing.T) {
		updateBody := map[string]any{
			"name": "products",
			"rename_columns": []map[string]any{
				{"old_name": "id", "new_name": "identifier"},
			},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for renaming id, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}

		var errResp map[string]any
		json.NewDecoder(w.Body).Decode(&errResp)
		if errStr, ok := errResp["error"].(string); ok {
			if !strings.Contains(errStr, "system column") {
				t.Errorf("Expected error to mention 'system column', got: %s", errStr)
			}
		}
	})

	// Test 7: Cannot modify 'pkid'
	t.Run("CannotModifyPkid", func(t *testing.T) {
		nullable := false
		updateBody := map[string]any{
			"name": "products",
			"modify_columns": []map[string]any{
				{"name": "pkid", "type": "string", "nullable": &nullable},
			},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for modifying pkid, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}

		var errResp map[string]any
		json.NewDecoder(w.Body).Decode(&errResp)
		if errStr, ok := errResp["error"].(string); ok {
			if !strings.Contains(errStr, "system column") {
				t.Errorf("Expected error to mention 'system column', got: %s", errStr)
			}
		}
	})

	// Test 8: Cannot modify 'id'
	t.Run("CannotModifyId", func(t *testing.T) {
		nullable := false
		updateBody := map[string]any{
			"name": "products",
			"modify_columns": []map[string]any{
				{"name": "id", "type": "integer", "nullable": &nullable},
			},
		}
		body, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPost, "/collections:update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Update(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for modifying id, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}

		var errResp map[string]any
		json.NewDecoder(w.Body).Decode(&errResp)
		if errStr, ok := errResp["error"].(string); ok {
			if !strings.Contains(errStr, "system column") {
				t.Errorf("Expected error to mention 'system column', got: %s", errStr)
			}
		}
	})
}
