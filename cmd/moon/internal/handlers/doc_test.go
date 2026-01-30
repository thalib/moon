package handlers

import (
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
	if !strings.Contains(body, "Schema Management") {
		t.Error("expected schema management section")
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
	req := httptest.NewRequest(http.MethodGet, "/doc/md", nil)
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
	if !strings.Contains(body, "# Moon API Documentation") {
		t.Error("expected markdown heading")
	}
	if !strings.Contains(body, "**Version:** 1.99") {
		t.Error("expected version number")
	}
	if !strings.Contains(body, "## Table of Contents") {
		t.Error("expected table of contents")
	}
	if !strings.Contains(body, "## Schema Management") {
		t.Error("expected schema management section")
	}
	if !strings.Contains(body, "## Data Access") {
		t.Error("expected data access section")
	}
	if !strings.Contains(body, "## Aggregation Operations") {
		t.Error("expected aggregation operations section")
	}
	if !strings.Contains(body, "X-API-KEY") {
		t.Error("expected API key header in auth section")
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
	if !strings.Contains(body, "/api/v1/health") {
		t.Error("expected prefix in health endpoint")
	}
	if !strings.Contains(body, "/api/v1/collections:list") {
		t.Error("expected prefix in collections endpoint")
	}
	if !strings.Contains(body, "/api/v1/{collection}:list") {
		t.Error("expected prefix in data endpoints")
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
			{Name: "price", Type: registry.TypeFloat, Nullable: false},
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

	// Test Markdown
	req := httptest.NewRequest(http.MethodGet, "/doc/md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "users") {
		t.Error("expected users collection in available collections")
	}
	if !strings.Contains(body, "products") {
		t.Error("expected products collection in available collections")
	}
	if !strings.Contains(body, "## Available Collections") {
		t.Error("expected available collections section")
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
	req := httptest.NewRequest(http.MethodGet, "/doc/md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	body := rec.Body.String()

	// Check for quickstart steps
	if !strings.Contains(body, "## Quickstart") {
		t.Error("expected Quickstart section")
	}
	if !strings.Contains(body, "### 1. Create a collection") {
		t.Error("expected step 1")
	}
	if !strings.Contains(body, "### 2. Insert a record") {
		t.Error("expected step 2")
	}
	if !strings.Contains(body, "### 3. List records") {
		t.Error("expected step 3")
	}
	if !strings.Contains(body, "### 4. Update a record") {
		t.Error("expected step 4")
	}
	if !strings.Contains(body, "### 5. Delete a record") {
		t.Error("expected step 5")
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
	req := httptest.NewRequest(http.MethodGet, "/doc/md", nil)
	rec := httptest.NewRecorder()
	handler.Markdown(rec, req)

	body := rec.Body.String()

	// Check for example categories
	if !strings.Contains(body, "## Example Requests") {
		t.Error("expected Example Requests section")
	}
	if !strings.Contains(body, "### Schema Management Example") {
		t.Error("expected schema management example")
	}
	if !strings.Contains(body, "### Data Access Example") {
		t.Error("expected data access example")
	}
	if !strings.Contains(body, "### Aggregation Example") {
		t.Error("expected aggregation example")
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
