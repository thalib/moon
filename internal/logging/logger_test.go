package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		logger := NewLogger(LoggerConfig{})

		if logger == nil {
			t.Fatal("NewLogger returned nil")
		}

		if logger.config.Level != LevelInfo {
			t.Errorf("Expected default level info, got %s", logger.config.Level)
		}
	})

	t.Run("Custom config", func(t *testing.T) {
		logger := NewLogger(LoggerConfig{
			Level:       LevelDebug,
			ServiceName: "test-service",
			Version:     "1.0.0",
		})

		if logger.config.Level != LevelDebug {
			t.Errorf("Expected level debug, got %s", logger.config.Level)
		}
	})
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	logger.Info("Test message")

	// Parse JSON output
	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v\nOutput: %s", err, buf.String())
	}

	if logEntry["message"] != "Test message" {
		t.Errorf("Expected message 'Test message', got '%v'", logEntry["message"])
	}

	if logEntry["level"] != "info" {
		t.Errorf("Expected level 'info', got '%v'", logEntry["level"])
	}
}

func TestLogger_ConsoleFormat(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "console",
		Output: &buf,
	})

	logger.Info("Test message")

	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("Expected output to contain 'Test message', got '%s'", output)
	}
}

func TestLogger_Levels(t *testing.T) {
	testCases := []struct {
		level         Level
		logFunc       func(*Logger)
		expectedLevel string
	}{
		{
			level:         LevelDebug,
			logFunc:       func(l *Logger) { l.Debug("debug msg") },
			expectedLevel: "debug",
		},
		{
			level:         LevelInfo,
			logFunc:       func(l *Logger) { l.Info("info msg") },
			expectedLevel: "info",
		},
		{
			level:         LevelWarn,
			logFunc:       func(l *Logger) { l.Warn("warn msg") },
			expectedLevel: "warn",
		},
		{
			level:         LevelError,
			logFunc:       func(l *Logger) { l.Error("error msg") },
			expectedLevel: "error",
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.level), func(t *testing.T) {
			var buf bytes.Buffer

			logger := NewLogger(LoggerConfig{
				Level:  LevelDebug, // Allow all levels
				Format: "json",
				Output: &buf,
			})

			tc.logFunc(logger)

			var logEntry map[string]any
			json.Unmarshal(buf.Bytes(), &logEntry)

			if logEntry["level"] != tc.expectedLevel {
				t.Errorf("Expected level '%s', got '%v'", tc.expectedLevel, logEntry["level"])
			}
		})
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelWarn, // Only warn and above
		Format: "json",
		Output: &buf,
	})

	logger.Debug("debug msg")
	logger.Info("info msg")

	// Buffer should be empty since debug and info are below warn
	if buf.Len() > 0 {
		t.Errorf("Expected no output for debug/info when level is warn, got: %s", buf.String())
	}

	logger.Warn("warn msg")

	if buf.Len() == 0 {
		t.Error("Expected output for warn level")
	}
}

func TestLogger_WithField(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	logger.WithField("user_id", "123").Info("User action")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["user_id"] != "123" {
		t.Errorf("Expected user_id '123', got '%v'", logEntry["user_id"])
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	fields := map[string]any{
		"user_id":   "123",
		"action":    "login",
		"ip":        "192.168.1.1",
	}

	logger.WithFields(fields).Info("User action")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	for key, expected := range fields {
		if logEntry[key] != expected {
			t.Errorf("Expected %s '%v', got '%v'", key, expected, logEntry[key])
		}
	}
}

func TestLogger_SensitiveFieldMasking(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	logger.WithField("password", "secret123").Info("Login attempt")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["password"] == "secret123" {
		t.Error("Password should be masked")
	}

	if logEntry["password"] != "***REDACTED***" {
		t.Errorf("Expected masked password, got '%v'", logEntry["password"])
	}
}

func TestLogger_CustomSensitiveFields(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:           LevelInfo,
		Format:          "json",
		Output:          &buf,
		SensitiveFields: []string{"credit_card"},
	})

	logger.WithField("credit_card", "1234-5678").Info("Payment")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["credit_card"] != "***REDACTED***" {
		t.Errorf("Expected credit_card to be masked, got '%v'", logEntry["credit_card"])
	}
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	ctx := SetRequestID(context.Background(), "req-123")
	logger.WithContext(ctx).Info("Request processed")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["request_id"] != "req-123" {
		t.Errorf("Expected request_id 'req-123', got '%v'", logEntry["request_id"])
	}
}

func TestLogger_ServiceContext(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:       LevelInfo,
		Format:      "json",
		Output:      &buf,
		ServiceName: "api-server",
		Version:     "2.0.0",
	})

	logger.Info("Service started")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["service"] != "api-server" {
		t.Errorf("Expected service 'api-server', got '%v'", logEntry["service"])
	}

	if logEntry["version"] != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%v'", logEntry["version"])
	}
}

func TestLogger_ErrorWithErr(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelError,
		Format: "json",
		Output: &buf,
	})

	testErr := &testError{msg: "connection failed"}
	logger.ErrorWithErr("Database error", testErr)

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["error"] != "connection failed" {
		t.Errorf("Expected error 'connection failed', got '%v'", logEntry["error"])
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestLogger_LogSlowQuery(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:              LevelWarn,
		Format:             "json",
		Output:             &buf,
		SlowQueryThreshold: 100 * time.Millisecond,
	})

	// Fast query - should not log
	logger.LogSlowQuery("SELECT * FROM users", 50*time.Millisecond)
	if buf.Len() > 0 {
		t.Error("Fast query should not be logged")
	}

	// Slow query - should log
	logger.LogSlowQuery("SELECT * FROM big_table", 200*time.Millisecond)

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["query"] != "SELECT * FROM big_table" {
		t.Errorf("Expected query to be logged")
	}
}

func TestSetRequestID_GetRequestID(t *testing.T) {
	ctx := context.Background()
	ctx = SetRequestID(ctx, "test-id-123")

	id := GetRequestID(ctx)
	if id != "test-id-123" {
		t.Errorf("Expected 'test-id-123', got '%s'", id)
	}
}

func TestGetRequestID_Empty(t *testing.T) {
	ctx := context.Background()
	id := GetRequestID(ctx)
	if id != "" {
		t.Errorf("Expected empty string, got '%s'", id)
	}
}

func TestNewRequestLogger(t *testing.T) {
	logger := NewLogger(LoggerConfig{Level: LevelInfo})
	rl := NewRequestLogger(RequestLoggerConfig{
		Logger:    logger,
		SkipPaths: []string{"/health"},
	})

	if rl == nil {
		t.Fatal("NewRequestLogger returned nil")
	}

	if !rl.skipPaths["/health"] {
		t.Error("Expected /health to be in skip paths")
	}
}

func TestRequestLogger_Middleware(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	rl := NewRequestLogger(RequestLoggerConfig{
		Logger: logger,
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	wrapped := rl.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check request ID header was set
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}

	// Check log output
	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["method"] != "GET" {
		t.Errorf("Expected method 'GET', got '%v'", logEntry["method"])
	}

	if logEntry["path"] != "/api/users" {
		t.Errorf("Expected path '/api/users', got '%v'", logEntry["path"])
	}

	if logEntry["status"] != float64(200) {
		t.Errorf("Expected status 200, got '%v'", logEntry["status"])
	}
}

func TestRequestLogger_Middleware_SkipPaths(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	rl := NewRequestLogger(RequestLoggerConfig{
		Logger:    logger,
		SkipPaths: []string{"/health"},
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrapped := rl.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	// Should not log anything for skipped path
	if buf.Len() > 0 {
		t.Errorf("Expected no log output for skipped path, got: %s", buf.String())
	}
}

func TestRequestLogger_Middleware_ExistingRequestID(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	rl := NewRequestLogger(RequestLoggerConfig{
		Logger: logger,
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrapped := rl.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	w := httptest.NewRecorder()

	wrapped(w, req)

	// Check that existing request ID was used
	if w.Header().Get("X-Request-ID") != "existing-id" {
		t.Errorf("Expected 'existing-id', got '%s'", w.Header().Get("X-Request-ID"))
	}

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["request_id"] != "existing-id" {
		t.Errorf("Expected request_id 'existing-id', got '%v'", logEntry["request_id"])
	}
}

func TestRequestLogger_Middleware_ErrorStatus(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(LoggerConfig{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	rl := NewRequestLogger(RequestLoggerConfig{
		Logger: logger,
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	wrapped := rl.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	// Error level should be used for 5xx status
	if logEntry["level"] != "error" {
		t.Errorf("Expected level 'error' for 500 status, got '%v'", logEntry["level"])
	}
}

func TestGlobalLogger(t *testing.T) {
	// Reset global logger
	globalLogger = nil

	// Test that GetLogger creates a default logger
	logger := GetLogger()
	if logger == nil {
		t.Fatal("GetLogger returned nil")
	}

	// Test Init
	var buf bytes.Buffer
	Init(LoggerConfig{
		Level:  LevelDebug,
		Format: "json",
		Output: &buf,
	})

	// Use global functions
	Info("Global info message")

	var logEntry map[string]any
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["message"] != "Global info message" {
		t.Errorf("Expected message 'Global info message', got '%v'", logEntry["message"])
	}
}

func TestGlobalLogFunctions(t *testing.T) {
	var buf bytes.Buffer

	Init(LoggerConfig{
		Level:  LevelDebug,
		Format: "json",
		Output: &buf,
	})

	// Test all global log functions
	Debug("debug")
	buf.Reset()

	Debugf("debug %s", "formatted")
	buf.Reset()

	Info("info")
	var entry1 map[string]any
	json.Unmarshal(buf.Bytes(), &entry1)
	if entry1["level"] != "info" {
		t.Errorf("Expected level 'info', got '%v'", entry1["level"])
	}
	buf.Reset()

	Infof("info %s", "formatted")
	buf.Reset()

	Warn("warn")
	var entry2 map[string]any
	json.Unmarshal(buf.Bytes(), &entry2)
	if entry2["level"] != "warn" {
		t.Errorf("Expected level 'warn', got '%v'", entry2["level"])
	}
	buf.Reset()

	Warnf("warn %s", "formatted")
	buf.Reset()

	Error("error")
	var entry3 map[string]any
	json.Unmarshal(buf.Bytes(), &entry3)
	if entry3["level"] != "error" {
		t.Errorf("Expected level 'error', got '%v'", entry3["level"])
	}
	buf.Reset()

	Errorf("error %s", "formatted")
	buf.Reset()

	testErr := &testError{msg: "test error"}
	ErrorWithErr("with error", testErr)
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		bytesWritten:   0,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rw.statusCode)
	}

	// Test Write
	data := []byte("test data")
	n, err := rw.Write(data)
	if err != nil {
		t.Errorf("Write returned error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}
	if rw.bytesWritten != len(data) {
		t.Errorf("Expected bytesWritten %d, got %d", len(data), rw.bytesWritten)
	}
}
