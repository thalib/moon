package handlers

import (
"bytes"
"context"
"encoding/json"
"net/http"
"net/http/httptest"
"testing"
"time"

"github.com/thalib/moon/internal/database"
"github.com/thalib/moon/internal/registry"
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
Name: "users",
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

if response.Collection.Name != "users" {
t.Errorf("Expected collection name 'users', got '%s'", response.Collection.Name)
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
{Name: "total", Type: registry.TypeFloat, Nullable: false},
{Name: "notes", Type: registry.TypeText, Nullable: true},
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

func TestValidateCollectionName(t *testing.T) {
tests := []struct {
name        string
input       string
expectError bool
}{
{"valid name", "users", false},
{"valid with underscore", "user_profiles", false},
{"valid with numbers", "table123", false},
{"empty", "", true},
{"starts with number", "123table", true},
{"reserved word", "select", true},
{"with dash", "user-profiles", true},
{"with space", "user profiles", true},
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
