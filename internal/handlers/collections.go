package handlers

import (
"encoding/json"
"fmt"
"net/http"
"regexp"
"strings"

"github.com/thalib/moon/internal/database"
"github.com/thalib/moon/internal/registry"
)

var (
// Valid collection name pattern: alphanumeric and underscores only
collectionNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// Reserved SQL keywords that cannot be used as collection names
reservedWords = map[string]bool{
"select": true, "insert": true, "update": true, "delete": true,
"from": true, "where": true, "join": true, "table": true,
"index": true, "view": true, "trigger": true, "function": true,
"procedure": true, "database": true, "schema": true, "user": true,
}
)

// CollectionsHandler handles schema management operations
type CollectionsHandler struct {
db       database.Driver
registry *registry.SchemaRegistry
}

// NewCollectionsHandler creates a new collections handler
func NewCollectionsHandler(db database.Driver, reg *registry.SchemaRegistry) *CollectionsHandler {
return &CollectionsHandler{
db:       db,
registry: reg,
}
}

// ListRequest represents the request for listing collections
type ListRequest struct {
// Optional filters can be added here
}

// ListResponse represents the response for listing collections
type ListResponse struct {
Collections []string `json:"collections"`
Count       int      `json:"count"`
}

// GetRequest represents the request for getting a collection schema
type GetRequest struct {
Name string `json:"name"`
}

// GetResponse represents the response for getting a collection schema
type GetResponse struct {
Collection *registry.Collection `json:"collection"`
}

// CreateRequest represents the request for creating a collection
type CreateRequest struct {
Name    string            `json:"name"`
Columns []registry.Column `json:"columns"`
}

// CreateResponse represents the response for creating a collection
type CreateResponse struct {
Collection *registry.Collection `json:"collection"`
Message    string               `json:"message"`
}

// UpdateRequest represents the request for updating a collection
type UpdateRequest struct {
Name       string            `json:"name"`
AddColumns []registry.Column `json:"add_columns,omitempty"`
// Future: support for dropping columns, renaming, etc.
}

// UpdateResponse represents the response for updating a collection
type UpdateResponse struct {
Collection *registry.Collection `json:"collection"`
Message    string               `json:"message"`
}

// DestroyRequest represents the request for destroying a collection
type DestroyRequest struct {
Name string `json:"name"`
}

// DestroyResponse represents the response for destroying a collection
type DestroyResponse struct {
Message string `json:"message"`
}

// List handles GET /collections:list
func (h *CollectionsHandler) List(w http.ResponseWriter, r *http.Request) {
collections := h.registry.List()

response := ListResponse{
Collections: collections,
Count:       len(collections),
}

writeJSON(w, http.StatusOK, response)
}

// Get handles GET /collections:get
func (h *CollectionsHandler) Get(w http.ResponseWriter, r *http.Request) {
name := r.URL.Query().Get("name")
if name == "" {
writeError(w, http.StatusBadRequest, "collection name is required")
return
}

collection, exists := h.registry.Get(name)
if !exists {
writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", name))
return
}

response := GetResponse{
Collection: collection,
}

writeJSON(w, http.StatusOK, response)
}

// Create handles POST /collections:create
func (h *CollectionsHandler) Create(w http.ResponseWriter, r *http.Request) {
var req CreateRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
writeError(w, http.StatusBadRequest, "invalid request body")
return
}

// Validate collection name
if err := validateCollectionName(req.Name); err != nil {
writeError(w, http.StatusBadRequest, err.Error())
return
}

// Check if collection already exists
if h.registry.Exists(req.Name) {
writeError(w, http.StatusConflict, fmt.Sprintf("collection '%s' already exists", req.Name))
return
}

// Validate columns
if len(req.Columns) == 0 {
writeError(w, http.StatusBadRequest, "at least one column is required")
return
}

for i, col := range req.Columns {
if col.Name == "" {
writeError(w, http.StatusBadRequest, fmt.Sprintf("column %d: name is required", i))
return
}
if !registry.ValidateColumnType(col.Type) {
writeError(w, http.StatusBadRequest, fmt.Sprintf("column '%s': invalid type '%s'", col.Name, col.Type))
return
}
}

// Generate CREATE TABLE DDL
ddl := generateCreateTableDDL(req.Name, req.Columns, h.db.Dialect())

// Execute DDL
ctx := r.Context()
if _, err := h.db.Exec(ctx, ddl); err != nil {
writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create table: %v", err))
return
}

// Update registry
collection := &registry.Collection{
Name:    req.Name,
Columns: req.Columns,
}

if err := h.registry.Set(collection); err != nil {
writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update registry: %v", err))
return
}

response := CreateResponse{
Collection: collection,
Message:    fmt.Sprintf("Collection '%s' created successfully", req.Name),
}

writeJSON(w, http.StatusCreated, response)
}

// Update handles POST /collections:update
func (h *CollectionsHandler) Update(w http.ResponseWriter, r *http.Request) {
var req UpdateRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
writeError(w, http.StatusBadRequest, "invalid request body")
return
}

// Validate collection name
if err := validateCollectionName(req.Name); err != nil {
writeError(w, http.StatusBadRequest, err.Error())
return
}

// Check if collection exists
collection, exists := h.registry.Get(req.Name)
if !exists {
writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", req.Name))
return
}

// Validate and add new columns
if len(req.AddColumns) == 0 {
writeError(w, http.StatusBadRequest, "no columns to add")
return
}

for i, col := range req.AddColumns {
if col.Name == "" {
writeError(w, http.StatusBadRequest, fmt.Sprintf("column %d: name is required", i))
return
}
if !registry.ValidateColumnType(col.Type) {
writeError(w, http.StatusBadRequest, fmt.Sprintf("column '%s': invalid type '%s'", col.Name, col.Type))
return
}

// Check if column already exists
for _, existing := range collection.Columns {
if existing.Name == col.Name {
writeError(w, http.StatusConflict, fmt.Sprintf("column '%s' already exists", col.Name))
return
}
}
}

// Generate ALTER TABLE DDL
ctx := r.Context()
for _, col := range req.AddColumns {
ddl := generateAddColumnDDL(req.Name, col, h.db.Dialect())
if _, err := h.db.Exec(ctx, ddl); err != nil {
writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add column '%s': %v", col.Name, err))
return
}
}

// Update registry
collection.Columns = append(collection.Columns, req.AddColumns...)
if err := h.registry.Set(collection); err != nil {
writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update registry: %v", err))
return
}

response := UpdateResponse{
Collection: collection,
Message:    fmt.Sprintf("Collection '%s' updated successfully", req.Name),
}

writeJSON(w, http.StatusOK, response)
}

// Destroy handles POST /collections:destroy
func (h *CollectionsHandler) Destroy(w http.ResponseWriter, r *http.Request) {
var req DestroyRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
writeError(w, http.StatusBadRequest, "invalid request body")
return
}

// Validate collection name
if err := validateCollectionName(req.Name); err != nil {
writeError(w, http.StatusBadRequest, err.Error())
return
}

// Check if collection exists
if !h.registry.Exists(req.Name) {
writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", req.Name))
return
}

// Generate DROP TABLE DDL
ddl := fmt.Sprintf("DROP TABLE %s", req.Name)

// Execute DDL
ctx := r.Context()
if _, err := h.db.Exec(ctx, ddl); err != nil {
writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to drop table: %v", err))
return
}

// Remove from registry
if err := h.registry.Delete(req.Name); err != nil {
writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update registry: %v", err))
return
}

response := DestroyResponse{
Message: fmt.Sprintf("Collection '%s' destroyed successfully", req.Name),
}

writeJSON(w, http.StatusOK, response)
}

// validateCollectionName validates a collection name
func validateCollectionName(name string) error {
if name == "" {
return fmt.Errorf("collection name cannot be empty")
}

if !collectionNameRegex.MatchString(name) {
return fmt.Errorf("collection name must start with a letter and contain only alphanumeric characters and underscores")
}

if reservedWords[strings.ToLower(name)] {
return fmt.Errorf("'%s' is a reserved keyword and cannot be used as a collection name", name)
}

return nil
}

// generateCreateTableDDL generates CREATE TABLE DDL for the given dialect
func generateCreateTableDDL(tableName string, columns []registry.Column, dialect database.DialectType) string {
var sb strings.Builder

sb.WriteString(fmt.Sprintf("CREATE TABLE %s (", tableName))

// Add id column (auto-increment primary key)
switch dialect {
case database.DialectPostgres:
sb.WriteString("\n  id SERIAL PRIMARY KEY")
case database.DialectMySQL:
sb.WriteString("\n  id INT AUTO_INCREMENT PRIMARY KEY")
case database.DialectSQLite:
sb.WriteString("\n  id INTEGER PRIMARY KEY AUTOINCREMENT")
}

// Add user-defined columns
for _, col := range columns {
sb.WriteString(",\n  ")
sb.WriteString(col.Name)
sb.WriteString(" ")
sb.WriteString(mapColumnTypeToSQL(col.Type, dialect))

if !col.Nullable {
sb.WriteString(" NOT NULL")
}

if col.Unique {
sb.WriteString(" UNIQUE")
}

if col.DefaultValue != nil {
sb.WriteString(" DEFAULT ")
sb.WriteString(*col.DefaultValue)
}
}

sb.WriteString("\n)")

return sb.String()
}

// generateAddColumnDDL generates ALTER TABLE ADD COLUMN DDL
func generateAddColumnDDL(tableName string, column registry.Column, dialect database.DialectType) string {
var sb strings.Builder

sb.WriteString(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
tableName, column.Name, mapColumnTypeToSQL(column.Type, dialect)))

if !column.Nullable {
sb.WriteString(" NOT NULL")
}

if column.Unique {
sb.WriteString(" UNIQUE")
}

if column.DefaultValue != nil {
sb.WriteString(" DEFAULT ")
sb.WriteString(*column.DefaultValue)
}

return sb.String()
}

// mapColumnTypeToSQL maps ColumnType to SQL type for the given dialect
func mapColumnTypeToSQL(colType registry.ColumnType, dialect database.DialectType) string {
switch dialect {
case database.DialectPostgres:
return mapColumnTypeToPostgres(colType)
case database.DialectMySQL:
return mapColumnTypeToMySQL(colType)
case database.DialectSQLite:
return mapColumnTypeToSQLite(colType)
default:
return "TEXT"
}
}

func mapColumnTypeToPostgres(colType registry.ColumnType) string {
switch colType {
case registry.TypeString:
return "VARCHAR(255)"
case registry.TypeInteger:
return "INTEGER"
case registry.TypeFloat:
return "DOUBLE PRECISION"
case registry.TypeBoolean:
return "BOOLEAN"
case registry.TypeDatetime:
return "TIMESTAMP"
case registry.TypeText:
return "TEXT"
case registry.TypeJSON:
return "JSONB"
default:
return "TEXT"
}
}

func mapColumnTypeToMySQL(colType registry.ColumnType) string {
switch colType {
case registry.TypeString:
return "VARCHAR(255)"
case registry.TypeInteger:
return "INT"
case registry.TypeFloat:
return "DOUBLE"
case registry.TypeBoolean:
return "BOOLEAN"
case registry.TypeDatetime:
return "DATETIME"
case registry.TypeText:
return "TEXT"
case registry.TypeJSON:
return "JSON"
default:
return "TEXT"
}
}

func mapColumnTypeToSQLite(colType registry.ColumnType) string {
switch colType {
case registry.TypeString:
return "TEXT"
case registry.TypeInteger:
return "INTEGER"
case registry.TypeFloat:
return "REAL"
case registry.TypeBoolean:
return "INTEGER"
case registry.TypeDatetime:
return "TEXT"
case registry.TypeText:
return "TEXT"
case registry.TypeJSON:
return "TEXT"
default:
return "TEXT"
}
}

// Helper functions for JSON responses
func writeJSON(w http.ResponseWriter, statusCode int, data any) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(statusCode)
json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
writeJSON(w, statusCode, map[string]any{
"error": message,
"code":  statusCode,
})
}
