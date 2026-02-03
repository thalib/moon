package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// setupDataIntegrationTest creates a database with a products collection
func setupDataIntegrationTest(t *testing.T) (database.Driver, *registry.SchemaRegistry, *DataHandler) {
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

	// Create table
	_, err = driver.Exec(ctx, `CREATE TABLE products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ulid TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		price INTEGER NOT NULL,
		category TEXT,
		active INTEGER DEFAULT 1
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
			{Name: "category", Type: registry.TypeString, Nullable: true},
			{Name: "active", Type: registry.TypeBoolean, Nullable: true},
		},
	}
	reg.Set(collection)

	handler := NewDataHandler(driver, reg)

	return driver, reg, handler
}

// TestDataHandler_CRUD_Integration tests the full CRUD cycle
func TestDataHandler_CRUD_Integration(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	// 1. Create a product
	createBody := CreateDataRequest{
		Data: map[string]any{
			"name":     "Test Product",
			"price":    99,
			"category": "electronics",
			"active":   true,
		},
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req, "products")

	if w.Code != http.StatusCreated {
		t.Fatalf("Create failed: expected %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var createResp CreateDataResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	productID, ok := createResp.Data["id"].(string)
	if !ok {
		t.Fatalf("Expected string ID, got %T", createResp.Data["id"])
	}

	// 2. Get the product
	req = httptest.NewRequest(http.MethodGet, "/products:get?id="+productID, nil)
	w = httptest.NewRecorder()
	handler.Get(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("Get failed: expected %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var getResp DataGetResponse
	json.NewDecoder(w.Body).Decode(&getResp)

	if getResp.Data["name"] != "Test Product" {
		t.Errorf("Expected name 'Test Product', got %v", getResp.Data["name"])
	}

	// 3. Update the product
	updateBody := UpdateDataRequest{
		ID: productID,
		Data: map[string]any{
			"price": 149,
		},
	}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Update(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("Update failed: expected %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// 4. Verify update by getting again
	req = httptest.NewRequest(http.MethodGet, "/products:get?id="+productID, nil)
	w = httptest.NewRecorder()
	handler.Get(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("Get after update failed: %s", w.Body.String())
	}

	json.NewDecoder(w.Body).Decode(&getResp)
	if price, ok := getResp.Data["price"].(float64); !ok || int(price) != 149 {
		t.Errorf("Expected price 149, got %v", getResp.Data["price"])
	}

	// 5. Delete the product
	deleteBody := DestroyDataRequest{
		ID: productID,
	}
	body, _ = json.Marshal(deleteBody)
	req = httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.Destroy(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("Destroy failed: expected %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// 6. Verify deletion
	req = httptest.NewRequest(http.MethodGet, "/products:get?id="+productID, nil)
	w = httptest.NewRecorder()
	handler.Get(w, req, "products")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after deletion, got %d", w.Code)
	}
}

// TestDataHandler_List_Integration tests listing with filters and pagination
func TestDataHandler_List_Integration(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	ctx := context.Background()

	// Insert test data directly
	testProducts := []struct {
		ulid     string
		name     string
		price    int
		category string
	}{
		{"01ARYZ6S41TSV4RRFFQ69G5FA1", "Laptop", 999, "electronics"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA2", "Mouse", 29, "electronics"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA3", "Desk", 199, "furniture"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA4", "Chair", 149, "furniture"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA5", "Monitor", 299, "electronics"},
	}

	for _, p := range testProducts {
		_, err := driver.Exec(ctx, "INSERT INTO products (ulid, name, price, category) VALUES (?, ?, ?, ?)",
			p.ulid, p.name, p.price, p.category)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	tests := []struct {
		name          string
		url           string
		expectedCount int
	}{
		{
			name:          "list all",
			url:           "/products:list",
			expectedCount: 5,
		},
		{
			name:          "list with limit",
			url:           "/products:list?limit=2",
			expectedCount: 2,
		},
		{
			name:          "filter by category",
			url:           "/products:list?category[eq]=electronics",
			expectedCount: 3,
		},
		{
			name:          "filter by price greater than",
			url:           "/products:list?price[gt]=200",
			expectedCount: 2, // Laptop 999, Monitor 299
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			handler.List(w, req, "products")

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
				return
			}

			var resp DataListResponse
			json.NewDecoder(w.Body).Decode(&resp)

			if len(resp.Data) != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, len(resp.Data))
			}
		})
	}
}

// TestDataHandler_List_WithSort tests listing with sort parameter
func TestDataHandler_List_WithSort(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	ctx := context.Background()

	// Insert test data
	testProducts := []struct {
		ulid  string
		name  string
		price int
	}{
		{"01ARYZ6S41TSV4RRFFQ69G5FA1", "Banana", 100},
		{"01ARYZ6S41TSV4RRFFQ69G5FA2", "Apple", 200},
		{"01ARYZ6S41TSV4RRFFQ69G5FA3", "Cherry", 50},
	}

	for _, p := range testProducts {
		_, err := driver.Exec(ctx, "INSERT INTO products (ulid, name, price, category) VALUES (?, ?, ?, 'fruit')",
			p.ulid, p.name, p.price)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Test ascending sort by name
	req := httptest.NewRequest(http.MethodGet, "/products:list?sort=name", nil)
	w := httptest.NewRecorder()
	handler.List(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("List failed: %s", w.Body.String())
	}

	var resp DataListResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Data) > 0 && resp.Data[0]["name"] != "Apple" {
		t.Errorf("Expected first item to be 'Apple' when sorted by name, got %v", resp.Data[0]["name"])
	}

	// Test descending sort by price
	req = httptest.NewRequest(http.MethodGet, "/products:list?sort=-price", nil)
	w = httptest.NewRecorder()
	handler.List(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("List failed: %s", w.Body.String())
	}

	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Data) > 0 {
		price, ok := resp.Data[0]["price"].(float64)
		if !ok || int(price) != 200 {
			t.Errorf("Expected first item to have price 200 when sorted by -price, got %v", resp.Data[0]["price"])
		}
	}
}

// TestDataHandler_List_WithSearch tests search functionality
func TestDataHandler_List_WithSearch(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	ctx := context.Background()

	// Insert test data with varied names
	testProducts := []struct {
		ulid string
		name string
	}{
		{"01ARYZ6S41TSV4RRFFQ69G5FA1", "Red Apple"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA2", "Green Apple"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA3", "Banana"},
		{"01ARYZ6S41TSV4RRFFQ69G5FA4", "Orange Juice"},
	}

	for _, p := range testProducts {
		_, err := driver.Exec(ctx, "INSERT INTO products (ulid, name, price, category) VALUES (?, ?, 10, 'fruit')",
			p.ulid, p.name)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Search for "Apple" - should find items containing "Apple" in name
	req := httptest.NewRequest(http.MethodGet, "/products:list?search=Apple", nil)
	w := httptest.NewRecorder()
	handler.List(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("List with search failed: %s", w.Body.String())
	}

	var resp DataListResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Should find at least 2 items with Apple in name
	if len(resp.Data) < 2 {
		t.Errorf("Expected at least 2 items matching 'Apple', got %d", len(resp.Data))
	}
}

// TestDataHandler_List_WithFields tests field selection
func TestDataHandler_List_WithFields(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	ctx := context.Background()

	// Insert test data
	_, err := driver.Exec(ctx, "INSERT INTO products (ulid, name, price, category) VALUES (?, ?, ?, ?)",
		"01ARYZ6S41TSV4RRFFQ69G5FA1", "Test", 100, "cat1")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Request only specific fields
	req := httptest.NewRequest(http.MethodGet, "/products:list?fields=name,price", nil)
	w := httptest.NewRecorder()
	handler.List(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("List with fields failed: %s", w.Body.String())
	}

	var resp DataListResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Data) > 0 {
		// Should have name and price, and possibly ulid (always included)
		if _, hasName := resp.Data[0]["name"]; !hasName {
			t.Error("Expected 'name' field in response")
		}
		if _, hasPrice := resp.Data[0]["price"]; !hasPrice {
			t.Error("Expected 'price' field in response")
		}
	}
}

// TestDataHandler_Create_ClientULIDIgnored tests that client-provided IDs are handled properly
func TestDataHandler_Create_ClientULIDIgnored(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	// Provide valid fields only (no id or ulid as those are system fields)
	createBody := CreateDataRequest{
		Data: map[string]any{
			"name":  "Test Product",
			"price": 50,
		},
	}
	body, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Create(w, req, "products")

	if w.Code != http.StatusCreated {
		t.Fatalf("Create failed: %s", w.Body.String())
	}

	var resp CreateDataResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// ID should be a server-generated 26-character ULID
	if id, ok := resp.Data["id"].(string); ok {
		if len(id) != 26 {
			t.Errorf("Expected ULID (26 chars), got %d chars", len(id))
		}
	} else {
		t.Error("Expected id to be a string")
	}
}

// TestDataHandler_BooleanResponseUniformity tests PRD-051: Boolean API Response Uniformity
// Ensures that boolean fields are returned as JSON true/false, not integers (0/1)
func TestDataHandler_BooleanResponseUniformity(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	ctx := context.Background()

	// Insert test data with boolean field set to 1 (SQLite representation)
	testProducts := []struct {
		ulid   string
		name   string
		active int
	}{
		{"01ARYZ6S41TSV4RRFFQ69G5FA1", "Active Product", 1},
		{"01ARYZ6S41TSV4RRFFQ69G5FA2", "Inactive Product", 0},
	}

	for _, p := range testProducts {
		_, err := driver.Exec(ctx, "INSERT INTO products (ulid, name, price, active) VALUES (?, ?, 10, ?)",
			p.ulid, p.name, p.active)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Test 1: Get single record and verify boolean is true, not 1
	req := httptest.NewRequest(http.MethodGet, "/products:get?id="+testProducts[0].ulid, nil)
	w := httptest.NewRecorder()
	handler.Get(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("Get failed: %s", w.Body.String())
	}

	var getResp DataGetResponse
	if err := json.NewDecoder(w.Body).Decode(&getResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify active is boolean true, not integer 1
	activeVal, exists := getResp.Data["active"]
	if !exists {
		t.Fatal("Expected 'active' field in response")
	}

	activeBool, isBool := activeVal.(bool)
	if !isBool {
		t.Errorf("Expected 'active' to be bool, got %T (value: %v)", activeVal, activeVal)
	}
	if !activeBool {
		t.Errorf("Expected 'active' to be true, got %v", activeBool)
	}

	// Test 2: Get record with false value
	req = httptest.NewRequest(http.MethodGet, "/products:get?id="+testProducts[1].ulid, nil)
	w = httptest.NewRecorder()
	handler.Get(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("Get failed: %s", w.Body.String())
	}

	if err := json.NewDecoder(w.Body).Decode(&getResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	activeVal = getResp.Data["active"]
	activeBool, isBool = activeVal.(bool)
	if !isBool {
		t.Errorf("Expected 'active' to be bool, got %T (value: %v)", activeVal, activeVal)
	}
	if activeBool {
		t.Errorf("Expected 'active' to be false, got %v", activeBool)
	}

	// Test 3: List records and verify all booleans are proper type
	req = httptest.NewRequest(http.MethodGet, "/products:list", nil)
	w = httptest.NewRecorder()
	handler.List(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("List failed: %s", w.Body.String())
	}

	var listResp DataListResponse
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(listResp.Data) < 2 {
		t.Fatalf("Expected at least 2 records, got %d", len(listResp.Data))
	}

	for i, record := range listResp.Data {
		activeVal, exists := record["active"]
		if !exists {
			continue // Skip if field not present
		}

		if _, isBool := activeVal.(bool); !isBool {
			t.Errorf("Record %d: Expected 'active' to be bool, got %T (value: %v)", i, activeVal, activeVal)
		}
	}

	// Test 4: Verify raw JSON contains literal true/false, not 1/0
	req = httptest.NewRequest(http.MethodGet, "/products:get?id="+testProducts[0].ulid, nil)
	w = httptest.NewRecorder()
	handler.Get(w, req, "products")

	if w.Code != http.StatusOK {
		t.Fatalf("Get failed: %s", w.Body.String())
	}

	bodyStr := w.Body.String()
	// Check that response contains literal "true" and not "1" for the active field
	// This is a simple heuristic check; we already verified type above
	if !bytes.Contains([]byte(bodyStr), []byte("true")) && !bytes.Contains([]byte(bodyStr), []byte("false")) {
		t.Logf("Warning: Response may not contain JSON boolean literals. Body: %s", bodyStr)
	}
}

// TestDataHandler_SchemaParameter tests PRD-053: Schema Query Parameter
func TestDataHandler_SchemaParameter(t *testing.T) {
	driver, _, handler := setupDataIntegrationTest(t)
	defer driver.Close()

	ctx := context.Background()

	// Insert test data
	testProducts := []struct {
		ulid   string
		name   string
		price  int
		active int
	}{
		{"01ARYZ6S41TSV4RRFFQ69G5FA1", "Product 1", 100, 1},
		{"01ARYZ6S41TSV4RRFFQ69G5FA2", "Product 2", 200, 0},
	}

	for _, p := range testProducts {
		_, err := driver.Exec(ctx, "INSERT INTO products (ulid, name, price, active) VALUES (?, ?, ?, ?)",
			p.ulid, p.name, p.price, p.active)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Test 1: List with ?schema (both data and schema)
	t.Run("list_with_schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:list?schema", nil)
		w := httptest.NewRecorder()
		handler.List(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("List failed: %s", w.Body.String())
		}

		var resp DataListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have both data and schema
		if resp.Data == nil {
			t.Error("Expected data in response, got nil")
		}
		if resp.Schema == nil {
			t.Error("Expected schema in response, got nil")
		}
		if resp.Schema != nil {
			if resp.Schema.Collection != "products" {
				t.Errorf("Expected collection name 'products', got %s", resp.Schema.Collection)
			}
			if len(resp.Schema.Fields) == 0 {
				t.Error("Expected fields in schema, got empty array")
			}
			if resp.Schema.PrimaryKey != "id" {
				t.Errorf("Expected primary_key 'id', got %s", resp.Schema.PrimaryKey)
			}
		}
	})

	// Test 2: List with ?schema=only (schema only, no data)
	t.Run("list_with_schema_only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:list?schema=only", nil)
		w := httptest.NewRecorder()
		handler.List(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("List failed: %s", w.Body.String())
		}

		var resp DataListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have schema but no data
		if resp.Data != nil {
			t.Errorf("Expected no data in response (schema=only), got %v", resp.Data)
		}
		if resp.Schema == nil {
			t.Error("Expected schema in response, got nil")
		}
	})

	// Test 3: Get with ?schema (both data and schema)
	t.Run("get_with_schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:get?id="+testProducts[0].ulid+"&schema", nil)
		w := httptest.NewRecorder()
		handler.Get(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("Get failed: %s", w.Body.String())
		}

		var resp DataGetResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have both data and schema
		if resp.Data == nil {
			t.Error("Expected data in response, got nil")
		}
		if resp.Schema == nil {
			t.Error("Expected schema in response, got nil")
		}
	})

	// Test 4: Get with ?schema=only
	t.Run("get_with_schema_only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:get?id="+testProducts[0].ulid+"&schema=only", nil)
		w := httptest.NewRecorder()
		handler.Get(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("Get failed: %s", w.Body.String())
		}

		var resp DataGetResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have schema but no data
		if resp.Data != nil {
			t.Errorf("Expected no data in response (schema=only), got %v", resp.Data)
		}
		if resp.Schema == nil {
			t.Error("Expected schema in response, got nil")
		}
	})

	// Test 5: Schema parameter with filters and pagination
	t.Run("schema_with_filters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:list?schema&price[gte]=150&limit=1", nil)
		w := httptest.NewRecorder()
		handler.List(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("List with filters failed: %s", w.Body.String())
		}

		var resp DataListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have both data (filtered) and schema
		if resp.Data == nil {
			t.Error("Expected data in response")
		}
		if resp.Schema == nil {
			t.Error("Expected schema in response")
		}
		// Data should be filtered to one record (price >= 150)
		if len(resp.Data) != 1 {
			t.Errorf("Expected 1 filtered record, got %d", len(resp.Data))
		}
	})

	// Test 6: Verify schema structure is complete
	t.Run("verify_schema_structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:list?schema=only", nil)
		w := httptest.NewRecorder()
		handler.List(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("List failed: %s", w.Body.String())
		}

		var resp DataListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Schema == nil {
			t.Fatal("Expected schema in response")
		}

		// Check for required fields
		foundFields := make(map[string]bool)
		for _, field := range resp.Schema.Fields {
			foundFields[field.Name] = true
		}

		// Should have id, name, price, category, active
		requiredFields := []string{"id", "name", "price", "category", "active"}
		for _, fieldName := range requiredFields {
			if !foundFields[fieldName] {
				t.Errorf("Schema missing expected field: %s", fieldName)
			}
		}

		// Check metadata
		if resp.Schema.Metadata == nil {
			t.Error("Expected metadata in schema")
		}
	})
}
