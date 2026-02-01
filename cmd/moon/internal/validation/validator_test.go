package validation

import (
	"regexp"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/registry"
)

func setupTestRegistry() *registry.SchemaRegistry {
	reg := registry.NewSchemaRegistry()

	// Add a test collection (updated for new type system - no text/float types)
	collection := &registry.Collection{
		Name: "users",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "email", Type: registry.TypeString, Nullable: false},
			{Name: "age", Type: registry.TypeInteger, Nullable: true},
			{Name: "score", Type: registry.TypeInteger, Nullable: true},
			{Name: "active", Type: registry.TypeBoolean, Nullable: true},
			{Name: "created_at", Type: registry.TypeDatetime, Nullable: true},
			{Name: "metadata", Type: registry.TypeJSON, Nullable: true},
			{Name: "bio", Type: registry.TypeString, Nullable: true},
		},
	}
	reg.Set(collection)

	return reg
}

func TestNewSchemaValidator(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	if validator == nil {
		t.Fatal("NewSchemaValidator returned nil")
	}

	if validator.registry != reg {
		t.Error("Validator registry not set correctly")
	}

	if validator.mode != StrictMode {
		t.Error("Default mode should be StrictMode")
	}
}

func TestSchemaValidator_SetMode(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	validator.SetMode(PermissiveMode)

	if validator.mode != PermissiveMode {
		t.Error("Mode was not set to PermissiveMode")
	}
}

func TestSchemaValidator_Validate_CollectionNotFound(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	errors := validator.Validate("nonexistent", map[string]any{"name": "test"})

	if errors == nil {
		t.Fatal("Expected validation errors for nonexistent collection")
	}

	if len(errors.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors.Errors))
	}

	if errors.Errors[0].Code != "COLLECTION_NOT_FOUND" {
		t.Errorf("Expected code COLLECTION_NOT_FOUND, got %s", errors.Errors[0].Code)
	}
}

func TestSchemaValidator_Validate_RequiredFields(t *testing.T) {
	reg := setupTestRegistry()
	validator := NewSchemaValidator(reg)

	// Missing required fields
	errors := validator.Validate("users", map[string]any{
		"age": 25,
	})

	if errors == nil {
		t.Fatal("Expected validation errors for missing required fields")
	}

	// Should have errors for name and email
	if len(errors.Errors) != 2 {
		t.Fatalf("Expected 2 errors, got %d", len(errors.Errors))
	}

	// Check that errors are for required fields
	errorFields := make(map[string]bool)
	for _, err := range errors.Errors {
		errorFields[err.Field] = true
		if err.Code != "REQUIRED_FIELD" {
			t.Errorf("Expected code REQUIRED_FIELD, got %s", err.Code)
		}
	}

	if !errorFields["name"] || !errorFields["email"] {
		t.Error("Expected errors for name and email fields")
	}
}

func TestSchemaValidator_Validate_ValidData(t *testing.T) {
	reg := setupTestRegistry()
	validator := NewSchemaValidator(reg)

	data := map[string]any{
		"name":       "John Doe",
		"email":      "john@example.com",
		"age":        30,
		"score":      100,
		"active":     true,
		"created_at": "2024-01-15T10:30:00Z",
		"metadata":   map[string]any{"key": "value"},
		"bio":        "A test user",
	}

	errors := validator.Validate("users", data)

	if errors != nil {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}
}

func TestSchemaValidator_Validate_StrictMode_UnknownField(t *testing.T) {
	reg := setupTestRegistry()
	validator := NewSchemaValidator(reg)
	validator.SetMode(StrictMode)

	data := map[string]any{
		"name":          "John Doe",
		"email":         "john@example.com",
		"unknown_field": "value",
	}

	errors := validator.Validate("users", data)

	if errors == nil {
		t.Fatal("Expected validation errors for unknown field in strict mode")
	}

	found := false
	for _, err := range errors.Errors {
		if err.Code == "UNKNOWN_FIELD" && err.Field == "unknown_field" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected UNKNOWN_FIELD error for unknown_field")
	}
}

func TestSchemaValidator_Validate_PermissiveMode_UnknownField(t *testing.T) {
	reg := setupTestRegistry()
	validator := NewSchemaValidator(reg)
	validator.SetMode(PermissiveMode)

	data := map[string]any{
		"name":          "John Doe",
		"email":         "john@example.com",
		"unknown_field": "value",
	}

	errors := validator.Validate("users", data)

	if errors != nil {
		t.Errorf("Expected no validation errors in permissive mode, got: %v", errors)
	}
}

func TestSchemaValidator_ValidateField_String(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "name", Type: registry.TypeString}

	t.Run("Valid string", func(t *testing.T) {
		err := validator.ValidateField("name", "John", column)
		if err != nil {
			t.Errorf("Expected no error for valid string, got: %v", err)
		}
	})

	t.Run("Invalid type", func(t *testing.T) {
		err := validator.ValidateField("name", 123, column)
		if err == nil {
			t.Error("Expected error for non-string value")
		}
		if err.Code != "INVALID_TYPE" {
			t.Errorf("Expected code INVALID_TYPE, got %s", err.Code)
		}
	})
}

func TestSchemaValidator_ValidateField_Integer(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "age", Type: registry.TypeInteger}

	t.Run("Valid integer", func(t *testing.T) {
		err := validator.ValidateField("age", 25, column)
		if err != nil {
			t.Errorf("Expected no error for valid integer, got: %v", err)
		}
	})

	t.Run("Valid float64 whole number", func(t *testing.T) {
		err := validator.ValidateField("age", float64(25), column)
		if err != nil {
			t.Errorf("Expected no error for whole number float64, got: %v", err)
		}
	})

	t.Run("Invalid float64 decimal", func(t *testing.T) {
		err := validator.ValidateField("age", 25.5, column)
		if err == nil {
			t.Error("Expected error for decimal float64")
		}
	})

	t.Run("Invalid type string", func(t *testing.T) {
		err := validator.ValidateField("age", "25", column)
		if err == nil {
			t.Error("Expected error for string value")
		}
	})
}

func TestSchemaValidator_ValidateField_Score(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "score", Type: registry.TypeInteger}

	t.Run("Valid integer", func(t *testing.T) {
		err := validator.ValidateField("score", 100, column)
		if err != nil {
			t.Errorf("Expected no error for valid integer, got: %v", err)
		}
	})

	t.Run("Valid float64 whole number", func(t *testing.T) {
		err := validator.ValidateField("score", float64(100), column)
		if err != nil {
			t.Errorf("Expected no error for whole number float64, got: %v", err)
		}
	})

	t.Run("Invalid type string", func(t *testing.T) {
		err := validator.ValidateField("score", "100", column)
		if err == nil {
			t.Error("Expected error for string value")
		}
	})
}

func TestSchemaValidator_ValidateField_Boolean(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "active", Type: registry.TypeBoolean}

	t.Run("Valid boolean true", func(t *testing.T) {
		err := validator.ValidateField("active", true, column)
		if err != nil {
			t.Errorf("Expected no error for true, got: %v", err)
		}
	})

	t.Run("Valid boolean false", func(t *testing.T) {
		err := validator.ValidateField("active", false, column)
		if err != nil {
			t.Errorf("Expected no error for false, got: %v", err)
		}
	})

	t.Run("Invalid type string", func(t *testing.T) {
		err := validator.ValidateField("active", "true", column)
		if err == nil {
			t.Error("Expected error for string value")
		}
	})

	t.Run("Invalid type integer", func(t *testing.T) {
		err := validator.ValidateField("active", 1, column)
		if err == nil {
			t.Error("Expected error for integer value")
		}
	})
}

func TestSchemaValidator_ValidateField_Datetime(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "created_at", Type: registry.TypeDatetime}

	validFormats := []string{
		"2024-01-15T10:30:00Z",
		"2024-01-15T10:30:00",
		"2024-01-15 10:30:00",
		"2024-01-15",
	}

	for _, format := range validFormats {
		t.Run("Valid datetime "+format, func(t *testing.T) {
			err := validator.ValidateField("created_at", format, column)
			if err != nil {
				t.Errorf("Expected no error for %s, got: %v", format, err)
			}
		})
	}

	t.Run("Invalid datetime format", func(t *testing.T) {
		err := validator.ValidateField("created_at", "01/15/2024", column)
		if err == nil {
			t.Error("Expected error for invalid datetime format")
		}
		if err.Code != "INVALID_DATETIME" {
			t.Errorf("Expected code INVALID_DATETIME, got %s", err.Code)
		}
	})
}

func TestSchemaValidator_ValidateField_JSON(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "metadata", Type: registry.TypeJSON}

	testCases := []struct {
		name  string
		value any
	}{
		{"Map", map[string]any{"key": "value"}},
		{"Array", []any{1, 2, 3}},
		{"String", "string value"},
		{"Number", 123},
		{"Boolean", true},
		{"Nil", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateField("metadata", tc.value, column)
			if err != nil {
				t.Errorf("Expected no error for JSON value %v, got: %v", tc.value, err)
			}
		})
	}
}

func TestSchemaValidator_ValidateField_Bio(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	// Bio is now string type, not text
	column := registry.Column{Name: "bio", Type: registry.TypeString}

	t.Run("Valid string", func(t *testing.T) {
		err := validator.ValidateField("bio", "A long text value", column)
		if err != nil {
			t.Errorf("Expected no error for valid string, got: %v", err)
		}
	})

	t.Run("Invalid type", func(t *testing.T) {
		err := validator.ValidateField("bio", 123, column)
		if err == nil {
			t.Error("Expected error for non-string value")
		}
	})
}

func TestSchemaValidator_StringConstraints(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "name", Type: registry.TypeString}

	t.Run("String normal length", func(t *testing.T) {
		err := validator.ValidateField("name", "John Doe", column)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("String long length allowed", func(t *testing.T) {
		// Since string now maps to TEXT, there is no length limit
		longString := make([]byte, 300)
		for i := range longString {
			longString[i] = 'a'
		}

		err := validator.ValidateField("name", string(longString), column)
		// No error expected - TEXT type has no length constraint
		if err != nil {
			t.Errorf("Expected no error for long string (TEXT type), got: %v", err)
		}
	})
}

func TestSchemaValidator_CustomRules(t *testing.T) {
	reg := setupTestRegistry()
	validator := NewSchemaValidator(reg)

	// Add custom email validation
	validator.AddCustomRule("users", "email", EmailRule())

	t.Run("Valid email", func(t *testing.T) {
		errors := validator.Validate("users", map[string]any{
			"name":  "John",
			"email": "john@example.com",
		})

		if errors != nil {
			t.Errorf("Expected no errors for valid email, got: %v", errors)
		}
	})

	t.Run("Invalid email", func(t *testing.T) {
		errors := validator.Validate("users", map[string]any{
			"name":  "John",
			"email": "invalid-email",
		})

		if errors == nil {
			t.Fatal("Expected validation errors for invalid email")
		}

		found := false
		for _, err := range errors.Errors {
			if err.Code == "INVALID_EMAIL" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected INVALID_EMAIL error")
		}
	})
}

func TestEmailRule(t *testing.T) {
	rule := EmailRule()

	testCases := []struct {
		email   string
		isValid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"user+tag@example.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"user@domain", false},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			err := rule(tc.email)
			if tc.isValid && err != nil {
				t.Errorf("Expected %s to be valid, got error: %v", tc.email, err)
			}
			if !tc.isValid && err == nil {
				t.Errorf("Expected %s to be invalid, got no error", tc.email)
			}
		})
	}
}

func TestMinLengthRule(t *testing.T) {
	rule := MinLengthRule("password", 8)

	t.Run("Valid length", func(t *testing.T) {
		err := rule("password123")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Too short", func(t *testing.T) {
		err := rule("short")
		if err == nil {
			t.Error("Expected error for short string")
		}
		if err.Code != "STRING_TOO_SHORT" {
			t.Errorf("Expected code STRING_TOO_SHORT, got %s", err.Code)
		}
	})
}

func TestMaxLengthRule(t *testing.T) {
	rule := MaxLengthRule("username", 20)

	t.Run("Valid length", func(t *testing.T) {
		err := rule("john_doe")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Too long", func(t *testing.T) {
		err := rule("this_username_is_way_too_long")
		if err == nil {
			t.Error("Expected error for long string")
		}
		if err.Code != "STRING_TOO_LONG" {
			t.Errorf("Expected code STRING_TOO_LONG, got %s", err.Code)
		}
	})
}

func TestRangeRule(t *testing.T) {
	rule := RangeRule("age", 0, 120)

	t.Run("Valid range int", func(t *testing.T) {
		err := rule(25)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Valid range float64", func(t *testing.T) {
		err := rule(float64(50))
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Below minimum", func(t *testing.T) {
		err := rule(-5)
		if err == nil {
			t.Error("Expected error for value below minimum")
		}
		if err.Code != "OUT_OF_RANGE" {
			t.Errorf("Expected code OUT_OF_RANGE, got %s", err.Code)
		}
	})

	t.Run("Above maximum", func(t *testing.T) {
		err := rule(150)
		if err == nil {
			t.Error("Expected error for value above maximum")
		}
	})
}

func TestPatternRule(t *testing.T) {
	phonePattern := regexp.MustCompile(`^\+?[0-9]{10,15}$`)
	rule := PatternRule("phone", phonePattern, "invalid phone number format")

	t.Run("Valid pattern", func(t *testing.T) {
		err := rule("+1234567890123")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Invalid pattern", func(t *testing.T) {
		err := rule("abc123")
		if err == nil {
			t.Error("Expected error for invalid pattern")
		}
		if err.Code != "PATTERN_MISMATCH" {
			t.Errorf("Expected code PATTERN_MISMATCH, got %s", err.Code)
		}
	})
}

func TestEnumRule(t *testing.T) {
	rule := EnumRule("status", []string{"pending", "active", "inactive"})

	t.Run("Valid enum value", func(t *testing.T) {
		err := rule("active")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Invalid enum value", func(t *testing.T) {
		err := rule("unknown")
		if err == nil {
			t.Error("Expected error for invalid enum value")
		}
		if err.Code != "INVALID_ENUM" {
			t.Errorf("Expected code INVALID_ENUM, got %s", err.Code)
		}
	})
}

func TestValidationErrors_Error(t *testing.T) {
	t.Run("No errors", func(t *testing.T) {
		errors := ValidationErrors{}
		if errors.Error() != "no validation errors" {
			t.Errorf("Expected 'no validation errors', got '%s'", errors.Error())
		}
	})

	t.Run("Multiple errors", func(t *testing.T) {
		errors := ValidationErrors{
			Errors: []ValidationError{
				{Field: "name", Message: "is required"},
				{Field: "email", Message: "invalid format"},
			},
		}

		errorStr := errors.Error()
		if errorStr != "name: is required; email: invalid format" {
			t.Errorf("Unexpected error string: %s", errorStr)
		}
	})
}

func TestValidationErrors_HasErrors(t *testing.T) {
	t.Run("No errors", func(t *testing.T) {
		errors := ValidationErrors{}
		if errors.HasErrors() {
			t.Error("Expected HasErrors to be false")
		}
	})

	t.Run("Has errors", func(t *testing.T) {
		errors := ValidationErrors{
			Errors: []ValidationError{
				{Field: "name", Message: "is required"},
			},
		}
		if !errors.HasErrors() {
			t.Error("Expected HasErrors to be true")
		}
	})
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "email",
		Message: "invalid format",
	}

	if err.Error() != "email: invalid format" {
		t.Errorf("Expected 'email: invalid format', got '%s'", err.Error())
	}
}

func TestULIDRule(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{
			name:    "Valid ULID",
			value:   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			wantErr: false,
		},
		{
			name:    "Invalid ULID - too short",
			value:   "01ARZ3NDEKTSV4RRFFQ69G5",
			wantErr: true,
		},
		{
			name:    "Invalid ULID - empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "Invalid ULID - invalid characters",
			value:   "ZZZZZZZZZZZZZZZZZZZZZZZZZZ",
			wantErr: true,
		},
		{
			name:    "Not a string",
			value:   12345,
			wantErr: false, // Type validation will catch this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := ULIDRule("id")
			err := rule(tt.value)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if err != nil && err.Code != "INVALID_ULID" {
				t.Errorf("expected error code INVALID_ULID, got %s", err.Code)
			}
		})
	}
}

func TestSchemaValidator_ValidateField_Decimal(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	validator := NewSchemaValidator(reg)

	column := registry.Column{Name: "price", Type: registry.TypeDecimal}

	t.Run("Valid decimal", func(t *testing.T) {
		err := validator.ValidateField("price", "123.45", column)
		if err != nil {
			t.Errorf("Expected no error for valid decimal, got: %v", err)
		}
	})

	t.Run("Valid decimal integer", func(t *testing.T) {
		err := validator.ValidateField("price", "100", column)
		if err != nil {
			t.Errorf("Expected no error for integer decimal, got: %v", err)
		}
	})

	t.Run("Valid decimal negative", func(t *testing.T) {
		err := validator.ValidateField("price", "-42.75", column)
		if err != nil {
			t.Errorf("Expected no error for negative decimal, got: %v", err)
		}
	})

	t.Run("Invalid decimal - not a string", func(t *testing.T) {
		err := validator.ValidateField("price", 123.45, column)
		if err == nil {
			t.Error("Expected error for non-string decimal value")
		}
		if err.Code != "INVALID_TYPE" {
			t.Errorf("Expected code INVALID_TYPE, got %s", err.Code)
		}
	})

	t.Run("Invalid decimal - excess precision", func(t *testing.T) {
		err := validator.ValidateField("price", "123.456", column)
		if err == nil {
			t.Error("Expected error for excess precision")
		}
		if err.Code != "INVALID_DECIMAL" {
			t.Errorf("Expected code INVALID_DECIMAL, got %s", err.Code)
		}
	})

	t.Run("Invalid decimal - non-numeric", func(t *testing.T) {
		err := validator.ValidateField("price", "abc", column)
		if err == nil {
			t.Error("Expected error for non-numeric string")
		}
		if err.Code != "INVALID_DECIMAL" {
			t.Errorf("Expected code INVALID_DECIMAL, got %s", err.Code)
		}
	})

	t.Run("Invalid decimal - scientific notation", func(t *testing.T) {
		err := validator.ValidateField("price", "1e10", column)
		if err == nil {
			t.Error("Expected error for scientific notation")
		}
		if err.Code != "INVALID_DECIMAL" {
			t.Errorf("Expected code INVALID_DECIMAL, got %s", err.Code)
		}
	})

	t.Run("Invalid decimal - trailing decimal point", func(t *testing.T) {
		err := validator.ValidateField("price", "123.", column)
		if err == nil {
			t.Error("Expected error for trailing decimal point")
		}
	})

	t.Run("Invalid decimal - leading decimal point", func(t *testing.T) {
		err := validator.ValidateField("price", ".45", column)
		if err == nil {
			t.Error("Expected error for leading decimal point")
		}
	})
}

func TestSchemaValidator_Validate_DecimalField(t *testing.T) {
	reg := registry.NewSchemaRegistry()
	collection := &registry.Collection{
		Name: "products",
		Columns: []registry.Column{
			{Name: "name", Type: registry.TypeString, Nullable: false},
			{Name: "price", Type: registry.TypeDecimal, Nullable: false},
			{Name: "tax", Type: registry.TypeDecimal, Nullable: true},
		},
	}
	reg.Set(collection)

	validator := NewSchemaValidator(reg)

	t.Run("Valid data with decimal", func(t *testing.T) {
		data := map[string]any{
			"name":  "Widget",
			"price": "199.99",
			"tax":   "15.00",
		}
		errors := validator.Validate("products", data)
		if errors != nil {
			t.Errorf("Expected no validation errors, got: %v", errors)
		}
	})

	t.Run("Valid data without optional decimal", func(t *testing.T) {
		data := map[string]any{
			"name":  "Widget",
			"price": "199.99",
		}
		errors := validator.Validate("products", data)
		if errors != nil {
			t.Errorf("Expected no validation errors, got: %v", errors)
		}
	})

	t.Run("Invalid decimal format", func(t *testing.T) {
		data := map[string]any{
			"name":  "Widget",
			"price": "abc",
		}
		errors := validator.Validate("products", data)
		if errors == nil {
			t.Fatal("Expected validation errors for invalid decimal")
		}
		found := false
		for _, err := range errors.Errors {
			if err.Code == "INVALID_DECIMAL" && err.Field == "price" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected INVALID_DECIMAL error for price field")
		}
	})
}
