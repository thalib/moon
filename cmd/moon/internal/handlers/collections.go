// Package handlers provides HTTP handlers for schema and data management.
// It implements the AIP-136 custom actions pattern with colon separators
// for RESTful API endpoints as specified in SPEC.md.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

var (
	// collectionNameRegex validates collection names.
	// Pattern: Must start with a letter, followed by letters, numbers, or underscores.
	collectionNameRegex = regexp.MustCompile(constants.CollectionNamePattern)

	// columnNameRegex validates column names (lowercase only).
	// Pattern: Must start with a lowercase letter, followed by lowercase letters, numbers, or underscores.
	columnNameRegex = regexp.MustCompile(constants.ColumnNamePattern)

	// System columns that cannot be added, removed, or renamed
	systemColumns = map[string]bool{
		"pkid": true,
		"id":   true,
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

// CollectionItem represents a collection with its metadata
type CollectionItem struct {
	Name    string `json:"name"`
	Records int    `json:"records"`
}

// ListResponse represents the response for listing collections
type ListResponse struct {
	Collections []CollectionItem `json:"collections"`
	Count       int              `json:"count"`
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

// decodeCreateRequest decodes a CreateRequest and validates that no default fields are present
func decodeCreateRequest(body io.Reader, req *CreateRequest) error {
	// Read body into buffer so we can parse it twice
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("invalid request body")
	}

	// First, validate for forbidden default fields
	if err := validateNoDefaultFields(bodyBytes); err != nil {
		return err
	}

	// Decode into the target struct
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(req); err != nil {
		return fmt.Errorf("invalid request body")
	}

	return nil
}

// decodeUpdateRequest decodes an UpdateRequest and validates that no default fields are present
func decodeUpdateRequest(body io.Reader, req *UpdateRequest) error {
	// Read body into buffer so we can parse it twice
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("invalid request body")
	}

	// First, validate for forbidden default fields
	if err := validateNoDefaultFields(bodyBytes); err != nil {
		return err
	}

	// Decode into the target struct
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(req); err != nil {
		return fmt.Errorf("invalid request body")
	}

	return nil
}

// validateNoDefaultFields checks if any default or default_value fields are present in the JSON
func validateNoDefaultFields(data []byte) error {
	// Parse as generic JSON to check for forbidden fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid request body")
	}

	// Check columns array if present
	if columns, ok := raw["columns"].([]any); ok {
		for i, col := range columns {
			if colMap, ok := col.(map[string]any); ok {
				if _, hasDefault := colMap["default"]; hasDefault {
					return fmt.Errorf("unknown field 'default' in columns[%d]", i)
				}
				if _, hasDefaultValue := colMap["default_value"]; hasDefaultValue {
					return fmt.Errorf("unknown field 'default_value' in columns[%d]", i)
				}
			}
		}
	}

	// Check add_columns array if present (for update requests)
	if addColumns, ok := raw["add_columns"].([]any); ok {
		for i, col := range addColumns {
			if colMap, ok := col.(map[string]any); ok {
				if _, hasDefault := colMap["default"]; hasDefault {
					return fmt.Errorf("unknown field 'default' in add_columns[%d]", i)
				}
				if _, hasDefaultValue := colMap["default_value"]; hasDefaultValue {
					return fmt.Errorf("unknown field 'default_value' in add_columns[%d]", i)
				}
			}
		}
	}

	// Check modify_columns array if present (for update requests)
	if modifyColumns, ok := raw["modify_columns"].([]any); ok {
		for i, col := range modifyColumns {
			if colMap, ok := col.(map[string]any); ok {
				if _, hasDefault := colMap["default"]; hasDefault {
					return fmt.Errorf("unknown field 'default' in modify_columns[%d]", i)
				}
				if _, hasDefaultValue := colMap["default_value"]; hasDefaultValue {
					return fmt.Errorf("unknown field 'default_value' in modify_columns[%d]", i)
				}
			}
		}
	}

	return nil
}

// List handles GET /collections:list
func (h *CollectionsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	allCollections := h.registry.List()

	// Filter out system tables and build collection items with record counts
	collections := make([]CollectionItem, 0, len(allCollections))
	for _, col := range allCollections {
		if !constants.IsSystemTable(col) {
			// Count records in this collection
			recordCount := h.getRecordCount(ctx, col)
			collections = append(collections, CollectionItem{
				Name:    col,
				Records: recordCount,
			})
		}
	}

	response := ListResponse{
		Collections: collections,
		Count:       len(collections),
	}

	writeJSON(w, http.StatusOK, response)
}

// getRecordCount returns the number of records in a collection
// Returns -1 if count cannot be retrieved (with warning log)
func (h *CollectionsHandler) getRecordCount(ctx context.Context, collectionName string) int {
	// Verify collection exists in registry (extra safety check)
	if !h.registry.Exists(collectionName) {
		log.Printf("WARNING: Attempted to count records for non-existent collection '%s'", collectionName)
		return -1
	}

	// Quote identifier based on dialect for defense-in-depth
	quotedName := h.quoteIdentifier(collectionName)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedName)
	var count int
	err := h.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		// Log warning but continue with -1 as per PRD requirement
		log.Printf("WARNING: Failed to count records for collection '%s': %v", collectionName, err)
		return -1
	}
	return count
}

// quoteIdentifier quotes an identifier (table/column name) based on database dialect
func (h *CollectionsHandler) quoteIdentifier(name string) string {
	switch h.db.Dialect() {
	case database.DialectPostgres:
		// PostgreSQL uses double quotes
		return fmt.Sprintf(`"%s"`, name)
	case database.DialectMySQL:
		// MySQL uses backticks
		return fmt.Sprintf("`%s`", name)
	case database.DialectSQLite:
		// SQLite supports double quotes and backticks, prefer double quotes
		return fmt.Sprintf(`"%s"`, name)
	default:
		// Fallback: double quotes (SQL standard)
		return fmt.Sprintf(`"%s"`, name)
	}
}

// Get handles GET /collections:get
func (h *CollectionsHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "collection name is required")
		return
	}

	// Normalize collection name to lowercase for lookup (PRD-047)
	name = strings.ToLower(name)

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
	if err := decodeCreateRequest(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Normalize collection name to lowercase (PRD-047)
	req.Name = strings.ToLower(req.Name)

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

	// Check collection count limit (PRD-048)
	if err := validateCollectionCount(h.registry); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Validate columns
	if len(req.Columns) == 0 {
		writeError(w, http.StatusBadRequest, "at least one column is required")
		return
	}

	// Check column count limit (PRD-048)
	// Total includes system columns (id, ulid) plus user-defined columns
	if len(req.Columns)+constants.SystemColumnsCount > constants.MaxColumnsPerCollection {
		writeError(w, http.StatusConflict, fmt.Sprintf("maximum number of columns (%d) exceeded", constants.MaxColumnsPerCollection))
		return
	}

	for i, col := range req.Columns {
		if col.Name == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("column %d: name is required", i))
			return
		}

		// Validate column name (PRD-048)
		if err := validateColumnName(col.Name); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("column '%s': %v", col.Name, err))
			return
		}

		// Validate column type with deprecated type checking (PRD-048)
		if err := validateColumnType(string(col.Type)); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("column '%s': %v", col.Name, err))
			return
		}

		// Validate default value if provided (PRD-048)
		if err := validateDefaultValue(&req.Columns[i]); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Apply type-based defaults for nullable fields if not explicitly set
		applyColumnDefaults(&req.Columns[i])
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
	if err := decodeUpdateRequest(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Normalize collection name to lowercase (PRD-047)
	req.Name = strings.ToLower(req.Name)

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

	// Normalize collection name to lowercase (PRD-047)
	req.Name = strings.ToLower(req.Name)

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

// validateCollectionName validates a collection name against all PRD-047 and PRD-048 rules.
// Rules applied:
// 1. Name cannot be empty
// 2. Length must be between 2 and 63 characters
// 3. Cannot be a reserved endpoint name (case-insensitive)
// 4. Must match pattern: start with a letter, contain only letters, numbers, and underscores
// 5. Cannot be a SQL reserved keyword
// 6. Cannot start with 'moon_' or be 'moon' (system prefix/namespace)
func validateCollectionName(name string) error {
	// 1. Empty check
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	// 2. Length validation
	if len(name) < constants.MinCollectionNameLength {
		return fmt.Errorf("collection name must be at least %d characters", constants.MinCollectionNameLength)
	}
	if len(name) > constants.MaxCollectionNameLength {
		return fmt.Errorf("collection name must not exceed %d characters", constants.MaxCollectionNameLength)
	}

	// 3. Reserved endpoint check (case-insensitive)
	if constants.IsReservedEndpointName(name) {
		return fmt.Errorf("collection name '%s' is reserved for system endpoints", name)
	}

	// 4. Pattern validation
	if !collectionNameRegex.MatchString(name) {
		return fmt.Errorf("collection name must start with a letter and contain only letters, numbers, and underscores")
	}

	// 5. Reserved keyword check (case-insensitive)
	if constants.IsReservedKeyword(name) {
		return fmt.Errorf("'%s' is a reserved keyword and cannot be used as a collection name", name)
	}

	// 6. System table/prefix check
	if constants.IsSystemTableOrPrefix(name) {
		return fmt.Errorf("collection name cannot start with 'moon_' or be 'moon' (reserved for system tables)")
	}

	return nil
}

// validateCollectionCount checks if the collection count limit has been reached.
func validateCollectionCount(reg *registry.SchemaRegistry) error {
	count := reg.Count()
	if count >= constants.MaxCollectionsPerServer {
		return fmt.Errorf("maximum number of collections (%d) reached", constants.MaxCollectionsPerServer)
	}
	return nil
}

// validateColumnName validates a column name against PRD-048 rules.
// Rules applied:
// 1. Name cannot be empty
// 2. Length must be between 3 and 63 characters
// 3. Cannot be a system column (id, ulid)
// 4. Must match pattern: start with lowercase letter, contain only lowercase letters, numbers, and underscores
// 5. Cannot be a SQL reserved keyword
func validateColumnName(name string) error {
	// 1. Empty check
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	// 2. Length validation
	if len(name) < constants.MinColumnNameLength {
		return fmt.Errorf("column name must be at least %d characters", constants.MinColumnNameLength)
	}
	if len(name) > constants.MaxColumnNameLength {
		return fmt.Errorf("column name must not exceed %d characters", constants.MaxColumnNameLength)
	}

	// 3. System column check
	if systemColumns[name] {
		return fmt.Errorf("cannot add system column '%s'", name)
	}

	// 4. Pattern validation (lowercase only)
	if !columnNameRegex.MatchString(name) {
		return fmt.Errorf("column name must start with a lowercase letter and contain only lowercase letters, numbers, and underscores")
	}

	// 5. Reserved keyword check
	if constants.IsReservedKeyword(name) {
		return fmt.Errorf("'%s' is a reserved keyword and cannot be used as a column name", name)
	}

	return nil
}

// validateColumnCount checks if adding more columns would exceed the limit.
func validateColumnCount(collection *registry.Collection, addingCount int) error {
	// collection.Columns does not include system columns, so add SystemColumnsCount
	totalColumns := len(collection.Columns) + constants.SystemColumnsCount + addingCount
	if totalColumns > constants.MaxColumnsPerCollection {
		return fmt.Errorf("maximum number of columns (%d) reached for collection '%s'",
			constants.MaxColumnsPerCollection, collection.Name)
	}
	return nil
}

// validateColumnType validates a column type with deprecated type checking.
func validateColumnType(typeStr string) error {
	// Check for deprecated types first
	switch strings.ToLower(typeStr) {
	case "text":
		return fmt.Errorf("type 'text' is deprecated and no longer supported. Use 'string' instead")
	case "float":
		return fmt.Errorf("type 'float' is deprecated and no longer supported. Use 'decimal' or 'integer' instead")
	}

	// Validate using registry's validation
	if !registry.ValidateColumnType(registry.ColumnType(typeStr)) {
		return fmt.Errorf("invalid column type '%s'. Supported types: string, integer, decimal, boolean, datetime, json", typeStr)
	}

	return nil
}

// applyColumnDefaults applies type-based defaults to nullable columns if not explicitly set.
// This ensures that nullable columns have database-level defaults during table creation.
func applyColumnDefaults(column *registry.Column) {
	// Only apply defaults for nullable fields
	if !column.Nullable {
		return
	}

	// If default is already set, don't override it
	if column.DefaultValue != nil {
		return
	}

	// Apply type-based defaults for nullable fields
	// Note: These are SQL DEFAULT values, so string types need quotes
	var defaultValue string
	switch column.Type {
	case registry.TypeString:
		defaultValue = "''"
	case registry.TypeInteger:
		defaultValue = "0"
	case registry.TypeDecimal:
		defaultValue = "'0.00'"
	case registry.TypeBoolean:
		defaultValue = "0" // SQLite uses 0/1 for boolean
	case registry.TypeDatetime:
		defaultValue = "NULL"
	case registry.TypeJSON:
		defaultValue = "'{}'"
	default:
		return
	}

	column.DefaultValue = &defaultValue
}

// validateDefaultValue validates a default value against column type.
func validateDefaultValue(column *registry.Column) error {
	if column.DefaultValue == nil {
		return nil // No default value specified
	}

	// Only nullable fields can have defaults
	if !column.Nullable {
		return fmt.Errorf("default values can only be set for nullable fields (column '%s' has nullable=false)", column.Name)
	}

	value := *column.DefaultValue

	// Check nullable constraint for null default
	if strings.ToLower(value) == "null" {
		return nil // "null" is always valid for nullable fields
	}

	// Validate format based on type
	switch column.Type {
	case registry.TypeString:
		// Any string is valid
		return nil

	case registry.TypeInteger:
		// Must be parseable as int64
		for i, c := range value {
			if i == 0 && c == '-' {
				continue
			}
			if c < '0' || c > '9' {
				return fmt.Errorf("default value '%s' is invalid for type 'integer'", value)
			}
		}
		return nil

	case registry.TypeDecimal:
		if err := validateDecimalFormat(value); err != nil {
			return fmt.Errorf("default value '%s' is invalid for type 'decimal': %v", value, err)
		}
		return nil

	case registry.TypeBoolean:
		lower := strings.ToLower(value)
		if lower != "true" && lower != "false" {
			return fmt.Errorf("default value '%s' is invalid for type 'boolean'. Use 'true' or 'false'", value)
		}
		return nil

	case registry.TypeDatetime:
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return fmt.Errorf("default value '%s' is invalid for type 'datetime'. Use RFC3339 format (e.g., '2024-01-01T00:00:00Z')", value)
		}
		return nil

	case registry.TypeJSON:
		// Basic JSON validation - check for valid JSON brackets/braces
		trimmed := strings.TrimSpace(value)
		if trimmed == "null" || trimmed == "true" || trimmed == "false" {
			return nil // Valid JSON literals
		}
		if len(trimmed) >= 2 {
			first, last := trimmed[0], trimmed[len(trimmed)-1]
			if (first == '{' && last == '}') || (first == '[' && last == ']') || (first == '"' && last == '"') {
				return nil // Basic structure check
			}
		}
		// Check if it's a number
		if _, err := fmt.Sscanf(trimmed, "%f", new(float64)); err == nil {
			return nil
		}
		return fmt.Errorf("default value '%s' is invalid JSON", value)

	default:
		return fmt.Errorf("unknown column type '%s'", column.Type)
	}
}

// validateDecimalFormat validates a decimal string format.
func validateDecimalFormat(value string) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}

	// Check for valid decimal format: optional sign, digits, optional decimal point and digits
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return fmt.Errorf("invalid decimal format")
	}

	// Validate integer part
	intPart := parts[0]
	if intPart == "" || intPart == "-" || intPart == "+" {
		return fmt.Errorf("invalid decimal format")
	}

	startIdx := 0
	if intPart[0] == '-' || intPart[0] == '+' {
		startIdx = 1
	}

	for i := startIdx; i < len(intPart); i++ {
		if intPart[i] < '0' || intPart[i] > '9' {
			return fmt.Errorf("invalid decimal format")
		}
	}

	// Validate decimal part if present
	if len(parts) == 2 {
		decPart := parts[1]
		if decPart == "" {
			return fmt.Errorf("trailing decimal point not allowed")
		}
		for _, c := range decPart {
			if c < '0' || c > '9' {
				return fmt.Errorf("invalid decimal format")
			}
		}
		// Check scale
		if len(decPart) > constants.DecimalMaxScale {
			return fmt.Errorf("decimal scale exceeds maximum (%d)", constants.DecimalMaxScale)
		}
	}

	return nil
}

// validateAddColumns validates columns to be added
func (h *CollectionsHandler) validateAddColumns(columns []registry.Column, collection *registry.Collection) error {
	// Check column count limit
	if err := validateColumnCount(collection, len(columns)); err != nil {
		return err
	}

	for i, col := range columns {
		if col.Name == "" {
			return fmt.Errorf("column %d: name is required", i)
		}

		// Validate column name
		if err := validateColumnName(col.Name); err != nil {
			return fmt.Errorf("column '%s': %v", col.Name, err)
		}

		// Validate column type with deprecated type checking
		if err := validateColumnType(string(col.Type)); err != nil {
			return fmt.Errorf("column '%s': %v", col.Name, err)
		}

		// Check if column already exists
		for _, existing := range collection.Columns {
			if existing.Name == col.Name {
				return fmt.Errorf("column '%s' already exists", col.Name)
			}
		}

		// Validate default value if provided
		if err := validateDefaultValue(&col); err != nil {
			return err
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

		// Validate new column name against PRD-048 rules
		if err := validateColumnName(rename.NewName); err != nil {
			return fmt.Errorf("new column name '%s': %v", rename.NewName, err)
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

		// Validate column type with deprecated type checking (PRD-048)
		if err := validateColumnType(string(modify.Type)); err != nil {
			return fmt.Errorf("column '%s': %v", modify.Name, err)
		}

		// System columns cannot be modified
		if systemColumns[modify.Name] {
			return fmt.Errorf("cannot modify system column '%s'", modify.Name)
		}

		// Check if column exists and validate default value changes
		found := false
		for _, existing := range collection.Columns {
			if existing.Name == modify.Name {
				found = true

				// Prevent changing default value after collection creation
				// to avoid data inconsistency and corruption
				if modify.DefaultValue != nil {
					existingDefault := ""
					if existing.DefaultValue != nil {
						existingDefault = *existing.DefaultValue
					}
					newDefault := *modify.DefaultValue
					if existingDefault != newDefault {
						return fmt.Errorf("cannot change default value for column '%s': default values are immutable after collection creation to prevent data inconsistency", modify.Name)
					}
				}
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

	// Add pkid column (auto-increment primary key)
	switch dialect {
	case database.DialectPostgres:
		sb.WriteString("\n  pkid SERIAL PRIMARY KEY")
	case database.DialectMySQL:
		sb.WriteString("\n  pkid INT AUTO_INCREMENT PRIMARY KEY")
	case database.DialectSQLite:
		sb.WriteString("\n  pkid INTEGER PRIMARY KEY AUTOINCREMENT")
	}

	// Add id column (ULID: unique, not null)
	sb.WriteString(",\n  id CHAR(26) NOT NULL UNIQUE")

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
