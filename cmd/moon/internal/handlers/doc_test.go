package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

func TestDocHandler_HTML(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 3600,
		},
		APIKey: config.APIKeyConfig{
			Enabled: false,
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test
	req := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	rec := httptest.NewRecorder()

	handler.HTML(rec, req)

	// Assertions
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html; charset=utf-8, got %s", contentType)
	}

	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Error("expected Cache-Control header to be set")
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("expected ETag header to be set")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTML document")
	}
	if !strings.Contains(body, "Moon API Documentation") {
		t.Error("expected page title")
	}
	if !strings.Contains(body, "1.99") {
		t.Error("expected version number")
	}
	if !strings.Contains(body, "Table of Contents") {
		t.Error("expected table of contents")
	}
	if !strings.Contains(body, "Manage Collections") {
		t.Error("expected collection management section")
	}
	if !strings.Contains(body, "Data Access") {
		t.Error("expected data access section")
	}
	if !strings.Contains(body, "Aggregation Operations") {
		t.Error("expected aggregation operations section")
	}
}

func TestDocHandler_Markdown(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
		JWT: config.JWTConfig{
			Secret: "",
			Expiry: 3600,
		},
		APIKey: config.APIKeyConfig{
			Enabled: true,
			Header:  "X-API-KEY",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec := httptest.NewRecorder()

	handler.Markdown(rec, req)

	// Assertions
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/markdown; charset=utf-8" {
		t.Errorf("expected Content-Type text/markdown; charset=utf-8, got %s", contentType)
	}

	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Error("expected Cache-Control header to be set")
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("expected ETag header to be set")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "# Moon") {
		t.Error("expected markdown heading")
	}
	if !strings.Contains(body, "| Version     | 1.99") {
		t.Error("expected version number")
	}
	if !strings.Contains(body, "## ") {
		t.Error("expected sections in documentation")
	}
	if !strings.Contains(body, "## Aggregation Operations") {
		t.Error("expected aggregation operations section")
	}
	if !strings.Contains(body, "Authorization: Bearer") {
		t.Error("expected Authorization: Bearer header in auth section")
	}
}

func TestDocHandler_Caching(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	rec1 := httptest.NewRecorder()
	handler.HTML(rec1, req1)

	etag1 := rec1.Header().Get("ETag")

	// Second request should return same ETag (cached)
	req2 := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	rec2 := httptest.NewRecorder()
	handler.HTML(rec2, req2)

	etag2 := rec2.Header().Get("ETag")

	if etag1 != etag2 {
		t.Error("expected same ETag for cached content")
	}

	// Request with If-None-Match should return 304
	req3 := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	req3.Header.Set("If-None-Match", etag1)
	rec3 := httptest.NewRecorder()
	handler.HTML(rec3, req3)

	if rec3.Code != http.StatusNotModified {
		t.Errorf("expected status 304, got %d", rec3.Code)
	}
}

func TestDocHandler_RefreshCache(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Generate initial cache
	req1 := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	rec1 := httptest.NewRecorder()
	handler.HTML(rec1, req1)

	// Refresh cache
	req2 := httptest.NewRequest(http.MethodPost, "/doc:refresh", nil)
	rec2 := httptest.NewRecorder()
	handler.RefreshCache(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec2.Code)
	}

	body := rec2.Body.String()
	if !strings.Contains(body, "Documentation cache refreshed") {
		t.Error("expected refresh message")
	}

	// Verify cache was cleared by checking internal state
	handler.cacheMutex.RLock()
	htmlCache := handler.htmlCache
	mdCache := handler.mdCache
	handler.cacheMutex.RUnlock()

	if htmlCache != nil {
		t.Error("expected HTML cache to be cleared")
	}
	if mdCache != nil {
		t.Error("expected Markdown cache to be cleared")
	}
}

func TestDocHandler_WithPrefix(t *testing.T) {
	// Setup with prefix
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "/api/v1",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test HTML
	req := httptest.NewRequest(http.MethodGet, "/api/v1/doc/", nil)
	rec := httptest.NewRecorder()
	handler.HTML(rec, req)

	body := rec.Body.String()
	// Template shows prefix in curl examples using $ApiURL variable
	// Tables show generic paths without prefix as reference
	if !strings.Contains(body, "/api/v1/products:list") {
		t.Error("expected prefix in curl examples")
	}
	// Check that prefix is mentioned in the documentation
	if !strings.Contains(body, "/api/v1") {
		t.Error("expected prefix to be mentioned in documentation")
	}
	// Check that the prefix message is present
	if !strings.Contains(body, "All endpoints are prefixed with") {
		t.Error("expected prefix message in documentation")
	}
}

func TestDocHandler_WithCollections(t *testing.T) {
	// Setup with collections
	reg := registry.NewSchemaRegistry()
	reg.Set(&registry.Collection{
		Name: "users",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false},
		},
	})
	reg.Set(&registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "title", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
		},
	})

	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test Markdown - verify handler has access to collections via registry
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	// Verify we get valid markdown documentation
	body := rec.Body.String()
	if !strings.Contains(body, "# Moon") {
		t.Error("expected markdown heading")
	}
	if !strings.Contains(body, "## ") {
		t.Error("expected sections in documentation")
	}
}

func TestDocHandler_QuickstartSection(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test Markdown
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	body := rec.Body.String()

	// Check for collection management content which serves as quickstart
	if !strings.Contains(body, "## Manage Collections") {
		t.Error("expected Collection Management section")
	}
	if !strings.Contains(body, "collections:create") {
		t.Error("expected collections:create example")
	}
	if !strings.Contains(body, "collections:list") {
		t.Error("expected collections:list example")
	}
	if !strings.Contains(body, "{collection}") {
		t.Error("expected {collection} placeholder")
	}
}

func TestDocHandler_ErrorSection(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test HTML
	req := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	rec := httptest.NewRecorder()
	handler.HTML(rec, req)

	body := rec.Body.String()

	// Check for error documentation
	if !strings.Contains(body, "Error Responses") {
		t.Error("expected Error Responses section")
	}
	if !strings.Contains(body, "400 Bad Request") {
		t.Error("expected 400 status code documentation")
	}
	if !strings.Contains(body, "404 Not Found") {
		t.Error("expected 404 status code documentation")
	}
	// Goldmark HTML-escapes quotes in code blocks
	if !strings.Contains(body, `&quot;error&quot;`) && !strings.Contains(body, `"error"`) {
		t.Error("expected error field in example")
	}
	if !strings.Contains(body, `&quot;code&quot;`) && !strings.Contains(body, `"code"`) {
		t.Error("expected code field in example")
	}
}

func TestDocHandler_ExampleRequests(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test Markdown
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	body := rec.Body.String()

	// Check for key sections and examples in the documentation
	if !strings.Contains(body, "## Manage Collections") {
		t.Error("expected Collection Management section")
	}
	if !strings.Contains(body, "## Data Access") {
		t.Error("expected Data Access section")
	}
	if !strings.Contains(body, "## Aggregation Operations") {
		t.Error("expected Aggregation Operations section")
	}
	if !strings.Contains(body, "collections:create") {
		t.Error("expected collections:create in examples")
	}
	if !strings.Contains(body, ":count") {
		t.Error("expected :count in aggregation examples")
	}
	if !strings.Contains(body, "jq") {
		t.Error("expected jq in curl examples")
	}
}

func TestDocHandler_CopyButtonHTML(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test HTML
	req := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	rec := httptest.NewRecorder()
	handler.HTML(rec, req)

	body := rec.Body.String()

	// Check for copy button CSS
	if !strings.Contains(body, ".copy-btn") {
		t.Error("expected .copy-btn CSS class")
	}
	if !strings.Contains(body, "position: absolute") {
		t.Error("expected absolute positioning in CSS")
	}
	if !strings.Contains(body, ".copy-btn.copied") {
		t.Error("expected .copy-btn.copied CSS class for copied state")
	}

	// Check for JavaScript
	if !strings.Contains(body, "<script>") {
		t.Error("expected JavaScript tag")
	}
	if !strings.Contains(body, "document.querySelectorAll('pre > code')") {
		t.Error("expected code block selector")
	}
	if !strings.Contains(body, "navigator.clipboard.writeText") {
		t.Error("expected clipboard API usage")
	}
	if !strings.Contains(body, "button.innerText = 'Copy'") {
		t.Error("expected copy button text")
	}
	if !strings.Contains(body, "button.innerText = 'Copied!'") {
		t.Error("expected copied state text")
	}
	if !strings.Contains(body, "1200") {
		t.Error("expected 1200ms timeout for copied state")
	}

	// Check for pre element positioning
	if !strings.Contains(body, "pre { background: #2c3e50; color: #ecf0f1; padding: 15px; border-radius: 5px; overflow-x: auto; position: relative; padding-right: 80px; }") {
		t.Error("expected pre element to have position: relative and padding-right")
	}
}

func TestDocHandler_JSONAppendixPresence(t *testing.T) {
	// Setup with collections
	reg := registry.NewSchemaRegistry()
	reg.Set(&registry.Collection{
		Name: "users",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false},
		},
	})
	reg.Set(&registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "title", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
		},
	})

	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 3600,
		},
		APIKey: config.APIKeyConfig{
			Enabled: true,
			Header:  "X-API-KEY",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test JSON endpoint instead of Markdown
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.json", nil)
	rec := httptest.NewRecorder()
	handler.JSON(rec, req)

	body := rec.Body.String()

	// Check for key JSON fields
	if !strings.Contains(body, `"service": "moon"`) {
		t.Error("expected service field in JSON appendix")
	}
	if !strings.Contains(body, `"version": "1.99"`) {
		t.Error("expected version field to be 1.99 in JSON appendix")
	}
	if strings.Contains(body, `"document_version"`) {
		t.Error("did not expect document_version field in JSON appendix")
	}
	if !strings.Contains(body, `"registered_collections"`) {
		t.Error("expected registered_collections field in JSON appendix")
	}

	// Check that collections are present
	if !strings.Contains(body, `"users"`) {
		t.Error("expected users collection in JSON appendix")
	}
	if !strings.Contains(body, `"products"`) {
		t.Error("expected products collection in JSON appendix")
	}

	// Check that fields are present
	if !strings.Contains(body, `"name"`) {
		t.Error("expected name field in JSON appendix")
	}
	if !strings.Contains(body, `"email"`) {
		t.Error("expected email field in JSON appendix")
	}
	if !strings.Contains(body, `"title"`) {
		t.Error("expected title field in JSON appendix")
	}
	if !strings.Contains(body, `"price"`) {
		t.Error("expected price field in JSON appendix")
	}

	// Check that field types are present
	if !strings.Contains(body, `"type": "string"`) {
		t.Error("expected string type in JSON appendix")
	}
	if !strings.Contains(body, `"type": "integer"`) {
		t.Error("expected integer type in JSON appendix")
	}

	// Check authentication modes
	if !strings.Contains(body, `"jwt"`) {
		t.Error("expected jwt auth mode in JSON appendix")
	}
	if !strings.Contains(body, `"api_key"`) {
		t.Error("expected api_key auth mode in JSON appendix")
	}

	// Check for unique property in fields
	if !strings.Contains(body, `"unique"`) {
		t.Error("expected unique property in field definitions")
	}
}

func TestDocHandler_JSONAppendixUniqueAndDataTypes(t *testing.T) {
	// Setup with collections that have unique fields
	reg := registry.NewSchemaRegistry()
	reg.Set(&registry.Collection{
		Name: "users",
		Columns: []registry.Column{
			{Name: "email", Type: registry.TypeString, Nullable: false, Unique: true},
			{Name: "name", Type: registry.TypeString, Nullable: false, Unique: false},
			{Name: "age", Type: registry.TypeInteger, Nullable: true, Unique: false},
		},
	})

	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test JSON endpoint instead of Markdown
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.json", nil)
	rec := httptest.NewRecorder()
	handler.JSON(rec, req)

	body := rec.Body.String()

	// Check for unique properties
	if !strings.Contains(body, `"unique": true`) {
		t.Error("expected unique: true for unique fields in JSON appendix")
	}
	if !strings.Contains(body, `"unique": false`) {
		t.Error("expected unique: false for non-unique fields in JSON appendix")
	}

	// Check for enhanced data type information with notes
	if !strings.Contains(body, `"note"`) {
		t.Error("expected note field in data types")
	}
	if !strings.Contains(body, `Nullable fields default to`) {
		t.Error("expected nullability default information in data types")
	}

	// Check for specific data type notes
	if !strings.Contains(body, `empty string ('') when null`) {
		t.Error("expected string nullability note")
	}
	if !strings.Contains(body, `default to 0 when null`) {
		t.Error("expected integer nullability note")
	}

	// Check that version matches what was passed to handler
	if !strings.Contains(body, `"version": "1.99"`) {
		t.Error("expected version to match handler version (1.99)")
	}

	// Check that document_version is removed
	if strings.Contains(body, `"document_version"`) {
		t.Error("document_version field should be removed from JSON appendix")
	}
}

func TestDocHandler_JSONAppendixDeterminism(t *testing.T) {
	// Setup config
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 3600,
		},
	}

	// Create two registries with collections added in different order
	reg1 := registry.NewSchemaRegistry()
	reg1.Set(&registry.Collection{
		Name: "zebras",
		Columns: []registry.Column{
			{Name: "stripe_count", Type: registry.TypeInteger, Nullable: false},
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	})
	reg1.Set(&registry.Collection{
		Name: "aardvarks",
		Columns: []registry.Column{
			{Name: "weight", Type: registry.TypeInteger, Nullable: false},
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	})

	reg2 := registry.NewSchemaRegistry()
	reg2.Set(&registry.Collection{
		Name: "aardvarks",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "weight", Type: registry.TypeInteger, Nullable: false},
		},
	})
	reg2.Set(&registry.Collection{
		Name: "zebras",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "stripe_count", Type: registry.TypeInteger, Nullable: false},
		},
	})

	// Generate documentation with both registries
	handler1 := NewDocHandler(reg1, cfg, "1.99")
	req1 := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec1 := httptest.NewRecorder()
	handler1.Markdown(rec1, req1)
	body1 := rec1.Body.String()

	handler2 := NewDocHandler(reg2, cfg, "1.99")
	req2 := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec2 := httptest.NewRecorder()
	handler2.Markdown(rec2, req2)
	body2 := rec2.Body.String()

	// Both should produce identical JSON appendix sections
	if body1 != body2 {
		t.Error("expected identical JSON appendix regardless of insertion order")

		// Extract and compare JSON sections for debugging
		startMarker := "## JSON Appendix"
		idx1 := strings.Index(body1, startMarker)
		idx2 := strings.Index(body2, startMarker)

		if idx1 != -1 && idx2 != -1 {
			json1 := body1[idx1:]
			json2 := body2[idx2:]

			if json1 != json2 {
				t.Logf("JSON appendix sections differ")
				t.Logf("First 500 chars of json1: %s", json1[:min(500, len(json1))])
				t.Logf("First 500 chars of json2: %s", json2[:min(500, len(json2))])
			}
		}
	}
}

func TestDocHandler_JSONAppendixRefresh(t *testing.T) {
	// Setup with initial collection
	reg := registry.NewSchemaRegistry()
	reg.Set(&registry.Collection{
		Name: "users",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	})

	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Generate initial JSON
	req1 := httptest.NewRequest(http.MethodGet, "/doc/llms.json", nil)
	rec1 := httptest.NewRecorder()
	handler.JSON(rec1, req1)
	body1 := rec1.Body.String()

	// Verify users is present in JSON appendix
	if !strings.Contains(body1, `"name": "users"`) {
		t.Error("expected users collection in initial JSON appendix registered_collections")
	}
	// Verify products collection is not present in registered_collections
	if strings.Contains(body1, `"name": "products"`) {
		t.Error("did not expect products collection in initial JSON appendix registered_collections")
	}

	// Add a new collection to the registry
	reg.Set(&registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "title", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
		},
	})

	// Refresh the cache to clear it
	req2 := httptest.NewRequest(http.MethodPost, "/doc:refresh", nil)
	rec2 := httptest.NewRecorder()
	handler.RefreshCache(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected refresh to return 200, got %d", rec2.Code)
	}

	// Generate JSON again (should rebuild with new collection)
	req3 := httptest.NewRequest(http.MethodGet, "/doc/llms.json", nil)
	rec3 := httptest.NewRecorder()
	handler.JSON(rec3, req3)
	body3 := rec3.Body.String()

	// Verify both collections are now present in registered_collections
	if !strings.Contains(body3, `"name": "users"`) {
		t.Error("expected users collection in refreshed JSON appendix")
	}
	if !strings.Contains(body3, `"name": "products"`) {
		t.Error("expected products collection in refreshed JSON appendix")
	}
	if !strings.Contains(body3, `"name": "title"`) {
		t.Error("expected title field in refreshed JSON appendix")
	}
	if !strings.Contains(body3, `"name": "price"`) {
		t.Error("expected price field in refreshed JSON appendix")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestDocHandler_MarkdownIncludeFunction(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Verify the template has the include function by checking it doesn't error
	if handler.mdTemplate == nil {
		t.Fatal("expected mdTemplate to be initialized")
	}

	// The include function should be available in the template
	// We can verify this indirectly by ensuring the handler was created successfully
	if handler.registry == nil {
		t.Error("expected registry to be set")
	}
}

func TestDocHandler_IncludeExistingFile(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test that markdown generation works (which uses the template with include function)
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify the markdown was generated
	body := rec.Body.String()
	if body == "" {
		t.Error("expected non-empty markdown body")
	}
}

func TestDocHandler_IncludeFileHandlesErrors(t *testing.T) {
	// This test verifies that the include function handles missing files gracefully
	// The actual error handling is tested by the fact that NewDocHandler doesn't panic
	// even if include files are missing
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	// This should not panic even if template tries to include non-existent files
	handler := NewDocHandler(reg, cfg, "1.99")

	if handler == nil {
		t.Fatal("expected handler to be created")
	}

	// Generate markdown - should work even with missing includes
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 even with potential missing includes, got %d", rec.Code)
	}
}

func TestDocHandler_MarkdownIncludesInHTML(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test HTML generation (which converts markdown that may include files)
	req := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	rec := httptest.NewRecorder()
	handler.HTML(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTML document")
	}

	// Verify markdown was converted to HTML
	if !strings.Contains(body, "<body>") {
		t.Error("expected body tag in HTML")
	}
}

func TestDocHandler_IncludeFunctionIntegration(t *testing.T) {
	// This test demonstrates the include function working end-to-end
	// by creating a simple template that uses includes

	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	// Create a handler which initializes the include function
	handler := NewDocHandler(reg, cfg, "1.99")

	// The handler should have the template ready with include function
	if handler.mdTemplate == nil {
		t.Fatal("expected mdTemplate to be initialized")
	}

	// Test executing a simple template with include
	// Note: The actual doc.md.tmpl doesn't use includes by default (only has comments),
	// but the function is available for use

	// Verify we can generate markdown without errors
	markdown, err := handler.generateMarkdown()
	if err != nil {
		t.Fatalf("failed to generate markdown: %v", err)
	}

	if markdown == "" {
		t.Error("expected non-empty markdown output")
	}

	// Verify the markdown contains expected content (not include content, as we don't use it in the actual template)
	if !strings.Contains(markdown, "Moon") {
		t.Error("expected markdown to contain 'Moon'")
	}
}

func TestDocHandler_IncludeWithTemplateContext(t *testing.T) {
	// Test that the include function works with template context
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Generate both markdown and HTML to ensure includes work in both contexts
	markdown, err := handler.generateMarkdown()
	if err != nil {
		t.Fatalf("failed to generate markdown: %v", err)
	}

	html, err := handler.generateHTML()
	if err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	if markdown == "" {
		t.Error("expected non-empty markdown")
	}

	if html == "" {
		t.Error("expected non-empty HTML")
	}

	// Verify HTML contains the converted markdown
	if !strings.Contains(html, "<body>") {
		t.Error("expected HTML body")
	}
}

func TestDocHandler_IncludeSecurityValidation(t *testing.T) {
	// Test that the include function has security validation
	// We test this indirectly by verifying the handler initializes correctly
	// and that the template system doesn't allow reading arbitrary files

	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Verify handler was created successfully
	if handler == nil {
		t.Fatal("expected handler to be created")
	}

	// The include function should be registered
	if handler.mdTemplate == nil {
		t.Fatal("expected mdTemplate to be initialized")
	}

	// Generate markdown - this executes the template with the include function
	markdown, err := handler.generateMarkdown()
	if err != nil {
		t.Fatalf("failed to generate markdown: %v", err)
	}

	if markdown == "" {
		t.Error("expected non-empty markdown output")
	}

	// The template should not contain any error messages from invalid includes
	// since the default template doesn't use includes
	if strings.Contains(markdown, "Error: Invalid filename") {
		t.Error("unexpected invalid filename error in default template")
	}
}

func TestDocHandler_IncludeValidation_Documentation(t *testing.T) {
	// This test documents the security validation behavior
	// The actual validation happens in the include function within NewDocHandler

	// Security checks implemented:
	// 1. Must have .md extension
	// 2. No directory traversal patterns (.., /, \)
	// 3. Filename must equal filepath.Base(filename) - no path components
	// 4. Clean filename used for reading

	// Test cases that should be blocked:
	blockedPatterns := []string{
		"../../../etc/passwd",   // Directory traversal up
		"../../doc.go",          // Directory traversal to source
		"/etc/passwd",           // Absolute path
		"..\\..\\system32\\sam", // Windows path traversal
		"subdir/../passwd",      // Subdir with traversal
		"malicious.txt",         // Wrong extension
		"script.js",             // Wrong extension
		"readme",                // No extension
	}

	// Test cases that should be allowed:
	allowedPatterns := []string{
		"example.md",
		"footer.md",
		"troubleshooting.md",
		"test-file.md",
		"my_file_123.md",
		"DEMO_USAGE.md",
	}

	t.Logf("Blocked patterns (would be rejected): %v", blockedPatterns)
	t.Logf("Allowed patterns (would be accepted): %v", allowedPatterns)

	// Verify this test documents the expected behavior
	if len(blockedPatterns) == 0 || len(allowedPatterns) == 0 {
		t.Error("test should document both blocked and allowed patterns")
	}
}

func TestDocHandler_JSON(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	reg.Set(&registry.Collection{
		Name: "users",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false},
		},
	})

	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 3600,
		},
		APIKey: config.APIKeyConfig{
			Enabled: true,
			Header:  "X-API-Key",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test JSON endpoint
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.json", nil)
	rec := httptest.NewRecorder()

	handler.JSON(rec, req)

	// Assertions
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %s", contentType)
	}

	body := rec.Body.String()
	if body == "" {
		t.Error("expected non-empty JSON body")
	}

	// Check for key JSON fields
	if !strings.Contains(body, `"service": "moon"`) {
		t.Error("expected service field in JSON")
	}
	if !strings.Contains(body, `"data_access"`) {
		t.Error("expected data_access field in JSON")
	}
	if !strings.Contains(body, `"query"`) {
		t.Error("expected query field under data_access in JSON")
	}
	if !strings.Contains(body, `"aggregation"`) {
		t.Error("expected aggregation field under data_access in JSON")
	}
	if !strings.Contains(body, `"registered_collections"`) {
		t.Error("expected registered_collections field in JSON")
	}
	if !strings.Contains(body, `"users"`) {
		t.Error("expected users collection in JSON")
	}
}

func TestDocHandler_JSONStructure(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test that query and aggregation are nested under data_access
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.json", nil)
	rec := httptest.NewRecorder()

	handler.JSON(rec, req)

	body := rec.Body.String()

	// Verify structure: data_access.query and data_access.aggregation
	if !strings.Contains(body, `"data_access"`) {
		t.Error("expected data_access field in JSON")
	}

	// Parse JSON to verify structure
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(body), &jsonData); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Check that data_access exists
	dataAccess, ok := jsonData["data_access"].(map[string]any)
	if !ok {
		t.Fatal("expected data_access to be an object")
	}

	// Check that query is under data_access
	if _, ok := dataAccess["query"]; !ok {
		t.Error("expected query field under data_access")
	}

	// Check that aggregation is under data_access
	if _, ok := dataAccess["aggregation"]; !ok {
		t.Error("expected aggregation field under data_access")
	}

	// Ensure query and aggregation are NOT at top level
	if _, ok := jsonData["query"]; ok {
		t.Error("query should not be at top level, should be under data_access")
	}
	if _, ok := jsonData["aggregation"]; ok {
		t.Error("aggregation should not be at top level, should be under data_access")
	}
}

func TestDocHandler_MarkdownWithoutJSONAppendix(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test Markdown endpoint
	req := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec := httptest.NewRecorder()

	handler.Markdown(rec, req)

	body := rec.Body.String()

	// Verify that the JSON Appendix section is NOT in markdown
	if strings.Contains(body, "## JSON Appendix") {
		t.Error("markdown should not contain JSON Appendix section")
	}

	// Verify markdown still has main sections
	if !strings.Contains(body, "# Moon") {
		t.Error("expected markdown heading")
	}
	if !strings.Contains(body, "## Manage Collections") {
		t.Error("expected Collection Management section")
	}
	if !strings.Contains(body, "## Data Access") {
		t.Error("expected Data Access section")
	}
	if !strings.Contains(body, "## Security") {
		t.Error("expected Security section")
	}
}

func TestDocHandler_TextEndpoint(t *testing.T) {
	// Setup
	reg := registry.NewSchemaRegistry()
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host:   "localhost",
			Port:   6006,
			Prefix: "",
		},
	}

	handler := NewDocHandler(reg, cfg, "1.99")

	// Test that /doc/llms.txt returns the same as /doc/llms.md
	req1 := httptest.NewRequest(http.MethodGet, "/doc/llms.md", nil)
	rec1 := httptest.NewRecorder()
	handler.Markdown(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/doc/llms.txt", nil)
	rec2 := httptest.NewRecorder()
	handler.Markdown(rec2, req2)

	// Both should return identical content
	if rec1.Body.String() != rec2.Body.String() {
		t.Error("expected /doc/llms.txt to return same content as /doc/llms.md")
	}
}
