package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// TestBatchCreate_BestEffort_PartialSuccess tests batch create with partial success
func TestBatchCreate_BestEffort_PartialSuccess(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg, testConfig())

	// Second item has invalid field type (string instead of integer for price)
	reqBody := BatchCreateDataRequest{
		Data: json.RawMessage(`[{"name": "Product1", "price": 100}, {"name": "Product2", "price": "invalid"}]`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create?atomic=false", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusMultiStatus {
		t.Errorf("expected status %d, got %d: %s", http.StatusMultiStatus, w.Code, w.Body.String())
	}

	var response BatchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Summary.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Summary.Total)
	}

	if response.Summary.Succeeded != 1 {
		t.Errorf("expected 1 success, got %d", response.Summary.Succeeded)
	}

	if response.Summary.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", response.Summary.Failed)
	}
}

// TestBatchCreate_BatchSizeExceeded tests batch size limit enforcement
func TestBatchCreate_BatchSizeExceeded(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	// Create config with small batch size
	cfg := &config.AppConfig{
		Batch: config.BatchConfig{
			MaxSize:         2,
			MaxPayloadBytes: 2097152,
		},
	}

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg, cfg)

	// Try to create 3 items (exceeds limit of 2)
	reqBody := BatchCreateDataRequest{
		Data: json.RawMessage(`[{"name": "Product1"}, {"name": "Product2"}, {"name": "Product3"}]`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d: %s", http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
	}
}

// TestBatchUpdate_BestEffort tests batch update in best-effort mode
func TestBatchUpdate_BestEffort(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeInteger, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{
		dialect: database.DialectSQLite,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return mockResult{rowsAffected: 1}, nil
		},
	}
	handler := NewDataHandler(driver, reg, testConfig())

	reqBody := BatchUpdateDataRequest{
		Data: json.RawMessage(`[{"id": "01HFXYZ1234567890ABCDEFGHI", "name": "UpdatedProduct1"}, {"id": "01HFXYZ1234567890ABCDEFGHJ", "price": 300}]`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:update?atomic=false", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	if w.Code != http.StatusMultiStatus {
		t.Errorf("expected status %d, got %d: %s", http.StatusMultiStatus, w.Code, w.Body.String())
	}

	var response BatchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Summary.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Summary.Total)
	}

	if response.Summary.Succeeded != 2 {
		t.Errorf("expected 2 successes, got %d", response.Summary.Succeeded)
	}
}

// TestBatchUpdate_MissingID tests batch update with missing ID
func TestBatchUpdate_MissingID(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg, testConfig())

	// Missing "id" field in second item, but using atomic=false
	reqBody := BatchUpdateDataRequest{
		Data: json.RawMessage(`[{"id": "01HFXYZ1234567890ABCDEFGHI", "name": "Product1"}, {"name": "Product2"}]`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:update?atomic=false", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	// In best-effort mode, one should succeed and one should fail
	if w.Code != http.StatusMultiStatus {
		t.Errorf("expected status %d, got %d: %s", http.StatusMultiStatus, w.Code, w.Body.String())
	}

	var response BatchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Summary.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", response.Summary.Failed)
	}
}

// TestBatchDestroy_BestEffort tests batch destroy in best-effort mode
func TestBatchDestroy_BestEffort(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	callCount := 0
	driver := &mockDataDriver{
		dialect: database.DialectSQLite,
		execFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			callCount++
			if callCount == 1 {
				// First ID exists
				return mockResult{rowsAffected: 1}, nil
			}
			// Second ID doesn't exist
			return mockResult{rowsAffected: 0}, nil
		},
	}
	handler := NewDataHandler(driver, reg, testConfig())

	reqBody := BatchDestroyDataRequest{
		Data: json.RawMessage(`["01HFXYZ1234567890ABCDEFGHI", "01HFXYZ1234567890ABCDEFGHJ"]`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:destroy?atomic=false", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Destroy(w, req, "products")

	if w.Code != http.StatusMultiStatus {
		t.Errorf("expected status %d, got %d: %s", http.StatusMultiStatus, w.Code, w.Body.String())
	}

	var response BatchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Summary.Total != 2 {
		t.Errorf("expected total 2, got %d", response.Summary.Total)
	}

	if response.Summary.Succeeded != 1 {
		t.Errorf("expected 1 success, got %d", response.Summary.Succeeded)
	}

	if response.Summary.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", response.Summary.Failed)
	}

	// Check that second item has not_found status
	if len(response.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(response.Results))
	}

	if response.Results[1].Status != BatchItemNotFound {
		t.Errorf("expected not_found status for second item, got %s", response.Results[1].Status)
	}
}

// TestDetectBatchMode tests the batch mode detection
func TestDetectBatchMode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantBatch bool
		wantErr   bool
	}{
		{
			name:      "single object",
			input:     `{"name": "test"}`,
			wantBatch: false,
			wantErr:   false,
		},
		{
			name:      "array of objects",
			input:     `[{"name": "test1"}, {"name": "test2"}]`,
			wantBatch: true,
			wantErr:   false,
		},
		{
			name:      "empty array",
			input:     `[]`,
			wantBatch: true,
			wantErr:   false,
		},
		{
			name:      "empty object",
			input:     `{}`,
			wantBatch: false,
			wantErr:   false,
		},
		{
			name:      "invalid json",
			input:     `invalid`,
			wantErr:   true,
			wantBatch: false,
		},
		{
			name:      "empty string",
			input:     ``,
			wantErr:   true,
			wantBatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isBatch, err := detectBatchMode(json.RawMessage(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("detectBatchMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if isBatch != tt.wantBatch {
				t.Errorf("detectBatchMode() = %v, want %v", isBatch, tt.wantBatch)
			}
		})
	}
}

// TestParseAtomicFlag tests the atomic flag parsing
func TestParseAtomicFlag(t *testing.T) {
	tests := []struct {
		name     string
		queryStr string
		want     bool
	}{
		{
			name:     "no atomic parameter",
			queryStr: "",
			want:     false, // Default is false (best-effort)
		},
		{
			name:     "atomic=true",
			queryStr: "atomic=true",
			want:     true,
		},
		{
			name:     "atomic=false",
			queryStr: "atomic=false",
			want:     false,
		},
		{
			name:     "atomic=1",
			queryStr: "atomic=1",
			want:     true,
		},
		{
			name:     "atomic=0",
			queryStr: "atomic=0",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/test?%s", tt.queryStr), nil)
			got := parseAtomicFlag(req)
			if got != tt.want {
				t.Errorf("parseAtomicFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBackwardCompatibility_SingleCreate tests backward compatibility for single create
func TestBackwardCompatibility_SingleCreate(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
		},
	}
	reg.Set(collection)

	driver := &mockDataDriver{dialect: database.DialectSQLite}
	handler := NewDataHandler(driver, reg, testConfig())

	// Old format: {"data": {"name": "test"}}
	reqBody := CreateDataRequest{
		Data: map[string]any{"name": "test"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req, "products")

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response CreateDataResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data["name"] != "test" {
		t.Errorf("expected name 'test', got %v", response.Data["name"])
	}

	if _, ok := response.Data["id"]; !ok {
		t.Error("expected 'id' field in response")
	}
}

// TestBackwardCompatibility_SingleUpdate tests backward compatibility for single update
func TestBackwardCompatibility_SingleUpdate(t *testing.T) {
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
			return mockResult{rowsAffected: 1}, nil
		},
	}
	handler := NewDataHandler(driver, reg, testConfig())

	// Old format: {"id": "...", "data": {"name": "test"}}
	reqBody := UpdateDataRequest{
		ID:   "01HFXYZ1234567890ABCDEFGHI",
		Data: map[string]any{"name": "updated"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Update(w, req, "products")

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UpdateDataResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data["name"] != "updated" {
		t.Errorf("expected name 'updated', got %v", response.Data["name"])
	}
}

// TestBackwardCompatibility_SingleDestroy tests backward compatibility for single destroy
func TestBackwardCompatibility_SingleDestroy(t *testing.T) {
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
			return mockResult{rowsAffected: 1}, nil
		},
	}
	handler := NewDataHandler(driver, reg, testConfig())

	// Old format: {"id": "..."}
	reqBody := DestroyDataRequest{
		ID: "01HFXYZ1234567890ABCDEFGHI",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products:destroy", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Destroy(w, req, "products")

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response DestroyDataResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(response.Message, "deleted successfully") {
		t.Errorf("unexpected message: %s", response.Message)
	}
}
