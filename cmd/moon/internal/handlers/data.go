package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

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
