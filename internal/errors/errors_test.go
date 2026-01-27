package errors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(http.StatusBadRequest, CodeBadRequest, "Test error")

	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, err.StatusCode)
	}

	if err.ErrorCode != CodeBadRequest {
		t.Errorf("Expected error code %s, got %s", CodeBadRequest, err.ErrorCode)
	}

	if err.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got '%s'", err.Message)
	}
}

func TestAPIError_Error(t *testing.T) {
	t.Run("Without wrapped error", func(t *testing.T) {
		err := NewAPIError(http.StatusBadRequest, CodeBadRequest, "Test error")
		if err.Error() != "Test error" {
			t.Errorf("Expected 'Test error', got '%s'", err.Error())
		}
	})

	t.Run("With wrapped error", func(t *testing.T) {
		wrapped := errors.New("underlying error")
		err := NewAPIError(http.StatusBadRequest, CodeBadRequest, "Test error").Wrap(wrapped)

		expected := "Test error: underlying error"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestAPIError_Unwrap(t *testing.T) {
	wrapped := errors.New("underlying error")
	err := NewAPIError(http.StatusBadRequest, CodeBadRequest, "Test error").Wrap(wrapped)

	unwrapped := err.Unwrap()
	if unwrapped != wrapped {
		t.Error("Unwrap should return the wrapped error")
	}
}

func TestAPIError_WithDetails(t *testing.T) {
	details := map[string]any{
		"field": "email",
		"value": "invalid",
	}

	err := NewAPIError(http.StatusBadRequest, CodeBadRequest, "Test error").WithDetails(details)

	if err.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if err.Details["field"] != "email" {
		t.Errorf("Expected field 'email', got '%v'", err.Details["field"])
	}
}

func TestPredefinedErrors(t *testing.T) {
	testCases := []struct {
		name           string
		errFunc        func() *APIError
		expectedStatus int
		expectedCode   ErrorCode
	}{
		{
			name:           "BadRequest",
			errFunc:        func() *APIError { return NewBadRequestError("bad request") },
			expectedStatus: http.StatusBadRequest,
			expectedCode:   CodeBadRequest,
		},
		{
			name:           "Unauthorized",
			errFunc:        func() *APIError { return NewUnauthorizedError("unauthorized") },
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   CodeUnauthorized,
		},
		{
			name:           "Forbidden",
			errFunc:        func() *APIError { return NewForbiddenError("forbidden") },
			expectedStatus: http.StatusForbidden,
			expectedCode:   CodeForbidden,
		},
		{
			name:           "NotFound",
			errFunc:        func() *APIError { return NewNotFoundError("resource") },
			expectedStatus: http.StatusNotFound,
			expectedCode:   CodeNotFound,
		},
		{
			name:           "Conflict",
			errFunc:        func() *APIError { return NewConflictError("conflict") },
			expectedStatus: http.StatusConflict,
			expectedCode:   CodeConflict,
		},
		{
			name:           "InternalError",
			errFunc:        func() *APIError { return NewInternalError("internal") },
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   CodeInternalError,
		},
		{
			name:           "ServiceUnavailable",
			errFunc:        func() *APIError { return NewServiceUnavailableError("unavailable") },
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   CodeServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.errFunc()

			if err.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, err.StatusCode)
			}

			if err.ErrorCode != tc.expectedCode {
				t.Errorf("Expected code %s, got %s", tc.expectedCode, err.ErrorCode)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	details := map[string]any{
		"field":   "email",
		"message": "invalid format",
	}

	err := NewValidationError("Validation failed", details)

	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, err.StatusCode)
	}

	if err.ErrorCode != CodeValidationFailed {
		t.Errorf("Expected code %s, got %s", CodeValidationFailed, err.ErrorCode)
	}

	if err.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if err.Details["field"] != "email" {
		t.Errorf("Expected field 'email', got '%v'", err.Details["field"])
	}
}

func TestNewDatabaseError(t *testing.T) {
	wrapped := errors.New("connection failed")
	err := NewDatabaseError(wrapped)

	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, err.StatusCode)
	}

	if err.ErrorCode != CodeDatabaseError {
		t.Errorf("Expected code %s, got %s", CodeDatabaseError, err.ErrorCode)
	}

	if err.Err != wrapped {
		t.Error("Wrapped error should be set")
	}
}

func TestNewErrorHandler(t *testing.T) {
	config := ErrorHandlerConfig{
		ShowInternalErrors: true,
		LogStackTrace:      true,
	}

	handler := NewErrorHandler(config)

	if handler == nil {
		t.Fatal("NewErrorHandler returned nil")
	}

	if !handler.config.ShowInternalErrors {
		t.Error("ShowInternalErrors should be true")
	}
}

func TestErrorHandler_WriteError(t *testing.T) {
	handler := NewErrorHandler(ErrorHandlerConfig{})

	err := NewBadRequestError("Test error")

	// Create request with request ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.WriteError(w, req, err)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error != "Test error" {
		t.Errorf("Expected error 'Test error', got '%s'", response.Error)
	}

	if response.Code != http.StatusBadRequest {
		t.Errorf("Expected code %d, got %d", http.StatusBadRequest, response.Code)
	}

	if response.RequestID != "test-request-id" {
		t.Errorf("Expected request ID 'test-request-id', got '%s'", response.RequestID)
	}
}

func TestErrorHandler_WriteError_WithDetails(t *testing.T) {
	handler := NewErrorHandler(ErrorHandlerConfig{})

	details := map[string]any{"field": "email"}
	err := NewValidationError("Validation failed", details)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.WriteError(w, req, err)

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if response.Details["field"] != "email" {
		t.Errorf("Expected detail field 'email', got '%v'", response.Details["field"])
	}
}

func TestErrorHandler_WriteError_ShowInternalErrors(t *testing.T) {
	handler := NewErrorHandler(ErrorHandlerConfig{
		ShowInternalErrors: true,
	})

	wrapped := errors.New("internal details")
	err := NewInternalError("Error occurred").Wrap(wrapped)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.WriteError(w, req, err)

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Details == nil {
		t.Fatal("Details should contain internal error when ShowInternalErrors is true")
	}

	if response.Details["internal_error"] != "internal details" {
		t.Errorf("Expected internal_error 'internal details', got '%v'", response.Details["internal_error"])
	}
}

func TestErrorHandler_WriteError_HideInternalErrors(t *testing.T) {
	handler := NewErrorHandler(ErrorHandlerConfig{
		ShowInternalErrors: false,
	})

	wrapped := errors.New("internal details")
	err := NewInternalError("Error occurred").Wrap(wrapped)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.WriteError(w, req, err)

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	// Details should be nil or not contain internal_error
	if response.Details != nil && response.Details["internal_error"] != nil {
		t.Error("Internal error should be hidden when ShowInternalErrors is false")
	}
}

func TestErrorHandler_RecoveryMiddleware(t *testing.T) {
	handler := NewErrorHandler(ErrorHandlerConfig{
		ShowInternalErrors: false,
		LogStackTrace:      false,
	})

	panicHandler := func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}

	wrapped := handler.RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	wrapped(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error != "Internal server error" {
		t.Errorf("Expected error 'Internal server error', got '%s'", response.Error)
	}
}

func TestErrorHandler_RecoveryMiddleware_NoPanic(t *testing.T) {
	handler := NewErrorHandler(ErrorHandlerConfig{})

	normalHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	wrapped := handler.RecoveryMiddleware(normalHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestErrorHandler_WriteErrorFromError(t *testing.T) {
	handler := NewErrorHandler(ErrorHandlerConfig{})

	t.Run("APIError", func(t *testing.T) {
		apiErr := NewBadRequestError("Bad request")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.WriteErrorFromError(w, req, apiErr)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Standard error", func(t *testing.T) {
		stdErr := errors.New("standard error")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.WriteErrorFromError(w, req, stdErr)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r)
		w.Write([]byte(requestID))
	}

	wrapped := RequestIDMiddleware(handler)

	t.Run("Generates request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrapped(w, req)

		// Check response header
		headerID := w.Header().Get("X-Request-ID")
		if headerID == "" {
			t.Error("X-Request-ID header should be set")
		}

		// Check that ID was passed to handler
		bodyID := w.Body.String()
		if bodyID != headerID {
			t.Errorf("Request ID in context '%s' doesn't match header '%s'", bodyID, headerID)
		}
	})

	t.Run("Uses existing request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "existing-id")
		w := httptest.NewRecorder()

		wrapped(w, req)

		headerID := w.Header().Get("X-Request-ID")
		if headerID != "existing-id" {
			t.Errorf("Expected 'existing-id', got '%s'", headerID)
		}
	})
}

func TestSetRequestID_GetRequestIDFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = SetRequestID(ctx, "test-id")

	id := GetRequestIDFromContext(ctx)
	if id != "test-id" {
		t.Errorf("Expected 'test-id', got '%s'", id)
	}
}

func TestGetRequestIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	id := GetRequestIDFromContext(ctx)
	if id != "" {
		t.Errorf("Expected empty string, got '%s'", id)
	}
}

func TestMapHTTPStatusToErrorCode(t *testing.T) {
	testCases := []struct {
		status   int
		expected ErrorCode
	}{
		{http.StatusBadRequest, CodeBadRequest},
		{http.StatusUnauthorized, CodeUnauthorized},
		{http.StatusForbidden, CodeForbidden},
		{http.StatusNotFound, CodeNotFound},
		{http.StatusConflict, CodeConflict},
		{http.StatusTooManyRequests, CodeTooManyRequests},
		{http.StatusInternalServerError, CodeInternalError},
		{http.StatusServiceUnavailable, CodeServiceUnavailable},
		{999, CodeInternalError}, // Unknown status
	}

	for _, tc := range testCases {
		t.Run(string(tc.expected), func(t *testing.T) {
			code := MapHTTPStatusToErrorCode(tc.status)
			if code != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, code)
			}
		})
	}
}

func TestMapDatabaseError(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   ErrorCode
	}{
		{
			name:           "Duplicate key",
			err:            errors.New("UNIQUE constraint failed: users.email"),
			expectedStatus: http.StatusConflict,
			expectedCode:   CodeConflict,
		},
		{
			name:           "Foreign key",
			err:            errors.New("FOREIGN KEY constraint failed"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   CodeBadRequest,
		},
		{
			name:           "Not null",
			err:            errors.New("NOT NULL constraint failed"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   CodeBadRequest,
		},
		{
			name:           "Connection refused",
			err:            errors.New("connection refused"),
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   CodeServiceUnavailable,
		},
		{
			name:           "Unknown error",
			err:            errors.New("unknown database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   CodeDatabaseError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := MapDatabaseError(tc.err)

			if apiErr.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, apiErr.StatusCode)
			}

			if apiErr.ErrorCode != tc.expectedCode {
				t.Errorf("Expected code %s, got %s", tc.expectedCode, apiErr.ErrorCode)
			}
		})
	}
}

func TestContains(t *testing.T) {
	testCases := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"hello world", []string{"hello"}, true},
		{"hello world", []string{"world"}, true},
		{"hello world", []string{"foo"}, false},
		{"hello world", []string{"foo", "hello"}, true},
		{"", []string{"foo"}, false},
		{"foo", []string{""}, true},
	}

	for _, tc := range testCases {
		result := contains(tc.s, tc.substrs...)
		if result != tc.expected {
			t.Errorf("contains(%q, %v) = %v, expected %v", tc.s, tc.substrs, result, tc.expected)
		}
	}
}

func TestErrorHandler_OnError(t *testing.T) {
	errorCalled := false
	var capturedErr error

	handler := NewErrorHandler(ErrorHandlerConfig{
		OnError: func(err error, r *http.Request) {
			errorCalled = true
			capturedErr = err
		},
	})

	apiErr := NewBadRequestError("Test error")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.WriteError(w, req, apiErr)

	if !errorCalled {
		t.Error("OnError callback should have been called")
	}

	if capturedErr == nil {
		t.Error("Captured error should not be nil")
	}
}
