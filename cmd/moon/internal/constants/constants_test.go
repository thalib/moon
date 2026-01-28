package constants

import (
	"os"
	"testing"
	"time"
)

// TestHeaderConstants verifies that HTTP header constants are defined.
func TestHeaderConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"RequestID header", HeaderRequestID, "X-Request-ID"},
		{"Authorization header", HeaderAuthorization, "Authorization"},
		{"API Key header", HeaderAPIKey, "X-API-Key"},
		{"Content-Type header", HeaderContentType, "Content-Type"},
		{"JSON MIME type", MIMEApplicationJSON, "application/json"},
		{"Bearer scheme", AuthSchemeBearer, "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

// TestPermissionConstants verifies that file permission constants are defined correctly.
func TestPermissionConstants(t *testing.T) {
	if DirPermissions != os.FileMode(0755) {
		t.Errorf("Expected DirPermissions to be 0755, got %o", DirPermissions)
	}

	if FilePermissions != os.FileMode(0644) {
		t.Errorf("Expected FilePermissions to be 0644, got %o", FilePermissions)
	}
}

// TestTimeoutConstants verifies that timeout constants are defined with expected values.
func TestTimeoutConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant time.Duration
		expected time.Duration
	}{
		{"Shutdown timeout", ShutdownTimeout, 30 * time.Second},
		{"HTTP read timeout", HTTPReadTimeout, 15 * time.Second},
		{"HTTP write timeout", HTTPWriteTimeout, 15 * time.Second},
		{"HTTP idle timeout", HTTPIdleTimeout, 60 * time.Second},
		{"Health check timeout", HealthCheckTimeout, 5 * time.Second},
		{"JWT clock skew", JWTClockSkew, 30 * time.Second},
		{"Slow query threshold", SlowQueryThreshold, 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be %v, got %v", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

// TestSensitiveFields verifies that sensitive fields list is defined.
func TestSensitiveFields(t *testing.T) {
	expectedFields := []string{
		"password",
		"token",
		"secret",
		"api_key",
		"apikey",
		"authorization",
	}

	if len(SensitiveFields) != len(expectedFields) {
		t.Errorf("Expected %d sensitive fields, got %d", len(expectedFields), len(SensitiveFields))
	}

	// Check that all expected fields are present
	fieldMap := make(map[string]bool)
	for _, field := range SensitiveFields {
		fieldMap[field] = true
	}

	for _, expected := range expectedFields {
		if !fieldMap[expected] {
			t.Errorf("Expected sensitive field %q not found", expected)
		}
	}

	if RedactedPlaceholder != "***REDACTED***" {
		t.Errorf("Expected RedactedPlaceholder to be %q, got %q", "***REDACTED***", RedactedPlaceholder)
	}
}

// TestPaginationConstants verifies pagination constant values.
func TestPaginationConstants(t *testing.T) {
	if DefaultPaginationLimit != 100 {
		t.Errorf("Expected DefaultPaginationLimit to be 100, got %d", DefaultPaginationLimit)
	}

	if DefaultPaginationOffset != 0 {
		t.Errorf("Expected DefaultPaginationOffset to be 0, got %d", DefaultPaginationOffset)
	}

	if QueryParamLimit != "limit" {
		t.Errorf("Expected QueryParamLimit to be %q, got %q", "limit", QueryParamLimit)
	}

	if QueryParamOffset != "offset" {
		t.Errorf("Expected QueryParamOffset to be %q, got %q", "offset", QueryParamOffset)
	}

	if QueryParamID != "id" {
		t.Errorf("Expected QueryParamID to be %q, got %q", "id", QueryParamID)
	}
}

// TestValidationConstants verifies validation constant values.
func TestValidationConstants(t *testing.T) {
	if DefaultVarcharMaxLength != 255 {
		t.Errorf("Expected DefaultVarcharMaxLength to be 255, got %d", DefaultVarcharMaxLength)
	}

	if MinAPIKeyLength != 40 {
		t.Errorf("Expected MinAPIKeyLength to be 40, got %d", MinAPIKeyLength)
	}

	if CollectionNamePattern != `^[a-zA-Z][a-zA-Z0-9_]*$` {
		t.Errorf("Expected CollectionNamePattern to be %q, got %q", `^[a-zA-Z][a-zA-Z0-9_]*$`, CollectionNamePattern)
	}
}

// TestErrorPatterns verifies error pattern constants.
func TestErrorPatterns(t *testing.T) {
	if len(DuplicateKeyPatterns) == 0 {
		t.Error("Expected DuplicateKeyPatterns to be non-empty")
	}

	if len(ConnectionErrorPatterns) == 0 {
		t.Error("Expected ConnectionErrorPatterns to be non-empty")
	}

	// Verify specific patterns exist
	expectedDuplicatePatterns := []string{"duplicate", "unique constraint", "UNIQUE constraint"}
	for _, pattern := range expectedDuplicatePatterns {
		found := false
		for _, p := range DuplicateKeyPatterns {
			if p == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected duplicate pattern %q not found", pattern)
		}
	}

	expectedConnectionPatterns := []string{"connection refused", "no such host", "timeout"}
	for _, pattern := range expectedConnectionPatterns {
		found := false
		for _, p := range ConnectionErrorPatterns {
			if p == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected connection pattern %q not found", pattern)
		}
	}
}

// TestContextKeys verifies context key constants.
func TestContextKeys(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"Request ID context key", ContextKeyRequestID, "request_id"},
		{"User claims context key", ContextKeyUserClaims, "user_claims"},
		{"API key info context key", ContextKeyAPIKeyInfo, "api_key_info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

// TestPathConstants verifies default path constants.
func TestPathConstants(t *testing.T) {
	if DefaultPIDFile != "/var/run/moon.pid" {
		t.Errorf("Expected DefaultPIDFile to be %q, got %q", "/var/run/moon.pid", DefaultPIDFile)
	}

	if DefaultWorkingDirectory != "/" {
		t.Errorf("Expected DefaultWorkingDirectory to be %q, got %q", "/", DefaultWorkingDirectory)
	}
}
