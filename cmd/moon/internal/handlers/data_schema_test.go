package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

func TestSchemaEndpoint(t *testing.T) {
	// Initialize database
	dbConfig := database.Config{
		ConnectionString: ":memory:",
	}
	driver, err := database.NewDriver(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer driver.Close()

	// Initialize registry
	reg := registry.NewSchemaRegistry()

	// Create a test collection
	testCollection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
			{Name: "description", Type: registry.TypeString, Nullable: true},
		},
	}

	// Add to registry
	reg.Set(testCollection)

	// Create the collection table
	_, err = driver.Exec(ctx, `
		CREATE TABLE products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ulid TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			price INTEGER NOT NULL,
			description TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	handler := NewDataHandler(driver, reg)

	// Test 1: Get schema for existing collection
	t.Run("get_schema_existing_collection", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:schema", nil)
		w := httptest.NewRecorder()
		handler.Schema(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SchemaResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response structure
		if resp.Collection != "products" {
			t.Errorf("Expected collection 'products', got %s", resp.Collection)
		}

		if len(resp.Fields) == 0 {
			t.Error("Expected fields in schema, got empty array")
		}

		// Verify fields include id and user-defined columns
		fieldNames := make(map[string]bool)
		for _, field := range resp.Fields {
			fieldNames[field.Name] = true
		}

		expectedFields := []string{"id", "name", "price", "description"}
		for _, expectedField := range expectedFields {
			if !fieldNames[expectedField] {
				t.Errorf("Expected field '%s' in schema, but not found", expectedField)
			}
		}

		// Verify field properties
		for _, field := range resp.Fields {
			if field.Name == "id" {
				if field.Type != "string" {
					t.Errorf("Expected id type 'string', got %s", field.Type)
				}
				if field.Nullable {
					t.Error("Expected id to be non-nullable")
				}
			}
			if field.Name == "name" {
				if field.Type != "string" {
					t.Errorf("Expected name type 'string', got %s", field.Type)
				}
				if field.Nullable {
					t.Error("Expected name to be non-nullable")
				}
			}
			if field.Name == "description" {
				if field.Type != "string" {
					t.Errorf("Expected description type 'string', got %s", field.Type)
				}
				if !field.Nullable {
					t.Error("Expected description to be nullable")
				}
			}
		}
	})

	// Test 2: Get schema for non-existent collection - should return 404
	t.Run("get_schema_nonexistent_collection", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent:schema", nil)
		w := httptest.NewRecorder()
		handler.Schema(w, req, "nonexistent")

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d: %s", w.Code, w.Body.String())
		}

		var errResp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if errResp["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	// Test 3: Verify schema response format matches PRD-054 specification
	t.Run("schema_response_format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products:schema", nil)
		w := httptest.NewRecorder()
		handler.Schema(w, req, "products")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		// Parse as generic JSON to verify structure
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify required fields exist
		if _, ok := resp["collection"]; !ok {
			t.Error("Expected 'collection' field in response")
		}

		if _, ok := resp["fields"]; !ok {
			t.Error("Expected 'fields' field in response")
		}

		// Verify fields is an array
		fields, ok := resp["fields"].([]any)
		if !ok {
			t.Error("Expected 'fields' to be an array")
		}

		// Verify field structure
		if len(fields) > 0 {
			firstField, ok := fields[0].(map[string]any)
			if !ok {
				t.Error("Expected field to be an object")
			}

			requiredFieldProps := []string{"name", "type", "nullable"}
			for _, prop := range requiredFieldProps {
				if _, ok := firstField[prop]; !ok {
					t.Errorf("Expected field to have '%s' property", prop)
				}
			}
		}
	})
}
