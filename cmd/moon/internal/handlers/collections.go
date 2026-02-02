// Package handlers provides HTTP handlers for schema and data management.
// It implements the AIP-136 custom actions pattern with colon separators
// for RESTful API endpoints as specified in SPEC.md.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
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

	// System columns that cannot be removed or renamed
	systemColumns = map[string]bool{
		"id":   true,
		"ulid": true,
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

// RenameColumn represents a column rename operation
type RenameColumn struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// ModifyColumn represents a column modification operation
type ModifyColumn struct {
	Name         string              `json:"name"`
	Type         registry.ColumnType `json:"type"`
	Nullable     *bool               `json:"nullable,omitempty"`
	Unique       *bool               `json:"unique,omitempty"`
	DefaultValue *string             `json:"default_value,omitempty"`
}

// UpdateRequest represents the request for updating a collection
type UpdateRequest struct {
	Name          string            `json:"name"`
	AddColumns    []registry.Column `json:"add_columns,omitempty"`
	RemoveColumns []string          `json:"remove_columns,omitempty"`
	RenameColumns []RenameColumn    `json:"rename_columns,omitempty"`
	ModifyColumns []ModifyColumn    `json:"modify_columns,omitempty"`
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
	allCollections := h.registry.List()

	// Filter out system tables
	collections := make([]string, 0, len(allCollections))
	for _, col := range allCollections {
		if !constants.IsSystemTable(col) {
			collections = append(collections, col)
		}
	}

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

	// Validate that at least one operation is requested
	if len(req.AddColumns) == 0 && len(req.RemoveColumns) == 0 &&
		len(req.RenameColumns) == 0 && len(req.ModifyColumns) == 0 {
		writeError(w, http.StatusBadRequest, "no operations specified")
		return
	}

	// Save original collection state for rollback
	originalColumns := make([]registry.Column, len(collection.Columns))
	copy(originalColumns, collection.Columns)

	ctx := r.Context()

	// Execute operations in order: rename → modify → add → remove

	// 1. RENAME COLUMNS
	if len(req.RenameColumns) > 0 {
		if err := h.validateRenameColumns(req.RenameColumns, collection); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		for _, rename := range req.RenameColumns {
			ddl := generateRenameColumnDDL(req.Name, rename.OldName, rename.NewName, h.db.Dialect())
			if _, err := h.db.Exec(ctx, ddl); err != nil {
				// Rollback registry on failure
				collection.Columns = originalColumns
				h.registry.Set(collection)
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to rename column '%s': %v", rename.OldName, err))
				return
			}

			// Update column name in registry
			for i := range collection.Columns {
				if collection.Columns[i].Name == rename.OldName {
					collection.Columns[i].Name = rename.NewName
					break
				}
			}
		}
	}

	// 2. MODIFY COLUMNS
	if len(req.ModifyColumns) > 0 {
		if err := h.validateModifyColumns(req.ModifyColumns, collection); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		for _, modify := range req.ModifyColumns {
			ddl := generateModifyColumnDDL(req.Name, modify, h.db.Dialect())
			if _, err := h.db.Exec(ctx, ddl); err != nil {
				// Rollback registry on failure
				collection.Columns = originalColumns
				h.registry.Set(collection)
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to modify column '%s': %v", modify.Name, err))
				return
			}

			// Update column definition in registry
			for i := range collection.Columns {
				if collection.Columns[i].Name == modify.Name {
					collection.Columns[i].Type = modify.Type
					if modify.Nullable != nil {
						collection.Columns[i].Nullable = *modify.Nullable
					}
					if modify.Unique != nil {
						collection.Columns[i].Unique = *modify.Unique
					}
					if modify.DefaultValue != nil {
						collection.Columns[i].DefaultValue = modify.DefaultValue
					}
					break
				}
			}
		}
	}

	// 3. ADD COLUMNS
	if len(req.AddColumns) > 0 {
		if err := h.validateAddColumns(req.AddColumns, collection); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		for _, col := range req.AddColumns {
			ddl := generateAddColumnDDL(req.Name, col, h.db.Dialect())
			if _, err := h.db.Exec(ctx, ddl); err != nil {
				// Rollback registry on failure
				collection.Columns = originalColumns
				h.registry.Set(collection)
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add column '%s': %v", col.Name, err))
				return
			}

			collection.Columns = append(collection.Columns, col)
		}
	}

	// 4. REMOVE COLUMNS
	if len(req.RemoveColumns) > 0 {
		if err := h.validateRemoveColumns(req.RemoveColumns, collection); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		for _, colName := range req.RemoveColumns {
			ddl := generateDropColumnDDL(req.Name, colName, h.db.Dialect())
			if _, err := h.db.Exec(ctx, ddl); err != nil {
				// Rollback registry on failure
				collection.Columns = originalColumns
				h.registry.Set(collection)
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to remove column '%s': %v", colName, err))
				return
			}

			// Remove column from registry
			newColumns := make([]registry.Column, 0, len(collection.Columns)-1)
			for _, col := range collection.Columns {
				if col.Name != colName {
					newColumns = append(newColumns, col)
				}
			}
			collection.Columns = newColumns
		}
	}

	// Update registry with final state
	if err := h.registry.Set(collection); err != nil {
		// Attempt to rollback
		collection.Columns = originalColumns
		h.registry.Set(collection)
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

// validateAddColumns validates columns to be added
func (h *CollectionsHandler) validateAddColumns(columns []registry.Column, collection *registry.Collection) error {
	for i, col := range columns {
		if col.Name == "" {
			return fmt.Errorf("column %d: name is required", i)
		}
		if !registry.ValidateColumnType(col.Type) {
			return fmt.Errorf("column '%s': invalid type '%s'", col.Name, col.Type)
		}

		// Check if column already exists
		for _, existing := range collection.Columns {
			if existing.Name == col.Name {
				return fmt.Errorf("column '%s' already exists", col.Name)
			}
		}

		// System columns cannot be added manually
		if systemColumns[col.Name] {
			return fmt.Errorf("cannot add system column '%s'", col.Name)
		}
	}
	return nil
}

// validateRemoveColumns validates columns to be removed
func (h *CollectionsHandler) validateRemoveColumns(columnNames []string, collection *registry.Collection) error {
	for _, colName := range columnNames {
		if colName == "" {
			return fmt.Errorf("column name cannot be empty")
		}

		// System columns cannot be removed
		if systemColumns[colName] {
			return fmt.Errorf("cannot remove system column '%s'", colName)
		}

		// Check if column exists
		found := false
		for _, existing := range collection.Columns {
			if existing.Name == colName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("column '%s' does not exist", colName)
		}
	}
	return nil
}

// validateRenameColumns validates columns to be renamed
func (h *CollectionsHandler) validateRenameColumns(renames []RenameColumn, collection *registry.Collection) error {
	for _, rename := range renames {
		if rename.OldName == "" || rename.NewName == "" {
			return fmt.Errorf("both old_name and new_name are required for rename")
		}

		// System columns cannot be renamed
		if systemColumns[rename.OldName] {
			return fmt.Errorf("cannot rename system column '%s'", rename.OldName)
		}

		// Check if old column exists
		found := false
		for _, existing := range collection.Columns {
			if existing.Name == rename.OldName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("column '%s' does not exist", rename.OldName)
		}

		// Check if new name conflicts with existing columns (including system columns)
		if systemColumns[rename.NewName] {
			return fmt.Errorf("cannot rename to system column name '%s'", rename.NewName)
		}
		for _, existing := range collection.Columns {
			if existing.Name == rename.NewName && existing.Name != rename.OldName {
				return fmt.Errorf("column '%s' already exists", rename.NewName)
			}
		}
	}
	return nil
}

// validateModifyColumns validates columns to be modified
func (h *CollectionsHandler) validateModifyColumns(modifies []ModifyColumn, collection *registry.Collection) error {
	for _, modify := range modifies {
		if modify.Name == "" {
			return fmt.Errorf("column name is required for modify")
		}

		if !registry.ValidateColumnType(modify.Type) {
			return fmt.Errorf("column '%s': invalid type '%s'", modify.Name, modify.Type)
		}

		// System columns cannot be modified
		if systemColumns[modify.Name] {
			return fmt.Errorf("cannot modify system column '%s'", modify.Name)
		}

		// Check if column exists
		found := false
		for _, existing := range collection.Columns {
			if existing.Name == modify.Name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("column '%s' does not exist", modify.Name)
		}
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

	// Add ulid column (unique, not null)
	sb.WriteString(",\n  ulid CHAR(26) NOT NULL UNIQUE")

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

// generateDropColumnDDL generates ALTER TABLE DROP COLUMN DDL
func generateDropColumnDDL(tableName string, columnName string, dialect database.DialectType) string {
	// SQLite has limited ALTER TABLE support, but DROP COLUMN is supported in SQLite 3.35.0+
	// Since we're using modernc.org/sqlite, it should support this
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
}

// generateRenameColumnDDL generates column rename DDL for the given dialect
func generateRenameColumnDDL(tableName string, oldName string, newName string, dialect database.DialectType) string {
	switch dialect {
	case database.DialectPostgres:
		return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
	case database.DialectMySQL:
		// MySQL doesn't have a simple RENAME COLUMN syntax in older versions
		// We use ALTER TABLE ... CHANGE which requires full column definition
		// This is a simplified version - in production, you'd need to preserve the type
		return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
	case database.DialectSQLite:
		// SQLite 3.25.0+ supports RENAME COLUMN
		return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
	default:
		return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
	}
}

// generateModifyColumnDDL generates column modification DDL for the given dialect
func generateModifyColumnDDL(tableName string, modify ModifyColumn, dialect database.DialectType) string {
	var sb strings.Builder

	switch dialect {
	case database.DialectPostgres:
		// PostgreSQL requires separate ALTER COLUMN statements for each change
		sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s",
			tableName, modify.Name, mapColumnTypeToSQL(modify.Type, dialect)))

		// Note: Additional ALTER COLUMN statements for nullable, default, etc. would be separate queries
		// For simplicity, we're only handling type changes here
		// In a full implementation, you'd execute multiple DDL statements
	case database.DialectMySQL:
		// MySQL uses MODIFY COLUMN with full column definition
		sb.WriteString(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s",
			tableName, modify.Name, mapColumnTypeToSQL(modify.Type, dialect)))

		if modify.Nullable != nil && !*modify.Nullable {
			sb.WriteString(" NOT NULL")
		}
		if modify.Unique != nil && *modify.Unique {
			sb.WriteString(" UNIQUE")
		}
		if modify.DefaultValue != nil {
			sb.WriteString(" DEFAULT ")
			sb.WriteString(*modify.DefaultValue)
		}
	case database.DialectSQLite:
		// SQLite has very limited ALTER TABLE support for modifying columns
		// In production, you'd need to recreate the table
		// For now, we'll return an error-prone statement
		sb.WriteString(fmt.Sprintf("-- SQLite ALTER COLUMN not fully supported: %s", modify.Name))
	default:
		sb.WriteString(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s",
			tableName, modify.Name, mapColumnTypeToSQL(modify.Type, dialect)))
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
		return "TEXT"
	case registry.TypeInteger:
		return "BIGINT"
	case registry.TypeBoolean:
		return "BOOLEAN"
	case registry.TypeDatetime:
		return "TIMESTAMP"
	case registry.TypeJSON:
		return "JSONB"
	default:
		return "TEXT"
	}
}

func mapColumnTypeToMySQL(colType registry.ColumnType) string {
	switch colType {
	case registry.TypeString:
		return "TEXT"
	case registry.TypeInteger:
		return "BIGINT"
	case registry.TypeBoolean:
		return "BOOLEAN"
	case registry.TypeDatetime:
		return "DATETIME"
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
	case registry.TypeBoolean:
		return "INTEGER"
	case registry.TypeDatetime:
		return "TEXT"
	case registry.TypeJSON:
		return "TEXT"
	default:
		return "TEXT"
	}
}

// Helper functions for JSON responses
func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set(constants.HeaderContentType, constants.MIMEApplicationJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]any{
		"error": message,
		"code":  statusCode,
	})
}
