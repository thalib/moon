// Package errors provides centralized error handling and HTTP error responses.
// It defines standard error codes, error types, and middleware for consistent
// error handling across the application.
package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"
	"github.com/thalib/moon/cmd/moon/internal/constants"
)

// ErrorCode represents a standard error code
type ErrorCode string

const (
	// Validation errors (PRD-049)
	CodeValidationFailed      ErrorCode = "VALIDATION_ERROR"
	CodeValidationError       ErrorCode = "VALIDATION_ERROR" // alias
	CodeInvalidInput          ErrorCode = "INVALID_INPUT"
	CodeMissingField          ErrorCode = "MISSING_FIELD"
	CodeRequiredField         ErrorCode = "REQUIRED_FIELD"
	CodeInvalidType           ErrorCode = "INVALID_TYPE"
	CodeInvalidJSON           ErrorCode = "INVALID_JSON"
	CodeInvalidULID           ErrorCode = "INVALID_ULID"
	CodeInvalidCursor         ErrorCode = "INVALID_CURSOR"
	CodePageSizeExceeded      ErrorCode = "PAGE_SIZE_EXCEEDED"
	CodeFiltersExceeded       ErrorCode = "FILTERS_EXCEEDED"
	CodeSortFieldsExceeded    ErrorCode = "SORT_FIELDS_EXCEEDED"
	CodeCollectionNameInvalid ErrorCode = "COLLECTION_NAME_INVALID"
	CodeColumnNameInvalid     ErrorCode = "COLUMN_NAME_INVALID"
	CodeReservedName          ErrorCode = "RESERVED_NAME"
	CodeDeprecatedType        ErrorCode = "DEPRECATED_TYPE"

	// Authentication errors
	CodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	CodeInvalidToken  ErrorCode = "INVALID_TOKEN"
	CodeTokenExpired  ErrorCode = "TOKEN_EXPIRED"
	CodeMissingToken  ErrorCode = "MISSING_TOKEN"
	CodeInvalidAPIKey ErrorCode = "INVALID_API_KEY"
	CodeMissingAPIKey ErrorCode = "MISSING_API_KEY"

	// Authorization errors
	CodeForbidden               ErrorCode = "FORBIDDEN"
	CodeInsufficientPermissions ErrorCode = "INSUFFICIENT_PERMISSIONS"

	// Resource errors (PRD-049)
	CodeNotFound              ErrorCode = "NOT_FOUND"
	CodeResourceNotFound      ErrorCode = "RESOURCE_NOT_FOUND"
	CodeCollectionNotFound    ErrorCode = "COLLECTION_NOT_FOUND"
	CodeRecordNotFound        ErrorCode = "RECORD_NOT_FOUND"
	CodeAlreadyExists         ErrorCode = "ALREADY_EXISTS"
	CodeConflict              ErrorCode = "CONFLICT"
	CodeDuplicateCollection   ErrorCode = "DUPLICATE_COLLECTION"
	CodeUniqueViolation       ErrorCode = "UNIQUE_CONSTRAINT_VIOLATION"
	CodeMaxCollectionsReached ErrorCode = "MAX_COLLECTIONS_REACHED"
	CodeMaxColumnsReached     ErrorCode = "MAX_COLUMNS_REACHED"

	// Server errors (PRD-049)
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
	CodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	CodeQueryTimeout       ErrorCode = "QUERY_TIMEOUT"

	// Request errors
	CodeBadRequest        ErrorCode = "BAD_REQUEST"
	CodeMethodNotAllowed  ErrorCode = "METHOD_NOT_ALLOWED"
	CodeTooManyRequests   ErrorCode = "TOO_MANY_REQUESTS"
	CodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error     string         `json:"error"`
	Code      int            `json:"code"`
	ErrorCode ErrorCode      `json:"error_code,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// APIError represents an application error
type APIError struct {
	Message    string
	StatusCode int
	ErrorCode  ErrorCode
	Details    map[string]any
	Err        error // Wrapped error
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *APIError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details map[string]any) *APIError {
	e.Details = details
	return e
}

// Wrap wraps an error with additional context
func (e *APIError) Wrap(err error) *APIError {
	e.Err = err
	return e
}

// NewAPIError creates a new API error
func NewAPIError(statusCode int, errorCode ErrorCode, message string) *APIError {
	return &APIError{
		Message:    message,
		StatusCode: statusCode,
		ErrorCode:  errorCode,
		Details:    nil,
		Err:        nil,
	}
}

// Predefined errors for common scenarios

// NewBadRequestError creates a 400 Bad Request error
func NewBadRequestError(message string) *APIError {
	return NewAPIError(http.StatusBadRequest, CodeBadRequest, message)
}

// NewValidationError creates a 400 Validation error
func NewValidationError(message string, details map[string]any) *APIError {
	return NewAPIError(http.StatusBadRequest, CodeValidationFailed, message).WithDetails(details)
}

// NewUnauthorizedError creates a 401 Unauthorized error
func NewUnauthorizedError(message string) *APIError {
	return NewAPIError(http.StatusUnauthorized, CodeUnauthorized, message)
}

// NewForbiddenError creates a 403 Forbidden error
func NewForbiddenError(message string) *APIError {
	return NewAPIError(http.StatusForbidden, CodeForbidden, message)
}

// NewNotFoundError creates a 404 Not Found error
func NewNotFoundError(resource string) *APIError {
	return NewAPIError(http.StatusNotFound, CodeNotFound, fmt.Sprintf("%s not found", resource))
}

// NewConflictError creates a 409 Conflict error
func NewConflictError(message string) *APIError {
	return NewAPIError(http.StatusConflict, CodeConflict, message)
}

// NewInternalError creates a 500 Internal Server Error
func NewInternalError(message string) *APIError {
	return NewAPIError(http.StatusInternalServerError, CodeInternalError, message)
}

// NewDatabaseError creates a 500 Database Error
func NewDatabaseError(err error) *APIError {
	return NewAPIError(http.StatusInternalServerError, CodeDatabaseError, "Database error").Wrap(err)
}

// NewServiceUnavailableError creates a 503 Service Unavailable error
func NewServiceUnavailableError(message string) *APIError {
	return NewAPIError(http.StatusServiceUnavailable, CodeServiceUnavailable, message)
}

// ErrorHandlerConfig holds configuration for error handling
type ErrorHandlerConfig struct {
	// ShowInternalErrors shows detailed error information in responses
	// Should be false in production
	ShowInternalErrors bool

	// LogStackTrace logs stack traces for panics
	LogStackTrace bool

	// OnError is called when an error occurs (for custom logging/metrics)
	OnError func(err error, r *http.Request)
}

// ErrorHandler provides error handling middleware and utilities
type ErrorHandler struct {
	config ErrorHandlerConfig
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config ErrorHandlerConfig) *ErrorHandler {
	return &ErrorHandler{
		config: config,
	}
}

// RecoveryMiddleware catches panics and converts them to 500 errors
func (h *ErrorHandler) RecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := GetRequestID(r)

				// Log the panic
				if h.config.LogStackTrace {
					log.Printf("PANIC [%s]: %v\n%s", requestID, rec, debug.Stack())
				} else {
					log.Printf("PANIC [%s]: %v", requestID, rec)
				}

				// Call custom error handler if set
				if h.config.OnError != nil {
					h.config.OnError(fmt.Errorf("panic: %v", rec), r)
				}

				// Send error response
				message := "Internal server error"
				if h.config.ShowInternalErrors {
					message = fmt.Sprintf("Internal server error: %v", rec)
				}

				h.WriteError(w, r, NewInternalError(message))
			}
		}()

		next(w, r)
	}
}

// WriteError writes an error response
func (h *ErrorHandler) WriteError(w http.ResponseWriter, r *http.Request, err *APIError) {
	requestID := GetRequestID(r)

	response := ErrorResponse{
		Error:     err.Message,
		Code:      err.StatusCode,
		ErrorCode: err.ErrorCode,
		RequestID: requestID,
	}

	// Add details if present
	if err.Details != nil {
		response.Details = err.Details
	}

	// In development, include wrapped error details
	if h.config.ShowInternalErrors && err.Err != nil {
		if response.Details == nil {
			response.Details = make(map[string]any)
		}
		response.Details["internal_error"] = err.Err.Error()
	}

	// Log the error
	if err.StatusCode >= 500 {
		log.Printf("ERROR [%s] %d %s: %s", requestID, err.StatusCode, err.ErrorCode, err.Message)
		if err.Err != nil {
			log.Printf("ERROR [%s] Caused by: %v", requestID, err.Err)
		}
	} else {
		log.Printf("WARN [%s] %d %s: %s", requestID, err.StatusCode, err.ErrorCode, err.Message)
	}

	// Call custom error handler if set
	if h.config.OnError != nil {
		h.config.OnError(err, r)
	}

	// Write response
	w.Header().Set(constants.HeaderContentType, constants.MIMEApplicationJSON)
	w.WriteHeader(err.StatusCode)
	json.NewEncoder(w).Encode(response)
}

// WriteErrorFromError converts a standard error to an API error response
func (h *ErrorHandler) WriteErrorFromError(w http.ResponseWriter, r *http.Request, err error) {
	// Check if it's already an APIError
	if apiErr, ok := err.(*APIError); ok {
		h.WriteError(w, r, apiErr)
		return
	}

	// Default to internal server error
	apiErr := NewInternalError("An unexpected error occurred")
	if h.config.ShowInternalErrors {
		apiErr = NewInternalError(err.Error())
	}
	apiErr.Err = err
	h.WriteError(w, r, apiErr)
}

// RequestIDMiddleware adds a request ID to each request
func RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID already exists in header
		requestID := r.Header.Get(constants.HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to response header
		w.Header().Set(constants.HeaderRequestID, requestID)

		// Add to request context
		ctx := SetRequestID(r.Context(), requestID)
		next(w, r.WithContext(ctx))
	}
}

// Context key for request ID
type contextKey string

const requestIDKey contextKey = constants.ContextKeyRequestID

// SetRequestID sets the request ID in the context
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID gets the request ID from the request context
func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetRequestIDFromContext gets the request ID from a context
func GetRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// MapHTTPStatusToErrorCode maps HTTP status codes to error codes
func MapHTTPStatusToErrorCode(statusCode int) ErrorCode {
	switch statusCode {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusTooManyRequests:
		return CodeTooManyRequests
	case http.StatusInternalServerError:
		return CodeInternalError
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	default:
		return CodeInternalError
	}
}

// MapDatabaseError maps database errors to appropriate API errors
func MapDatabaseError(err error) *APIError {
	// Check for common database error patterns
	errStr := err.Error()

	// Duplicate key / unique constraint
	if containsAny(errStr, constants.DuplicateKeyPatterns) {
		return NewConflictError("Resource already exists")
	}

	// Foreign key constraint
	if contains(errStr, "foreign key", "FOREIGN KEY") {
		return NewBadRequestError("Referenced resource does not exist")
	}

	// Not null constraint
	if contains(errStr, "not null", "NOT NULL") {
		return NewBadRequestError("Required field is missing")
	}

	// Connection error
	if containsAny(errStr, constants.ConnectionErrorPatterns) {
		return NewServiceUnavailableError("Database unavailable")
	}

	// Default to internal error
	return NewDatabaseError(err)
}

// contains checks if any of the substrings are in the string
func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// containsAny checks if any of the patterns in the slice are in the string
func containsAny(s string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}
