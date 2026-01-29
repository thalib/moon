package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/query"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// AggregationHandler handles aggregation operations on collection data
type AggregationHandler struct {
	db       database.Driver
	registry *registry.SchemaRegistry
}

// NewAggregationHandler creates a new aggregation handler
func NewAggregationHandler(db database.Driver, reg *registry.SchemaRegistry) *AggregationHandler {
	return &AggregationHandler{
		db:       db,
		registry: reg,
	}
}

// AggregationResponse represents response for aggregation operations
type AggregationResponse struct {
	Value any `json:"value"`
}

// Count handles GET /{name}:count
func (h *AggregationHandler) Count(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
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

	// Create query builder
	builder := query.NewBuilder(h.db.Dialect())

	// Build COUNT query
	sqlQuery, args := builder.Count(collectionName, conditions)

	// Execute query
	ctx := r.Context()
	var count int64
	err = h.db.QueryRow(ctx, sqlQuery, args...).Scan(&count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to execute count: %v", err))
		return
	}

	response := AggregationResponse{
		Value: count,
	}

	writeJSON(w, http.StatusOK, response)
}

// Sum handles GET /{name}:sum?field={field}
func (h *AggregationHandler) Sum(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Get field parameter
	field := r.URL.Query().Get("field")
	if field == "" {
		writeError(w, http.StatusBadRequest, "field parameter is required")
		return
	}

	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Validate field exists and is numeric
	if err := validateNumericField(collection, field); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
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

	// Create query builder
	builder := query.NewBuilder(h.db.Dialect())

	// Build SUM query
	sqlQuery, args := builder.Sum(collectionName, field, conditions)

	// Execute query
	ctx := r.Context()
	var sum sql.NullFloat64
	err = h.db.QueryRow(ctx, sqlQuery, args...).Scan(&sum)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to execute sum: %v", err))
		return
	}

	// Return 0 if no rows or NULL result
	var result float64
	if sum.Valid {
		result = sum.Float64
	}

	response := AggregationResponse{
		Value: result,
	}

	writeJSON(w, http.StatusOK, response)
}

// Avg handles GET /{name}:avg?field={field}
func (h *AggregationHandler) Avg(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Get field parameter
	field := r.URL.Query().Get("field")
	if field == "" {
		writeError(w, http.StatusBadRequest, "field parameter is required")
		return
	}

	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Validate field exists and is numeric
	if err := validateNumericField(collection, field); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
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

	// Create query builder
	builder := query.NewBuilder(h.db.Dialect())

	// Build AVG query
	sqlQuery, args := builder.Avg(collectionName, field, conditions)

	// Execute query
	ctx := r.Context()
	var avg sql.NullFloat64
	err = h.db.QueryRow(ctx, sqlQuery, args...).Scan(&avg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to execute avg: %v", err))
		return
	}

	// Return 0 if no rows or NULL result
	var result float64
	if avg.Valid {
		result = avg.Float64
	}

	response := AggregationResponse{
		Value: result,
	}

	writeJSON(w, http.StatusOK, response)
}

// Min handles GET /{name}:min?field={field}
func (h *AggregationHandler) Min(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Get field parameter
	field := r.URL.Query().Get("field")
	if field == "" {
		writeError(w, http.StatusBadRequest, "field parameter is required")
		return
	}

	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Validate field exists and is numeric
	if err := validateNumericField(collection, field); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
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

	// Create query builder
	builder := query.NewBuilder(h.db.Dialect())

	// Build MIN query
	sqlQuery, args := builder.Min(collectionName, field, conditions)

	// Execute query
	ctx := r.Context()
	var min sql.NullFloat64
	err = h.db.QueryRow(ctx, sqlQuery, args...).Scan(&min)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to execute min: %v", err))
		return
	}

	// Return 0 if no rows or NULL result
	var result float64
	if min.Valid {
		result = min.Float64
	}

	response := AggregationResponse{
		Value: result,
	}

	writeJSON(w, http.StatusOK, response)
}

// Max handles GET /{name}:max?field={field}
func (h *AggregationHandler) Max(w http.ResponseWriter, r *http.Request, collectionName string) {
	// Get field parameter
	field := r.URL.Query().Get("field")
	if field == "" {
		writeError(w, http.StatusBadRequest, "field parameter is required")
		return
	}

	// Validate collection exists in registry
	collection, exists := h.registry.Get(collectionName)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("collection '%s' not found", collectionName))
		return
	}

	// Validate field exists and is numeric
	if err := validateNumericField(collection, field); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
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

	// Create query builder
	builder := query.NewBuilder(h.db.Dialect())

	// Build MAX query
	sqlQuery, args := builder.Max(collectionName, field, conditions)

	// Execute query
	ctx := r.Context()
	var max sql.NullFloat64
	err = h.db.QueryRow(ctx, sqlQuery, args...).Scan(&max)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to execute max: %v", err))
		return
	}

	// Return 0 if no rows or NULL result
	var result float64
	if max.Valid {
		result = max.Float64
	}

	response := AggregationResponse{
		Value: result,
	}

	writeJSON(w, http.StatusOK, response)
}

// validateNumericField checks if a field exists and is numeric type
func validateNumericField(collection *registry.Collection, fieldName string) error {
	for _, col := range collection.Columns {
		if col.Name == fieldName {
			if col.Type == registry.TypeInteger || col.Type == registry.TypeFloat {
				return nil
			}
			return fmt.Errorf("field '%s' is not numeric (type: %s)", fieldName, col.Type)
		}
	}
	return fmt.Errorf("field '%s' not found in collection", fieldName)
}
