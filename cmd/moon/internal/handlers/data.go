package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/query"
	"github.com/thalib/moon/cmd/moon/internal/registry"
	"github.com/thalib/moon/cmd/moon/internal/schema"
	moonulid "github.com/thalib/moon/cmd/moon/internal/ulid"
)

// DataHandler handles CRUD operations on collection data
type DataHandler struct {
	db       database.Driver
	registry *registry.SchemaRegistry
	config   *config.AppConfig
}

// NewDataHandler creates a new data handler
func NewDataHandler(db database.Driver, reg *registry.SchemaRegistry, cfg *config.AppConfig) *DataHandler {
	return &DataHandler{
		db:       db,
		registry: reg,
		config:   cfg,
	}
}

// DataListRequest represents query parameters for list operation
type DataListRequest struct {
	Limit  int               `json:"limit"`
	After  string            `json:"after,omitempty"` // ULID cursor for pagination
	Filter map[string]string `json:"filter,omitempty"`
}

// DataListResponse represents response for list operation (PRD-062)
type DataListResponse struct {
	Data       []map[string]any `json:"data"`
	Total      int              `json:"total"`       // PRD-062: Total record count matching the query
	NextCursor *string          `json:"next_cursor"` // Next ULID cursor, null if no more data
	Limit      int              `json:"limit"`       // Always include pagination limit
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

// BatchCreateDataRequest represents request for batch create operation (PRD-064)
type BatchCreateDataRequest struct {
	Data json.RawMessage `json:"data"`
}

// BatchUpdateDataRequest represents request for batch update operation (PRD-064)
type BatchUpdateDataRequest struct {
	Data json.RawMessage `json:"data"`
}

// BatchDestroyDataRequest represents request for batch destroy operation (PRD-064)
type BatchDestroyDataRequest struct {
	Data json.RawMessage `json:"data"`
}

// BatchItemStatus represents the status of an individual item in a batch operation (PRD-064)
type BatchItemStatus string

const (
	BatchItemCreated  BatchItemStatus = "created"
	BatchItemUpdated  BatchItemStatus = "updated"
	BatchItemDeleted  BatchItemStatus = "deleted"
	BatchItemFailed   BatchItemStatus = "failed"
	BatchItemNotFound BatchItemStatus = "not_found"
)

// BatchItemResult represents the result of processing a single item in a batch (PRD-064)
type BatchItemResult struct {
	Index        int             `json:"index"`
	ID           string          `json:"id,omitempty"`
	Status       BatchItemStatus `json:"status"`
	Data         map[string]any  `json:"data,omitempty"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// BatchSummary represents summary statistics for a batch operation (PRD-064)
type BatchSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

// BatchResponse represents response for batch operations in partial success mode (PRD-064)
type BatchResponse struct {
	Results []BatchItemResult `json:"results"`
	Summary BatchSummary      `json:"summary"`
}

// BatchCreateResponse represents response for successful batch create operation (PRD-064)
type BatchCreateResponse struct {
	Data    []map[string]any `json:"data"`
	Message string           `json:"message"`
}

// BatchUpdateResponse represents response for successful batch update operation (PRD-064)
type BatchUpdateResponse struct {
	Data    []map[string]any `json:"data"`
	Message string           `json:"message"`
}

// BatchDestroyResponse represents response for successful batch destroy operation (PRD-064)
type BatchDestroyResponse struct {
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

	// Parse and validate limit (PRD-046)
	limit := constants.DefaultPaginationLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Enforce pagination limits (PRD-046)
	if limit < constants.MinPageSize {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("limit must be at least %d", constants.MinPageSize))
		return
	}
	if limit > constants.MaxPaginationLimit {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("limit cannot exceed %d", constants.MaxPaginationLimit))
		return
	}

	// Validate after cursor if provided
	if after != "" {
		if err := validateULID(after); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid cursor: %v", err))
			return
		}
	}

	// Parse filters from query parameters
	filters, err := parseFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid filter: %v", err))
		return
	}

	// Build conditions from filters
	conditions, err := buildConditions(filters, collection)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse search query
	searchQuery := r.URL.Query().Get("q")
	var searchSQL string
	var searchArgs []any
	if searchQuery != "" {
		// Validate search term
		if len(searchQuery) < 1 {
			writeError(w, http.StatusBadRequest, "search term must be at least 1 character")
			return
		}

		// Build search conditions (OR across all text columns)
		searchSQL, searchArgs = buildSearchConditions(searchQuery, collection, h.db.Dialect())
	}

	// Create query builder
	builder := query.NewBuilder(h.db.Dialect())

	// Calculate total count with current filters (PRD-062)
	// Must be done BEFORE adding cursor condition
	ctx := r.Context()
	var total int
	if searchSQL != "" {
		// Count query with search and filters (no cursor)
		countSQL, countArgs := buildCountQuery(collectionName, conditions, searchSQL, searchArgs, h.db.Dialect())
		row := h.db.QueryRow(ctx, countSQL, countArgs...)
		if err := row.Scan(&total); err != nil {
			// If count fails, default to 0
			total = 0
		}
	} else {
		// Count query without search (no cursor)
		countSQL, countArgs := builder.Count(collectionName, conditions)
		row := h.db.QueryRow(ctx, countSQL, countArgs...)
		if err := row.Scan(&total); err != nil {
			// If count fails, default to 0
			total = 0
		}
	}

	// Add cursor condition if provided (AFTER counting)
	if after != "" {
		conditions = append(conditions, query.Condition{
			Column:   "ulid",
			Operator: query.OpGreaterThan,
			Value:    after,
		})
	}

	// Parse sort parameters
	sorts, err := parseSort(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid sort parameter: %v", err))
		return
	}

	// Build ORDER BY clause
	orderBy, err := buildOrderBy(sorts, collection, builder)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse field selection
	fields, err := parseFields(r, collection)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build SELECT query
	var sql string
	var args []any
	if searchSQL != "" {
		// Manual query construction with search (OR) and filters (AND)
		sql, args = buildSearchQueryWithFields(collectionName, fields, conditions, searchSQL, searchArgs, orderBy, limit+1, h.db.Dialect())
	} else {
		// Use query builder for non-search queries
		sql, args = builder.Select(
			collectionName,
			fields,
			conditions,
			orderBy,
			limit+1, // Fetch one extra to determine if there's more data
			0,
		)
	}

	// Execute query
	rows, err := h.db.Query(ctx, sql, args...)
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
		// More data available, use the ULID of the last returned record as cursor
		// Truncate to limit first
		data = data[:limit]
		// Now get the last item from the returned data
		lastItem := data[len(data)-1]
		if ulidVal, ok := lastItem["id"].(string); ok {
			nextCursor = &ulidVal
		}
	}

	// Build response (PRD-062: include total)
	response := DataListResponse{
		Data:       data,
		Total:      total,
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

// Create handles POST /{name}:create - supports both single and batch modes (PRD-064)
func (h *DataHandler) Create(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Check payload size (PRD-064)
	if err := h.validatePayloadSize(r); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	// Parse request body with raw JSON to detect mode
	var batchReq BatchCreateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Detect batch vs single mode
	isBatch, err := detectBatchMode(batchReq.Data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !isBatch {
		// Single-object mode (backward compatible)
		h.createSingle(w, r, collectionName, collection, batchReq.Data)
		return
	}

	// Batch mode
	atomic := parseAtomicFlag(r)
	h.createBatch(w, r, collectionName, collection, batchReq.Data, atomic)
}

// createSingle handles single-object create (backward compatible)
func (h *DataHandler) createSingle(w http.ResponseWriter, r *http.Request, collectionName string, collection *registry.Collection, rawData json.RawMessage) {
	var data map[string]any
	if err := json.Unmarshal(rawData, &data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid data format")
		return
	}

	// Validate fields against schema
	if err := validateFields(data, collection); err != nil {
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
		if val, ok := data[col.Name]; ok {
			columns = append(columns, col.Name)
			if h.db.Dialect() == database.DialectPostgres {
				placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			} else {
				placeholders = append(placeholders, "?")
			}
			values = append(values, val)
			i++
		} else if !col.Nullable && col.DefaultValue == nil && col.Name != "ulid" {
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
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, fmt.Sprintf("unique constraint violation: %v", err))
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to insert data: %v", err))
		return
	}

	// Add ULID to response data (API field name is "id" but value is ULID)
	responseData := make(map[string]any)
	responseData["id"] = ulid
	for k, v := range data {
		responseData[k] = v
	}

	response := CreateDataResponse{
		Data:    responseData,
		Message: fmt.Sprintf("Record created successfully with id %s", ulid),
	}

	writeJSON(w, http.StatusCreated, response)
}

// createBatch handles batch create operations (PRD-064)
func (h *DataHandler) createBatch(w http.ResponseWriter, r *http.Request, collectionName string, collection *registry.Collection, rawData json.RawMessage, atomic bool) {
	var items []map[string]any
	if err := json.Unmarshal(rawData, &items); err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch data format")
		return
	}

	// Validate batch size
	if err := h.validateBatchSize(len(items)); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "batch must contain at least one item")
		return
	}

	ctx := r.Context()

	if atomic {
		// Atomic mode: all-or-nothing with transaction
		h.createBatchAtomic(w, ctx, collectionName, collection, items)
	} else {
		// Best-effort mode: partial success
		h.createBatchBestEffort(w, ctx, collectionName, collection, items)
	}
}

// createBatchAtomic handles atomic batch create with transaction (PRD-064)
func (h *DataHandler) createBatchAtomic(w http.ResponseWriter, ctx context.Context, collectionName string, collection *registry.Collection, items []map[string]any) {
	// Validate all items first
	for idx, item := range items {
		if err := validateFields(item, collection); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("validation error at index %d: %v", idx, err))
			return
		}
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to begin transaction: %v", err))
		return
	}
	defer tx.Rollback()

	var createdRecords []map[string]any

	// Insert each item
	for _, item := range items {
		ulid := generateULID()

		// Build INSERT query
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
			if val, ok := item[col.Name]; ok {
				columns = append(columns, col.Name)
				if h.db.Dialect() == database.DialectPostgres {
					placeholders = append(placeholders, fmt.Sprintf("$%d", i))
				} else {
					placeholders = append(placeholders, "?")
				}
				values = append(values, val)
				i++
			} else if !col.Nullable && col.DefaultValue == nil && col.Name != "ulid" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("required field '%s' is missing", col.Name))
				return
			}
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			collectionName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		// Execute insert within transaction
		_, err := tx.ExecContext(ctx, query, values...)
		if err != nil {
			// Check for unique constraint violations
			if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
				writeError(w, http.StatusConflict, fmt.Sprintf("unique constraint violation: %v", err))
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to insert data: %v", err))
			return
		}

		// Build response record
		responseData := make(map[string]any)
		responseData["id"] = ulid
		for k, v := range item {
			responseData[k] = v
		}
		createdRecords = append(createdRecords, responseData)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to commit transaction: %v", err))
		return
	}

	response := BatchCreateResponse{
		Data:    createdRecords,
		Message: fmt.Sprintf("%d records created successfully", len(createdRecords)),
	}

	writeJSON(w, http.StatusCreated, response)
}

// createBatchBestEffort handles best-effort batch create (PRD-064)
func (h *DataHandler) createBatchBestEffort(w http.ResponseWriter, ctx context.Context, collectionName string, collection *registry.Collection, items []map[string]any) {
	var results []BatchItemResult
	succeeded := 0
	failed := 0

	// Process each item independently
	for idx, item := range items {
		// Validate item
		if err := validateFields(item, collection); err != nil {
			results = append(results, BatchItemResult{
				Index:        idx,
				Status:       BatchItemFailed,
				ErrorCode:    "validation_error",
				ErrorMessage: err.Error(),
			})
			failed++
			continue
		}

		ulid := generateULID()

		// Build INSERT query
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

		requiredFieldMissing := false
		for _, col := range collection.Columns {
			if val, ok := item[col.Name]; ok {
				columns = append(columns, col.Name)
				if h.db.Dialect() == database.DialectPostgres {
					placeholders = append(placeholders, fmt.Sprintf("$%d", i))
				} else {
					placeholders = append(placeholders, "?")
				}
				values = append(values, val)
				i++
			} else if !col.Nullable && col.DefaultValue == nil && col.Name != "ulid" {
				results = append(results, BatchItemResult{
					Index:        idx,
					Status:       BatchItemFailed,
					ErrorCode:    "validation_error",
					ErrorMessage: fmt.Sprintf("required field '%s' is missing", col.Name),
				})
				failed++
				requiredFieldMissing = true
				break
			}
		}
		if requiredFieldMissing {
			continue
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			collectionName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		// Execute insert
		_, err := h.db.Exec(ctx, query, values...)
		if err != nil {
			// Check for unique constraint violations
			errorCode := "database_error"
			if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
				errorCode = "duplicate"
			}
			results = append(results, BatchItemResult{
				Index:        idx,
				Status:       BatchItemFailed,
				ErrorCode:    errorCode,
				ErrorMessage: err.Error(),
			})
			failed++
			continue
		}

		// Build response record
		responseData := make(map[string]any)
		responseData["id"] = ulid
		for k, v := range item {
			responseData[k] = v
		}

		results = append(results, BatchItemResult{
			Index:  idx,
			ID:     ulid,
			Status: BatchItemCreated,
			Data:   responseData,
		})
		succeeded++
	}

	response := BatchResponse{
		Results: results,
		Summary: BatchSummary{
			Total:     len(items),
			Succeeded: succeeded,
			Failed:    failed,
		},
	}

	writeJSON(w, http.StatusMultiStatus, response)
}

// Update handles POST /{name}:update
func (h *DataHandler) Update(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Check payload size (PRD-064)
	if err := h.validatePayloadSize(r); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	// Read body into buffer for multiple parses
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	bodyBytes := buf.Bytes()

	// Try to detect format: old format has "id" and "data" at root, new format has only "data" field
	var rawReq map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawReq); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if this is old format (has both "id" and "data" fields)
	_, hasID := rawReq["id"]
	dataField, hasData := rawReq["data"]

	if hasID && hasData {
		// Old format: {"id": "...", "data": {...}}
		var req UpdateDataRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		h.updateSingleLegacy(w, r, collectionName, collection, req)
		return
	}

	if !hasData {
		writeError(w, http.StatusBadRequest, "missing data field")
		return
	}

	// New format: detect batch vs single mode
	isBatch, err := detectBatchMode(dataField)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !isBatch {
		// Single-object mode (backward compatible)
		h.updateSingle(w, r, collectionName, collection, dataField)
		return
	}

	// Batch mode
	atomic := parseAtomicFlag(r)
	h.updateBatch(w, r, collectionName, collection, dataField, atomic)
}

// updateSingleLegacy handles single-object update in legacy format (backward compatible)
func (h *DataHandler) updateSingleLegacy(w http.ResponseWriter, r *http.Request, collectionName string, collection *registry.Collection, req UpdateDataRequest) {
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
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, fmt.Sprintf("unique constraint violation: %v", err))
			return
		}
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

// updateSingle handles single-object update in new format (backward compatible)
func (h *DataHandler) updateSingle(w http.ResponseWriter, r *http.Request, collectionName string, collection *registry.Collection, rawData json.RawMessage) {
	var item map[string]any
	if err := json.Unmarshal(rawData, &item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid data format")
		return
	}

	// Check for id field
	idVal, hasID := item["id"]
	if !hasID {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, ok := idVal.(string)
	if !ok {
		writeError(w, http.StatusBadRequest, "id must be a string")
		return
	}

	// Validate ULID format
	if err := validateULID(id); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid id: %v", err))
		return
	}

	// Validate fields against schema
	if err := validateFields(item, collection); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build UPDATE query
	setClauses := []string{}
	values := []any{}
	i := 1

	for _, col := range collection.Columns {
		if val, ok := item[col.Name]; ok {
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
	values = append(values, id)

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
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, fmt.Sprintf("unique constraint violation: %v", err))
			return
		}
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
		writeError(w, http.StatusNotFound, fmt.Sprintf("record with id %s not found", id))
		return
	}

	// Add ULID to response data (API field name is "id" but value is ULID)
	responseData := make(map[string]any)
	responseData["id"] = id
	for k, v := range item {
		if k != "id" {
			responseData[k] = v
		}
	}

	response := UpdateDataResponse{
		Data:    responseData,
		Message: fmt.Sprintf("Record %s updated successfully", id),
	}

	writeJSON(w, http.StatusOK, response)
}

// updateBatch handles batch update operations (PRD-064)
func (h *DataHandler) updateBatch(w http.ResponseWriter, r *http.Request, collectionName string, collection *registry.Collection, rawData json.RawMessage, atomic bool) {
	var items []map[string]any
	if err := json.Unmarshal(rawData, &items); err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch data format")
		return
	}

	// Validate batch size
	if err := h.validateBatchSize(len(items)); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "batch must contain at least one item")
		return
	}

	ctx := r.Context()

	if atomic {
		// Atomic mode: all-or-nothing with transaction
		h.updateBatchAtomic(w, ctx, collectionName, collection, items)
	} else {
		// Best-effort mode: partial success
		h.updateBatchBestEffort(w, ctx, collectionName, collection, items)
	}
}

// updateBatchAtomic handles atomic batch update with transaction (PRD-064)
func (h *DataHandler) updateBatchAtomic(w http.ResponseWriter, ctx context.Context, collectionName string, collection *registry.Collection, items []map[string]any) {
	// Validate all items first
	for idx, item := range items {
		// Check for id field
		idVal, hasID := item["id"]
		if !hasID {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("validation error at index %d: id is required", idx))
			return
		}
		id, ok := idVal.(string)
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("validation error at index %d: id must be a string", idx))
			return
		}
		// Validate ULID format
		if err := validateULID(id); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("validation error at index %d: invalid id: %v", idx, err))
			return
		}
		if err := validateFields(item, collection); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("validation error at index %d: %v", idx, err))
			return
		}
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to begin transaction: %v", err))
		return
	}
	defer tx.Rollback()

	var updatedRecords []map[string]any

	// Update each item
	for _, item := range items {
		id := item["id"].(string)

		// Build UPDATE query
		setClauses := []string{}
		values := []any{}
		i := 1

		for _, col := range collection.Columns {
			if val, ok := item[col.Name]; ok {
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
		values = append(values, id)

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

		// Execute update within transaction
		result, err := tx.ExecContext(ctx, query, values...)
		if err != nil {
			// Check for unique constraint violations
			if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
				writeError(w, http.StatusConflict, fmt.Sprintf("unique constraint violation: %v", err))
				return
			}
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
			writeError(w, http.StatusNotFound, fmt.Sprintf("record with id %s not found", id))
			return
		}

		// Build response record
		responseData := make(map[string]any)
		responseData["id"] = id
		for k, v := range item {
			if k != "id" {
				responseData[k] = v
			}
		}
		updatedRecords = append(updatedRecords, responseData)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to commit transaction: %v", err))
		return
	}

	response := BatchUpdateResponse{
		Data:    updatedRecords,
		Message: fmt.Sprintf("%d records updated successfully", len(updatedRecords)),
	}

	writeJSON(w, http.StatusOK, response)
}

// updateBatchBestEffort handles best-effort batch update (PRD-064)
func (h *DataHandler) updateBatchBestEffort(w http.ResponseWriter, ctx context.Context, collectionName string, collection *registry.Collection, items []map[string]any) {
	var results []BatchItemResult
	succeeded := 0
	failed := 0

	// Process each item independently
	for idx, item := range items {
		// Check for id field
		idVal, hasID := item["id"]
		if !hasID {
			results = append(results, BatchItemResult{
				Index:        idx,
				Status:       BatchItemFailed,
				ErrorCode:    "validation_error",
				ErrorMessage: "id is required",
			})
			failed++
			continue
		}
		id, ok := idVal.(string)
		if !ok {
			results = append(results, BatchItemResult{
				Index:        idx,
				Status:       BatchItemFailed,
				ErrorCode:    "validation_error",
				ErrorMessage: "id must be a string",
			})
			failed++
			continue
		}
		// Validate ULID format
		if err := validateULID(id); err != nil {
			results = append(results, BatchItemResult{
				Index:        idx,
				Status:       BatchItemFailed,
				ErrorCode:    "validation_error",
				ErrorMessage: fmt.Sprintf("invalid id: %v", err),
			})
			failed++
			continue
		}

		// Validate item
		if err := validateFields(item, collection); err != nil {
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemFailed,
				ErrorCode:    "validation_error",
				ErrorMessage: err.Error(),
			})
			failed++
			continue
		}

		// Build UPDATE query
		setClauses := []string{}
		values := []any{}
		i := 1

		for _, col := range collection.Columns {
			if val, ok := item[col.Name]; ok {
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
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemFailed,
				ErrorCode:    "validation_error",
				ErrorMessage: "no fields to update",
			})
			failed++
			continue
		}

		// Add ULID to values
		values = append(values, id)

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
		result, err := h.db.Exec(ctx, query, values...)
		if err != nil {
			// Check for unique constraint violations
			errorCode := "database_error"
			if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
				errorCode = "duplicate"
			}
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemFailed,
				ErrorCode:    errorCode,
				ErrorMessage: err.Error(),
			})
			failed++
			continue
		}

		// Check if any rows were affected
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemFailed,
				ErrorCode:    "database_error",
				ErrorMessage: fmt.Sprintf("failed to get rows affected: %v", err),
			})
			failed++
			continue
		}

		if rowsAffected == 0 {
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemNotFound,
				ErrorCode:    "not_found",
				ErrorMessage: fmt.Sprintf("record with id %s not found", id),
			})
			failed++
			continue
		}

		// Build response record
		responseData := make(map[string]any)
		responseData["id"] = id
		for k, v := range item {
			if k != "id" {
				responseData[k] = v
			}
		}

		results = append(results, BatchItemResult{
			Index:  idx,
			ID:     id,
			Status: BatchItemUpdated,
			Data:   responseData,
		})
		succeeded++
	}

	response := BatchResponse{
		Results: results,
		Summary: BatchSummary{
			Total:     len(items),
			Succeeded: succeeded,
			Failed:    failed,
		},
	}

	writeJSON(w, http.StatusMultiStatus, response)
}

// Destroy handles POST /{name}:destroy
func (h *DataHandler) Destroy(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	_, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Check payload size (PRD-064)
	if err := h.validatePayloadSize(r); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	// Read body into buffer for multiple parses
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	bodyBytes := buf.Bytes()

	// Try to detect format: old format has "id" at root, new format has "data" field
	var rawReq map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawReq); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if this is old format (has "id" field at root)
	_, hasID := rawReq["id"]
	dataField, hasData := rawReq["data"]

	if hasID && !hasData {
		// Old format: {"id": "..."}
		var req DestroyDataRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		h.destroySingleLegacy(w, r, collectionName, req)
		return
	}

	if !hasData {
		writeError(w, http.StatusBadRequest, "missing data field")
		return
	}

	// New format: detect batch vs single mode (array of IDs)
	isBatch, err := detectBatchMode(dataField)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !isBatch {
		// Single-object mode (backward compatible) - just a string ID
		var id string
		if err := json.Unmarshal(dataField, &id); err != nil {
			writeError(w, http.StatusBadRequest, "invalid data format")
			return
		}
		h.destroySingle(w, r, collectionName, id)
		return
	}

	// Batch mode
	atomic := parseAtomicFlag(r)
	h.destroyBatch(w, r, collectionName, dataField, atomic)
}

// destroySingleLegacy handles single-object destroy in legacy format (backward compatible)
func (h *DataHandler) destroySingleLegacy(w http.ResponseWriter, r *http.Request, collectionName string, req DestroyDataRequest) {
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

// destroySingle handles single-object destroy in new format (backward compatible)
func (h *DataHandler) destroySingle(w http.ResponseWriter, r *http.Request, collectionName string, id string) {
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Validate ULID format
	if err := validateULID(id); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid id: %v", err))
		return
	}

	// Build DELETE query using ULID
	query := fmt.Sprintf("DELETE FROM %s WHERE ulid = ?", collectionName)
	args := []any{id}

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
		writeError(w, http.StatusNotFound, fmt.Sprintf("record with id %s not found", id))
		return
	}

	response := DestroyDataResponse{
		Message: fmt.Sprintf("Record %s deleted successfully", id),
	}

	writeJSON(w, http.StatusOK, response)
}

// destroyBatch handles batch destroy operations (PRD-064)
func (h *DataHandler) destroyBatch(w http.ResponseWriter, r *http.Request, collectionName string, rawData json.RawMessage, atomic bool) {
	var ids []string
	if err := json.Unmarshal(rawData, &ids); err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch data format")
		return
	}

	// Validate batch size
	if err := h.validateBatchSize(len(ids)); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "batch must contain at least one id")
		return
	}

	ctx := r.Context()

	if atomic {
		// Atomic mode: all-or-nothing with transaction
		h.destroyBatchAtomic(w, ctx, collectionName, ids)
	} else {
		// Best-effort mode: partial success
		h.destroyBatchBestEffort(w, ctx, collectionName, ids)
	}
}

// destroyBatchAtomic handles atomic batch destroy with transaction (PRD-064)
func (h *DataHandler) destroyBatchAtomic(w http.ResponseWriter, ctx context.Context, collectionName string, ids []string) {
	// Validate all IDs first
	for idx, id := range ids {
		if err := validateULID(id); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("validation error at index %d: invalid id: %v", idx, err))
			return
		}
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to begin transaction: %v", err))
		return
	}
	defer tx.Rollback()

	// Delete each item
	for _, id := range ids {
		query := fmt.Sprintf("DELETE FROM %s WHERE ulid = ?", collectionName)
		args := []any{id}

		// Adjust placeholder style based on dialect
		if h.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("DELETE FROM %s WHERE ulid = $1", collectionName)
		}

		// Execute delete within transaction
		result, err := tx.ExecContext(ctx, query, args...)
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
			writeError(w, http.StatusNotFound, fmt.Sprintf("record with id %s not found", id))
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to commit transaction: %v", err))
		return
	}

	response := BatchDestroyResponse{
		Message: fmt.Sprintf("%d records deleted successfully", len(ids)),
	}

	writeJSON(w, http.StatusOK, response)
}

// destroyBatchBestEffort handles best-effort batch destroy (PRD-064)
func (h *DataHandler) destroyBatchBestEffort(w http.ResponseWriter, ctx context.Context, collectionName string, ids []string) {
	var results []BatchItemResult
	succeeded := 0
	failed := 0

	// Process each item independently
	for idx, id := range ids {
		// Validate ULID format
		if err := validateULID(id); err != nil {
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemFailed,
				ErrorCode:    "validation_error",
				ErrorMessage: fmt.Sprintf("invalid id: %v", err),
			})
			failed++
			continue
		}

		// Build DELETE query using ULID
		query := fmt.Sprintf("DELETE FROM %s WHERE ulid = ?", collectionName)
		args := []any{id}

		// Adjust placeholder style based on dialect
		if h.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("DELETE FROM %s WHERE ulid = $1", collectionName)
		}

		// Execute delete
		result, err := h.db.Exec(ctx, query, args...)
		if err != nil {
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemFailed,
				ErrorCode:    "database_error",
				ErrorMessage: err.Error(),
			})
			failed++
			continue
		}

		// Check if any rows were affected
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemFailed,
				ErrorCode:    "database_error",
				ErrorMessage: fmt.Sprintf("failed to get rows affected: %v", err),
			})
			failed++
			continue
		}

		if rowsAffected == 0 {
			results = append(results, BatchItemResult{
				Index:        idx,
				ID:           id,
				Status:       BatchItemNotFound,
				ErrorCode:    "not_found",
				ErrorMessage: fmt.Sprintf("record with id %s not found", id),
			})
			failed++
			continue
		}

		results = append(results, BatchItemResult{
			Index:  idx,
			ID:     id,
			Status: BatchItemDeleted,
		})
		succeeded++
	}

	response := BatchResponse{
		Results: results,
		Summary: BatchSummary{
			Total:     len(ids),
			Succeeded: succeeded,
			Failed:    failed,
		},
	}

	writeJSON(w, http.StatusMultiStatus, response)
}

// SchemaResponse represents the response for the schema endpoint (PRD-054, PRD-061)
type SchemaResponse struct {
	Collection string               `json:"collection"`
	Fields     []schema.FieldSchema `json:"fields"`
	Total      int                  `json:"total"` // PRD-061: Total record count in collection
}

// Schema handles GET /{name}:schema (PRD-054, PRD-061)
func (h *DataHandler) Schema(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, "Collection not found")
		return
	}

	// Build schema response
	schemaBuilder := schema.NewBuilder()
	fullSchema := schemaBuilder.FromCollection(collection)

	// Get total record count for the collection (PRD-061)
	ctx := r.Context()
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", collectionName)
	var total int
	row := h.db.QueryRow(ctx, countSQL)
	if err := row.Scan(&total); err != nil {
		// If error (e.g., table doesn't exist), default to 0
		total = 0
	}

	// Create response matching PRD-054 and PRD-061 specification
	response := SchemaResponse{
		Collection: fullSchema.Collection,
		Fields:     fullSchema.Fields,
		Total:      total,
	}

	writeJSON(w, http.StatusOK, response)
}

// filterParam represents a parsed filter from query string
type filterParam struct {
	column   string
	operator string
	value    string
}

// parseFilters parses filter query parameters from URL
// Expected format: ?column[operator]=value
// Example: ?price[gt]=100&name[like]=moon
// Enforces MaxFiltersPerRequest limit (PRD-048)
func parseFilters(r *http.Request) ([]filterParam, error) {
	var filters []filterParam
	filterRegex := regexp.MustCompile(`^(.+)\[(eq|ne|gt|lt|gte|lte|like|in)\]$`)

	for key, values := range r.URL.Query() {
		// Skip standard query params
		if key == constants.QueryParamLimit || key == "after" || key == "sort" || key == "q" || key == "fields" || key == "field" {
			continue
		}

		matches := filterRegex.FindStringSubmatch(key)
		if matches == nil {
			// Skip if not a filter parameter
			continue
		}

		// Check filter count limit (PRD-048)
		if len(filters) >= constants.MaxFiltersPerRequest {
			return nil, fmt.Errorf("maximum number of filters (%d) exceeded", constants.MaxFiltersPerRequest)
		}

		column := matches[1]
		operator := matches[2]

		if len(values) > 0 {
			filters = append(filters, filterParam{
				column:   column,
				operator: operator,
				value:    values[0],
			})
		}
	}

	return filters, nil
}

// mapOperatorToSQL maps short operator names to SQL operators
func mapOperatorToSQL(op string) string {
	switch op {
	case "eq":
		return query.OpEqual
	case "ne":
		return query.OpNotEqual
	case "gt":
		return query.OpGreaterThan
	case "lt":
		return query.OpLessThan
	case "gte":
		return query.OpGreaterThanOrEqual
	case "lte":
		return query.OpLessThanOrEqual
	case "like":
		return query.OpLike
	case "in":
		return query.OpIn
	default:
		return query.OpEqual
	}
}

// buildConditions converts filter params to query conditions
func buildConditions(filters []filterParam, collection *registry.Collection) ([]query.Condition, error) {
	var conditions []query.Condition

	// Create a map of valid column names
	validColumns := make(map[string]registry.Column)
	for _, col := range collection.Columns {
		validColumns[col.Name] = col
	}
	// Also allow filtering by ulid
	validColumns["ulid"] = registry.Column{Name: "ulid", Type: registry.TypeString}

	for _, filter := range filters {
		// Validate column exists in schema
		col, exists := validColumns[filter.column]
		if !exists {
			return nil, fmt.Errorf("invalid filter column: %s", filter.column)
		}

		sqlOp := mapOperatorToSQL(filter.operator)

		// Handle IN operator - split comma-separated values
		if sqlOp == query.OpIn {
			parts := strings.Split(filter.value, ",")
			values := make([]any, len(parts))
			for i, part := range parts {
				values[i] = strings.TrimSpace(part)
			}
			conditions = append(conditions, query.Condition{
				Column:   filter.column,
				Operator: sqlOp,
				Value:    values,
			})
		} else if sqlOp == query.OpLike {
			// For LIKE, wrap value with wildcards
			value := "%" + filter.value + "%"
			conditions = append(conditions, query.Condition{
				Column:   filter.column,
				Operator: sqlOp,
				Value:    value,
			})
		} else {
			// Convert value based on column type
			value, err := convertValue(filter.value, col.Type)
			if err != nil {
				return nil, fmt.Errorf("invalid value for column %s: %v", filter.column, err)
			}

			conditions = append(conditions, query.Condition{
				Column:   filter.column,
				Operator: sqlOp,
				Value:    value,
			})
		}
	}

	return conditions, nil
}

// convertValue converts a string value to the appropriate type
func convertValue(value string, colType registry.ColumnType) (any, error) {
	switch colType {
	case registry.TypeInteger:
		return strconv.ParseInt(value, 10, 64)
	case registry.TypeBoolean:
		return strconv.ParseBool(value)
	case registry.TypeString, registry.TypeDatetime, registry.TypeJSON:
		return value, nil
	default:
		return value, nil
	}
}

// sortField represents a parsed sort field with direction
type sortField struct {
	column    string
	direction string // "ASC" or "DESC"
}

// parseSort parses the sort query parameter
// Supports: ?sort=field (ASC), ?sort=-field (DESC), ?sort=field1,-field2 (multiple)
// Enforces MaxSortFieldsPerRequest limit (PRD-048)
func parseSort(r *http.Request) ([]sortField, error) {
	sortParam := r.URL.Query().Get("sort")
	if sortParam == "" {
		return nil, nil
	}

	var fields []sortField
	parts := strings.Split(sortParam, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check sort fields count limit (PRD-048)
		if len(fields) >= constants.MaxSortFieldsPerRequest {
			return nil, fmt.Errorf("maximum number of sort fields (%d) exceeded", constants.MaxSortFieldsPerRequest)
		}

		var field sortField
		if strings.HasPrefix(part, "-") {
			// Descending
			field.column = part[1:]
			field.direction = "DESC"
		} else if strings.HasPrefix(part, "+") {
			// Explicit ascending
			field.column = part[1:]
			field.direction = "ASC"
		} else {
			// Default ascending
			field.column = part
			field.direction = "ASC"
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// parseFields parses the fields query parameter
// Returns nil to select all fields, or a list of requested fields (always includes ulid)
func parseFields(r *http.Request, collection *registry.Collection) ([]string, error) {
	fieldsParam := r.URL.Query().Get("fields")
	if fieldsParam == "" {
		// No fields parameter, return nil to select all
		return nil, nil
	}

	// Parse comma-separated field names
	requestedFields := strings.Split(fieldsParam, ",")

	// Create a map of valid column names
	validColumns := make(map[string]bool)
	for _, col := range collection.Columns {
		validColumns[col.Name] = true
	}
	validColumns["ulid"] = true
	validColumns["id"] = true // Allow "id" as alias for ulid

	// Validate and collect fields
	fieldsMap := make(map[string]bool)
	for _, field := range requestedFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		// Map "id" to "ulid" internally
		if field == "id" {
			field = "ulid"
		}

		if !validColumns[field] {
			return nil, fmt.Errorf("invalid field: %s", field)
		}

		fieldsMap[field] = true
	}

	// Always include ulid for pagination consistency
	fieldsMap["ulid"] = true

	// Convert map to slice
	fields := make([]string, 0, len(fieldsMap))
	for field := range fieldsMap {
		fields = append(fields, field)
	}

	return fields, nil
}

// buildOrderBy constructs ORDER BY clause from sort fields
func buildOrderBy(sorts []sortField, collection *registry.Collection, builder query.Builder) (string, error) {
	if len(sorts) == 0 {
		// Default sorting by ulid
		return "ulid ASC", nil
	}

	// Create a map of valid column names
	validColumns := make(map[string]bool)
	for _, col := range collection.Columns {
		validColumns[col.Name] = true
	}
	// Also allow sorting by ulid
	validColumns["ulid"] = true

	var orderParts []string
	for _, sort := range sorts {
		// Validate column exists
		if !validColumns[sort.column] {
			return "", fmt.Errorf("invalid sort column: %s", sort.column)
		}

		// Escape identifier based on dialect
		escapedCol := sort.column
		switch builder.Dialect() {
		case database.DialectPostgres:
			escapedCol = fmt.Sprintf(`"%s"`, sort.column)
		case database.DialectMySQL:
			escapedCol = fmt.Sprintf("`%s`", sort.column)
		}

		orderParts = append(orderParts, fmt.Sprintf("%s %s", escapedCol, sort.direction))
	}

	return strings.Join(orderParts, ", "), nil
}

// buildSearchConditions builds search conditions for full-text search
// Returns SQL fragment and args for OR-connected LIKE conditions
func buildSearchConditions(searchTerm string, collection *registry.Collection, dialect database.DialectType) (string, []any) {
	// Escape LIKE wildcards in search term
	escapedTerm := strings.ReplaceAll(searchTerm, `\`, `\\`)
	escapedTerm = strings.ReplaceAll(escapedTerm, `%`, `\%`)
	escapedTerm = strings.ReplaceAll(escapedTerm, `_`, `\_`)

	// Wrap with wildcards for partial matching
	searchValue := "%" + escapedTerm + "%"

	// Find all string columns (full-text search on string fields)
	var textColumns []string
	for _, col := range collection.Columns {
		if col.Type == registry.TypeString {
			textColumns = append(textColumns, col.Name)
		}
	}

	if len(textColumns) == 0 {
		return "", nil
	}

	// Build OR conditions for each text column
	var conditions []string
	var args []any
	placeholderNum := 1

	for _, col := range textColumns {
		escapedCol := col
		switch dialect {
		case database.DialectPostgres:
			escapedCol = fmt.Sprintf(`"%s"`, col)
			conditions = append(conditions, fmt.Sprintf("%s LIKE $%d", escapedCol, placeholderNum))
		case database.DialectMySQL:
			escapedCol = fmt.Sprintf("`%s`", col)
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", escapedCol))
		case database.DialectSQLite:
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", col))
		}
		args = append(args, searchValue)
		placeholderNum++
	}

	searchSQL := "(" + strings.Join(conditions, " OR ") + ")"
	return searchSQL, args
}

// buildCountQuery builds COUNT query with search (OR) and filters (AND) (PRD-062)
func buildCountQuery(tableName string, filters []query.Condition, searchSQL string, searchArgs []any, dialect database.DialectType) (string, []any) {
	var sb strings.Builder
	args := []any{}

	// SELECT COUNT(*) clause
	sb.WriteString("SELECT COUNT(*) FROM ")

	// Escape table name
	switch dialect {
	case database.DialectPostgres:
		sb.WriteString(fmt.Sprintf(`"%s"`, tableName))
	case database.DialectMySQL:
		sb.WriteString(fmt.Sprintf("`%s`", tableName))
	default:
		sb.WriteString(tableName)
	}

	// WHERE clause
	sb.WriteString(" WHERE ")

	// Add search conditions first
	sb.WriteString(searchSQL)
	args = append(args, searchArgs...)

	// Add filter conditions with AND
	placeholderNum := len(searchArgs) + 1
	for _, cond := range filters {
		sb.WriteString(" AND ")

		// Escape column name
		escapedCol := cond.Column
		switch dialect {
		case database.DialectPostgres:
			escapedCol = fmt.Sprintf(`"%s"`, cond.Column)
		case database.DialectMySQL:
			escapedCol = fmt.Sprintf("`%s`", cond.Column)
		}

		sb.WriteString(escapedCol)
		sb.WriteString(" ")
		sb.WriteString(cond.Operator)
		sb.WriteString(" ")

		// Handle special operators
		if cond.Operator == query.OpIn {
			values, ok := cond.Value.([]any)
			if !ok {
				values = []any{cond.Value}
			}
			sb.WriteString("(")
			for j, v := range values {
				if j > 0 {
					sb.WriteString(", ")
				}
				if dialect == database.DialectPostgres {
					sb.WriteString(fmt.Sprintf("$%d", placeholderNum))
				} else {
					sb.WriteString("?")
				}
				args = append(args, v)
				placeholderNum++
			}
			sb.WriteString(")")
		} else {
			// Regular operators
			if dialect == database.DialectPostgres {
				sb.WriteString(fmt.Sprintf("$%d", placeholderNum))
			} else {
				sb.WriteString("?")
			}
			args = append(args, cond.Value)
			placeholderNum++
		}
	}

	return sb.String(), args
}

// buildSearchQueryWithFields builds complete SELECT query with field selection, search (OR) and filters (AND)
func buildSearchQueryWithFields(tableName string, fields []string, filters []query.Condition, searchSQL string, searchArgs []any, orderBy string, limit int, dialect database.DialectType) (string, []any) {
	var sb strings.Builder
	args := []any{}

	// SELECT clause
	sb.WriteString("SELECT ")
	if len(fields) == 0 {
		sb.WriteString("*")
	} else {
		for i, field := range fields {
			if i > 0 {
				sb.WriteString(", ")
			}
			// Escape field name
			switch dialect {
			case database.DialectPostgres:
				sb.WriteString(fmt.Sprintf(`"%s"`, field))
			case database.DialectMySQL:
				sb.WriteString(fmt.Sprintf("`%s`", field))
			default:
				sb.WriteString(field)
			}
		}
	}
	sb.WriteString(" FROM ")

	// Escape table name
	switch dialect {
	case database.DialectPostgres:
		sb.WriteString(fmt.Sprintf(`"%s"`, tableName))
	case database.DialectMySQL:
		sb.WriteString(fmt.Sprintf("`%s`", tableName))
	default:
		sb.WriteString(tableName)
	}

	// WHERE clause
	sb.WriteString(" WHERE ")

	// Add search conditions first
	sb.WriteString(searchSQL)
	args = append(args, searchArgs...)

	// Add filter conditions with AND
	placeholderNum := len(searchArgs) + 1
	for _, cond := range filters {
		sb.WriteString(" AND ")

		// Escape column name
		escapedCol := cond.Column
		switch dialect {
		case database.DialectPostgres:
			escapedCol = fmt.Sprintf(`"%s"`, cond.Column)
		case database.DialectMySQL:
			escapedCol = fmt.Sprintf("`%s`", cond.Column)
		}

		sb.WriteString(escapedCol)
		sb.WriteString(" ")
		sb.WriteString(cond.Operator)
		sb.WriteString(" ")

		// Handle special operators
		if cond.Operator == query.OpIn {
			values, ok := cond.Value.([]any)
			if !ok {
				values = []any{cond.Value}
			}
			sb.WriteString("(")
			for j, v := range values {
				if j > 0 {
					sb.WriteString(", ")
				}
				if dialect == database.DialectPostgres {
					sb.WriteString(fmt.Sprintf("$%d", placeholderNum))
				} else {
					sb.WriteString("?")
				}
				args = append(args, v)
				placeholderNum++
			}
			sb.WriteString(")")
		} else {
			if dialect == database.DialectPostgres {
				sb.WriteString(fmt.Sprintf("$%d", placeholderNum))
			} else {
				sb.WriteString("?")
			}
			args = append(args, cond.Value)
			placeholderNum++
		}
	}

	// ORDER BY clause
	if orderBy != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(orderBy)
	}

	// LIMIT clause
	if limit > 0 {
		sb.WriteString(" LIMIT ")
		if dialect == database.DialectPostgres {
			sb.WriteString(fmt.Sprintf("$%d", placeholderNum))
		} else {
			sb.WriteString("?")
		}
		args = append(args, limit)
	}

	return sb.String(), args
}

// parseRows parses SQL rows into a slice of maps
func parseRows(rows *sql.Rows, collection *registry.Collection) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Create a map of column names to their types for boolean conversion (PRD-051)
	columnTypes := make(map[string]registry.ColumnType)
	for _, col := range collection.Columns {
		columnTypes[col.Name] = col.Type
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

			// Convert boolean values (PRD-051: Boolean API Response Uniformity)
			// SQLite stores booleans as integers (0/1), we need to convert to true/false
			if colType, exists := columnTypes[col]; exists && colType == registry.TypeBoolean {
				val = convertToBoolean(val)
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

// convertToBoolean converts various boolean representations to Go bool (PRD-051)
func convertToBoolean(val any) bool {
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case string:
		// Handle string representations
		return v == "1" || v == "true" || v == "TRUE" || v == "t" || v == "T"
	default:
		return false
	}
}

// validateFields validates request data against collection schema
func validateFields(data map[string]any, collection *registry.Collection) error {
	// Check for unknown fields
	validFields := make(map[string]bool)
	for _, col := range collection.Columns {
		validFields[col.Name] = true
	}
	// Allow system columns (id, ulid) in request data
	validFields["id"] = true
	validFields["ulid"] = true

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
	case registry.TypeString, registry.TypeDatetime:
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

// detectBatchMode detects whether the request is for single or batch operation (PRD-064)
// Returns true if data is an array, false if it's a single object or string
func detectBatchMode(rawData json.RawMessage) (bool, error) {
	// Trim whitespace
	trimmed := bytes.TrimSpace(rawData)
	if len(trimmed) == 0 {
		return false, fmt.Errorf("empty data field")
	}

	// Check first character to determine if it's an array
	if trimmed[0] == '[' {
		return true, nil
	}
	// Single object or string (for destroy with single ID)
	if trimmed[0] == '{' || trimmed[0] == '"' {
		return false, nil
	}

	return false, fmt.Errorf("invalid data format: expected object, string, or array")
}

// parseAtomicFlag parses the atomic query parameter (PRD-064)
// Returns false (best-effort mode) by default if not specified
// Set atomic=true or atomic=1 to enable atomic mode (all-or-nothing)
func parseAtomicFlag(r *http.Request) bool {
	atomicStr := r.URL.Query().Get("atomic")
	if atomicStr == "" {
		return false // Default to best-effort mode
	}
	return atomicStr == "true" || atomicStr == "1"
}

// validateBatchSize checks if batch size is within configured limits (PRD-064)
func (h *DataHandler) validateBatchSize(size int) error {
	maxSize := h.config.Batch.MaxSize
	if size > maxSize {
		return fmt.Errorf("batch size %d exceeds limit of %d", size, maxSize)
	}
	return nil
}

// validatePayloadSize checks if payload size is within configured limits (PRD-064)
func (h *DataHandler) validatePayloadSize(r *http.Request) error {
	maxSize := int64(h.config.Batch.MaxPayloadBytes)
	// ContentLength can be -1 if not provided by the client
	// In that case, we'll let it through and rely on batch size limit
	if r.ContentLength > 0 && r.ContentLength > maxSize {
		return fmt.Errorf("payload size %d exceeds limit of %d bytes", r.ContentLength, maxSize)
	}
	return nil
}
