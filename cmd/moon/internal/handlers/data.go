package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
	moonulid "github.com/thalib/moon/cmd/moon/internal/ulid"
)

// DataHandler handles CRUD operations on collection data
type DataHandler struct {
	db       database.Driver
	registry *registry.SchemaRegistry
}

// NewDataHandler creates a new data handler
func NewDataHandler(db database.Driver, reg *registry.SchemaRegistry) *DataHandler {
	return &DataHandler{
		db:       db,
		registry: reg,
	}
}

// DataListRequest represents query parameters for list operation
type DataListRequest struct {
	Limit  int               `json:"limit"`
	After  string            `json:"after,omitempty"` // ULID cursor for pagination
	Filter map[string]string `json:"filter,omitempty"`
}

// DataListResponse represents response for list operation
type DataListResponse struct {
	Data       []map[string]any `json:"data"`
	NextCursor *string          `json:"next_cursor"` // Next ULID cursor, null if no more data
	Limit      int              `json:"limit"`
}

// DataGetResponse represents response for get operation
type DataGetResponse struct {
	Data map[string]any `json:"data"`
}

// CreateDataRequest represents request for create operation
type CreateDataRequest struct {
	Data map[string]any `json:"data"`
}

// CreateDataResponse represents response for create operation
type CreateDataResponse struct {
	Data    map[string]any `json:"data"`
	Message string         `json:"message"`
}

// UpdateDataRequest represents request for update operation
type UpdateDataRequest struct {
	ID   string         `json:"id"` // ULID
	Data map[string]any `json:"data"`
}

// UpdateDataResponse represents response for update operation
type UpdateDataResponse struct {
	Data    map[string]any `json:"data"`
	Message string         `json:"message"`
}

// DestroyDataRequest represents request for destroy operation
type DestroyDataRequest struct {
	ID string `json:"id"` // ULID
}

// DestroyDataResponse represents response for destroy operation
type DestroyDataResponse struct {
	Message string `json:"message"`
}

// List handles GET /{name}:list
func (h *DataHandler) List(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get(constants.QueryParamLimit)
	after := r.URL.Query().Get("after") // ULID cursor

	limit := constants.DefaultPaginationLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Validate after cursor if provided
	if after != "" {
		if err := validateULID(after); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid cursor: %v", err))
			return
		}
	}

	// Build SELECT query with ULID-based pagination
	var query string
	var args []any

	if after == "" {
		// No cursor, start from beginning
		query = fmt.Sprintf("SELECT * FROM %s ORDER BY ulid ASC LIMIT ?", collectionName)
		args = []any{limit + 1} // Fetch one extra to determine if there's more data
	} else {
		// With cursor, fetch records after the cursor
		query = fmt.Sprintf("SELECT * FROM %s WHERE ulid > ? ORDER BY ulid ASC LIMIT ?", collectionName)
		args = []any{after, limit + 1}
	}

	// Adjust placeholder style based on dialect
	if h.db.Dialect() == database.DialectPostgres {
		if after == "" {
			query = fmt.Sprintf("SELECT * FROM %s ORDER BY ulid ASC LIMIT $1", collectionName)
		} else {
			query = fmt.Sprintf("SELECT * FROM %s WHERE ulid > $1 ORDER BY ulid ASC LIMIT $2", collectionName)
		}
	}

	// Execute query
	ctx := r.Context()
	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to query data: %v", err))
		return
	}
	defer rows.Close()

	// Parse results
	data, err := parseRows(rows, collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to parse results: %v", err))
		return
	}

	// Determine next cursor
	var nextCursor *string
	if len(data) > limit {
		// More data available, use the ULID of the last item as cursor
		lastItem := data[len(data)-1]
		if ulidVal, ok := lastItem["ulid"].(string); ok {
			nextCursor = &ulidVal
		}
		// Remove the extra item we fetched
		data = data[:limit]
	}

	response := DataListResponse{
		Data:       data,
		NextCursor: nextCursor,
		Limit:      limit,
	}

	writeJSON(w, http.StatusOK, response)
}

// Get handles GET /{name}:get
func (h *DataHandler) Get(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Get ID from query parameter (ULID)
	idStr := r.URL.Query().Get(constants.QueryParamID)
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	// Validate ULID format
	if err := validateULID(idStr); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid id: %v", err))
		return
	}

	// Build SELECT query using ULID
	query := fmt.Sprintf("SELECT * FROM %s WHERE ulid = ?", collectionName)
	args := []any{idStr}

	// Adjust placeholder style based on dialect
	if h.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT * FROM %s WHERE ulid = $1", collectionName)
	}

	// Execute query
	ctx := r.Context()
	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to query data: %v", err))
		return
	}
	defer rows.Close()

	// Parse results
	data, err := parseRows(rows, collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to parse results: %v", err))
		return
	}

	if len(data) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("record with id %s not found", idStr))
		return
	}

	response := DataGetResponse{
		Data: data[0],
	}

	writeJSON(w, http.StatusOK, response)
}

// Create handles POST /{name}:create
func (h *DataHandler) Create(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Parse request body
	var req CreateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate fields against schema
	if err := validateFields(req.Data, collection); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Generate ULID for the new record
	ulid := generateULID()

	// Build INSERT query including ULID
	columns := []string{"ulid"}
	placeholders := []string{}
	values := []any{ulid}
	i := 1

	if h.db.Dialect() == database.DialectPostgres {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
	} else {
		placeholders = append(placeholders, "?")
	}
	i++

	for _, col := range collection.Columns {
		if val, ok := req.Data[col.Name]; ok {
			columns = append(columns, col.Name)
			if h.db.Dialect() == database.DialectPostgres {
				placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			} else {
				placeholders = append(placeholders, "?")
			}
			values = append(values, val)
			i++
		} else if !col.Nullable && col.DefaultValue == nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("required field '%s' is missing", col.Name))
			return
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		collectionName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// Execute insert
	ctx := r.Context()
	_, err := h.db.Exec(ctx, query, values...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to insert data: %v", err))
		return
	}

	// Add ULID to response data (API field name is "id" but value is ULID)
	responseData := make(map[string]any)
	responseData["id"] = ulid
	for k, v := range req.Data {
		responseData[k] = v
	}

	response := CreateDataResponse{
		Data:    responseData,
		Message: fmt.Sprintf("Record created successfully with id %s", ulid),
	}

	writeJSON(w, http.StatusCreated, response)
}

// Update handles POST /{name}:update
func (h *DataHandler) Update(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Parse request body
	var req UpdateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Validate ULID format
	if err := validateULID(req.ID); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid id: %v", err))
		return
	}

	// Validate fields against schema
	if err := validateFields(req.Data, collection); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build UPDATE query
	setClauses := []string{}
	values := []any{}
	i := 1

	for _, col := range collection.Columns {
		if val, ok := req.Data[col.Name]; ok {
			if h.db.Dialect() == database.DialectPostgres {
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col.Name, i))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = ?", col.Name))
			}
			values = append(values, val)
			i++
		}
	}

	if len(setClauses) == 0 {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	// Add ULID to values
	values = append(values, req.ID)

	var query string
	if h.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("UPDATE %s SET %s WHERE ulid = $%d",
			collectionName,
			strings.Join(setClauses, ", "),
			i)
	} else {
		query = fmt.Sprintf("UPDATE %s SET %s WHERE ulid = ?",
			collectionName,
			strings.Join(setClauses, ", "))
	}

	// Execute update
	ctx := r.Context()
	result, err := h.db.Exec(ctx, query, values...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update data: %v", err))
		return
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get rows affected: %v", err))
		return
	}

	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("record with id %s not found", req.ID))
		return
	}

	// Add ULID to response data (API field name is "id" but value is ULID)
	responseData := make(map[string]any)
	responseData["id"] = req.ID
	for k, v := range req.Data {
		responseData[k] = v
	}

	response := UpdateDataResponse{
		Data:    responseData,
		Message: fmt.Sprintf("Record %s updated successfully", req.ID),
	}

	writeJSON(w, http.StatusOK, response)
}

// Destroy handles POST /{name}:destroy
func (h *DataHandler) Destroy(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	_, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Parse request body
	var req DestroyDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Validate ULID format
	if err := validateULID(req.ID); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid id: %v", err))
		return
	}

	// Build DELETE query using ULID
	query := fmt.Sprintf("DELETE FROM %s WHERE ulid = ?", collectionName)
	args := []any{req.ID}

	// Adjust placeholder style based on dialect
	if h.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE ulid = $1", collectionName)
	}

	// Execute delete
	ctx := r.Context()
	result, err := h.db.Exec(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete data: %v", err))
		return
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get rows affected: %v", err))
		return
	}

	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("record with id %s not found", req.ID))
		return
	}

	response := DestroyDataResponse{
		Message: fmt.Sprintf("Record %s deleted successfully", req.ID),
	}

	writeJSON(w, http.StatusOK, response)
}

// parseRows parses SQL rows into a slice of maps
func parseRows(rows *sql.Rows, collection *registry.Collection) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := []map[string]any{}

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowData := make(map[string]any)
		for i, col := range columns {
			val := values[i]

			// Convert []byte to string for text fields
			if b, ok := val.([]byte); ok {
				val = string(b)
			}

			// Map internal 'id' column to nothing (never expose it)
			// Map 'ulid' column to 'id' in API response (per PRD requirement)
			if col == "id" {
				// Skip internal SQL id
				continue
			} else if col == "ulid" {
				// Expose ulid as 'id' in API
				rowData["id"] = val
			} else {
				rowData[col] = val
			}
		}

		result = append(result, rowData)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// validateFields validates request data against collection schema
func validateFields(data map[string]any, collection *registry.Collection) error {
	// Check for unknown fields
	validFields := make(map[string]bool)
	for _, col := range collection.Columns {
		validFields[col.Name] = true
	}

	for field := range data {
		if !validFields[field] {
			return fmt.Errorf("unknown field '%s'", field)
		}
	}

	// Validate field types
	for _, col := range collection.Columns {
		if val, ok := data[col.Name]; ok && val != nil {
			if err := validateFieldType(col.Name, val, col.Type); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateFieldType validates a field value against expected type
func validateFieldType(fieldName string, value any, expectedType registry.ColumnType) error {
	switch expectedType {
	case registry.TypeString, registry.TypeText, registry.TypeDatetime:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", fieldName)
		}
	case registry.TypeInteger:
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float64:
			// JSON numbers come as float64, accept them
		default:
			return fmt.Errorf("field '%s' must be an integer", fieldName)
		}
	case registry.TypeFloat:
		switch value.(type) {
		case float32, float64, int, int8, int16, int32, int64:
			// Accept integers as floats
		default:
			return fmt.Errorf("field '%s' must be a number", fieldName)
		}
	case registry.TypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean", fieldName)
		}
	case registry.TypeJSON:
		// JSON can be any type
	}

	return nil
}

// generateULID generates a new ULID
func generateULID() string {
	return moonulid.Generate()
}

// validateULID validates a ULID string
func validateULID(id string) error {
	return moonulid.Validate(id)
}
