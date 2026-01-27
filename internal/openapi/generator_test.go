package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/internal/registry"
)

func setupTestRegistry() *registry.SchemaRegistry {
	reg := registry.NewSchemaRegistry()

	// Add test collections
	reg.Set(&registry.Collection{
		Name: "users",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false},
			{Name: "age", Type: registry.TypeInteger, Nullable: true},
			{Name: "active", Type: registry.TypeBoolean, Nullable: true},
		},
	})

	reg.Set(&registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "title", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeFloat, Nullable: false},
			{Name: "description", Type: registry.TypeText, Nullable: true},
			{Name: "created_at", Type: registry.TypeDatetime, Nullable: true},
			{Name: "metadata", Type: registry.TypeJSON, Nullable: true},
		},
	})

	return reg
}

func TestNewGenerator(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	config := GeneratorConfig{}

	gen := NewGenerator(reg, config)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}

	if gen.config.Title != "Moon API" {
		t.Errorf("Expected default title 'Moon API', got '%s'", gen.config.Title)
	}

	if gen.config.Version != "1.0.0" {
		t.Errorf("Expected default version '1.0.0', got '%s'", gen.config.Version)
	}
}

func TestNewGenerator_CustomConfig(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	config := GeneratorConfig{
		Title:       "Custom API",
		Version:     "2.0.0",
		Description: "Custom description",
	}

	gen := NewGenerator(reg, config)

	if gen.config.Title != "Custom API" {
		t.Errorf("Expected title 'Custom API', got '%s'", gen.config.Title)
	}

	if gen.config.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", gen.config.Version)
	}
}

func TestGenerate_BasicStructure(t *testing.T) {
	reg := setupTestRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	if spec.OpenAPI != "3.0.3" {
		t.Errorf("Expected OpenAPI version '3.0.3', got '%s'", spec.OpenAPI)
	}

	if spec.Info.Title != "Moon API" {
		t.Errorf("Expected title 'Moon API', got '%s'", spec.Info.Title)
	}

	if spec.Paths == nil {
		t.Fatal("Paths should not be nil")
	}

	if spec.Components.Schemas == nil {
		t.Fatal("Components.Schemas should not be nil")
	}

	if spec.Components.SecuritySchemes == nil {
		t.Fatal("Components.SecuritySchemes should not be nil")
	}
}

func TestGenerate_SecuritySchemes(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	// Check JWT auth
	jwtAuth, ok := spec.Components.SecuritySchemes["bearerAuth"]
	if !ok {
		t.Fatal("Expected bearerAuth security scheme")
	}

	if jwtAuth.Type != "http" {
		t.Errorf("Expected type 'http', got '%s'", jwtAuth.Type)
	}

	if jwtAuth.Scheme != "bearer" {
		t.Errorf("Expected scheme 'bearer', got '%s'", jwtAuth.Scheme)
	}

	// Check API key auth
	apiKeyAuth, ok := spec.Components.SecuritySchemes["apiKeyAuth"]
	if !ok {
		t.Fatal("Expected apiKeyAuth security scheme")
	}

	if apiKeyAuth.Type != "apiKey" {
		t.Errorf("Expected type 'apiKey', got '%s'", apiKeyAuth.Type)
	}

	if apiKeyAuth.Name != "X-API-Key" {
		t.Errorf("Expected name 'X-API-Key', got '%s'", apiKeyAuth.Name)
	}
}

func TestGenerate_HealthEndpoints(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	// Check /health endpoint
	healthPath, ok := spec.Paths["/health"]
	if !ok {
		t.Fatal("Expected /health path")
	}

	if healthPath.Get == nil {
		t.Fatal("Expected GET operation on /health")
	}

	if healthPath.Get.Summary != "Liveness check" {
		t.Errorf("Expected summary 'Liveness check', got '%s'", healthPath.Get.Summary)
	}

	// Check /health/ready endpoint
	readyPath, ok := spec.Paths["/health/ready"]
	if !ok {
		t.Fatal("Expected /health/ready path")
	}

	if readyPath.Get == nil {
		t.Fatal("Expected GET operation on /health/ready")
	}
}

func TestGenerate_CollectionEndpoints(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	collectionPaths := []string{
		"/api/v1/collections:list",
		"/api/v1/collections:get",
		"/api/v1/collections:create",
		"/api/v1/collections:update",
		"/api/v1/collections:destroy",
	}

	for _, path := range collectionPaths {
		_, ok := spec.Paths[path]
		if !ok {
			t.Errorf("Expected path %s", path)
		}
	}
}

func TestGenerate_DataEndpoints(t *testing.T) {
	reg := setupTestRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	// Check users collection endpoints
	userPaths := []string{
		"/api/v1/users:list",
		"/api/v1/users:get",
		"/api/v1/users:create",
		"/api/v1/users:update",
		"/api/v1/users:destroy",
	}

	for _, path := range userPaths {
		pathItem, ok := spec.Paths[path]
		if !ok {
			t.Errorf("Expected path %s", path)
			continue
		}

		// Check that endpoints have appropriate operations
		if path == "/api/v1/users:list" || path == "/api/v1/users:get" {
			if pathItem.Get == nil {
				t.Errorf("Expected GET operation on %s", path)
			}
		} else {
			if pathItem.Post == nil {
				t.Errorf("Expected POST operation on %s", path)
			}
		}
	}

	// Check products collection endpoints
	productPaths := []string{
		"/api/v1/products:list",
		"/api/v1/products:get",
		"/api/v1/products:create",
		"/api/v1/products:update",
		"/api/v1/products:destroy",
	}

	for _, path := range productPaths {
		_, ok := spec.Paths[path]
		if !ok {
			t.Errorf("Expected path %s", path)
		}
	}
}

func TestGenerate_CollectionSchemas(t *testing.T) {
	reg := setupTestRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	// Check Users schema
	usersSchema, ok := spec.Components.Schemas["Users"]
	if !ok {
		t.Fatal("Expected Users schema")
	}

	if usersSchema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", usersSchema.Type)
	}

	// Check that schema has expected properties
	expectedProps := []string{"id", "name", "email", "age", "active"}
	for _, prop := range expectedProps {
		if _, ok := usersSchema.Properties[prop]; !ok {
			t.Errorf("Expected property '%s' in Users schema", prop)
		}
	}

	// Check UsersInput schema
	usersInputSchema, ok := spec.Components.Schemas["UsersInput"]
	if !ok {
		t.Fatal("Expected UsersInput schema")
	}

	// Input schema should not have id
	if _, ok := usersInputSchema.Properties["id"]; ok {
		t.Error("UsersInput should not have 'id' property")
	}

	// Check required fields
	requiredFound := make(map[string]bool)
	for _, r := range usersInputSchema.Required {
		requiredFound[r] = true
	}

	if !requiredFound["name"] || !requiredFound["email"] {
		t.Error("Expected name and email to be required in UsersInput")
	}
}

func TestGenerate_CommonSchemas(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	commonSchemas := []string{
		"ErrorResponse",
		"HealthResponse",
		"ReadinessResponse",
		"Column",
		"CollectionListResponse",
		"CollectionResponse",
		"CreateCollectionRequest",
		"CreateCollectionResponse",
		"UpdateCollectionRequest",
		"UpdateCollectionResponse",
	}

	for _, schema := range commonSchemas {
		if _, ok := spec.Components.Schemas[schema]; !ok {
			t.Errorf("Expected common schema '%s'", schema)
		}
	}
}

func TestGenerate_Tags(t *testing.T) {
	reg := setupTestRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	expectedTags := []string{"Health", "Collections", "users", "products"}
	tagMap := make(map[string]bool)

	for _, tag := range spec.Tags {
		tagMap[tag.Name] = true
	}

	for _, expected := range expectedTags {
		if !tagMap[expected] {
			t.Errorf("Expected tag '%s'", expected)
		}
	}
}

func TestColumnToSchema(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	testCases := []struct {
		column       registry.Column
		expectedType string
		expectedFormat string
	}{
		{
			column:       registry.Column{Name: "name", Type: registry.TypeString},
			expectedType: "string",
		},
		{
			column:       registry.Column{Name: "bio", Type: registry.TypeText},
			expectedType: "string",
		},
		{
			column:       registry.Column{Name: "age", Type: registry.TypeInteger},
			expectedType: "integer",
		},
		{
			column:         registry.Column{Name: "price", Type: registry.TypeFloat},
			expectedType:   "number",
			expectedFormat: "double",
		},
		{
			column:       registry.Column{Name: "active", Type: registry.TypeBoolean},
			expectedType: "boolean",
		},
		{
			column:         registry.Column{Name: "created_at", Type: registry.TypeDatetime},
			expectedType:   "string",
			expectedFormat: "date-time",
		},
		{
			column:       registry.Column{Name: "metadata", Type: registry.TypeJSON},
			expectedType: "object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.column.Name, func(t *testing.T) {
			schema := gen.columnToSchema(tc.column)

			if schema.Type != tc.expectedType {
				t.Errorf("Expected type '%s', got '%s'", tc.expectedType, schema.Type)
			}

			if tc.expectedFormat != "" && schema.Format != tc.expectedFormat {
				t.Errorf("Expected format '%s', got '%s'", tc.expectedFormat, schema.Format)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/test": {
				Get: &Operation{
					Summary: "Test endpoint",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
	}

	data, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse back and verify
	var parsed OpenAPI
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.OpenAPI != "3.0.3" {
		t.Errorf("Expected OpenAPI '3.0.3', got '%s'", parsed.OpenAPI)
	}

	if parsed.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", parsed.Info.Title)
	}
}

func TestHandler(t *testing.T) {
	reg := setupTestRegistry()
	gen := NewGenerator(reg, GeneratorConfig{
		Title:   "Test API",
		Version: "1.0.0",
	})

	handler := gen.Handler()

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Check CORS header
	cors := w.Header().Get("Access-Control-Allow-Origin")
	if cors != "*" {
		t.Errorf("Expected CORS header '*', got '%s'", cors)
	}

	// Parse response
	var spec OpenAPI
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", spec.Info.Title)
	}
}

func TestCapitalize(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"users", "Users"},
		{"products", "Products"},
		{"", ""},
		{"a", "A"},
		{"ABC", "ABC"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := capitalize(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGenerateCreateExample(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	collection := &registry.Collection{
		Name: "test",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString},
			{Name: "count", Type: registry.TypeInteger},
			{Name: "price", Type: registry.TypeFloat},
			{Name: "active", Type: registry.TypeBoolean},
			{Name: "created_at", Type: registry.TypeDatetime},
			{Name: "metadata", Type: registry.TypeJSON},
		},
	}

	example := gen.generateCreateExample(collection)

	data, ok := example["data"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'data' key in example")
	}

	// Check that all fields have examples
	for _, col := range collection.Columns {
		if _, ok := data[col.Name]; !ok {
			t.Errorf("Expected example for field '%s'", col.Name)
		}
	}
}

func TestGenerate_EmptyRegistry(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	gen := NewGenerator(reg, GeneratorConfig{})

	spec := gen.Generate()

	// Should still have health endpoints
	if _, ok := spec.Paths["/health"]; !ok {
		t.Error("Expected /health path even with empty registry")
	}

	// Should still have collection management endpoints
	if _, ok := spec.Paths["/api/v1/collections:list"]; !ok {
		t.Error("Expected collections:list path even with empty registry")
	}

	// Should have common schemas
	if _, ok := spec.Components.Schemas["ErrorResponse"]; !ok {
		t.Error("Expected ErrorResponse schema")
	}
}

func TestGenerator_WithServers(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	config := GeneratorConfig{
		Servers: []Server{
			{URL: "http://localhost:8080", Description: "Development"},
			{URL: "https://api.example.com", Description: "Production"},
		},
	}

	gen := NewGenerator(reg, config)
	spec := gen.Generate()

	if len(spec.Servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(spec.Servers))
	}

	if spec.Servers[0].URL != "http://localhost:8080" {
		t.Errorf("Expected first server URL 'http://localhost:8080', got '%s'", spec.Servers[0].URL)
	}
}

func TestGenerator_WithContactAndLicense(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	config := GeneratorConfig{
		Contact: &Contact{
			Name:  "API Support",
			Email: "support@example.com",
		},
		License: &License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	}

	gen := NewGenerator(reg, config)
	spec := gen.Generate()

	if spec.Info.Contact == nil {
		t.Fatal("Expected Contact info")
	}

	if spec.Info.Contact.Name != "API Support" {
		t.Errorf("Expected contact name 'API Support', got '%s'", spec.Info.Contact.Name)
	}

	if spec.Info.License == nil {
		t.Fatal("Expected License info")
	}

	if spec.Info.License.Name != "MIT" {
		t.Errorf("Expected license name 'MIT', got '%s'", spec.Info.License.Name)
	}
}
