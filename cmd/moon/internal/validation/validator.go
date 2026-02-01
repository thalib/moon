// Package validation provides input validation for API requests.
// It validates collection names, column definitions, and data payloads
// against schema definitions stored in the registry.
package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/decimal"
	"github.com/thalib/moon/cmd/moon/internal/registry"
	moonulid "github.com/thalib/moon/cmd/moon/internal/ulid"
)

// ValidationMode determines how the validator handles unknown fields
type ValidationMode int

const (
	// StrictMode rejects unknown fields
	StrictMode ValidationMode = iota
	// PermissiveMode ignores unknown fields
	PermissiveMode
)

// ValidationError represents a single validation error
type ValidationError struct {
	Field        string `json:"field"`
	Message      string `json:"message"`
	ExpectedType string `json:"expected_type,omitempty"`
	ActualValue  any    `json:"actual_value,omitempty"`
	Code         string `json:"code"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (e ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}

	messages := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

// HasErrors returns true if there are validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validator validates request data against collection schemas
type Validator interface {
	// Validate validates data against a collection schema
	Validate(collectionName string, data map[string]any) *ValidationErrors

	// ValidateField validates a single field value
	ValidateField(fieldName string, value any, column registry.Column) *ValidationError

	// SetMode sets the validation mode (strict or permissive)
	SetMode(mode ValidationMode)
}

// SchemaValidator implements the Validator interface
type SchemaValidator struct {
	registry    *registry.SchemaRegistry
	mode        ValidationMode
	customRules map[string]CustomRule
}

// CustomRule represents a custom validation rule for a column
type CustomRule struct {
	ColumnName string
	Validator  func(value any) *ValidationError
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(reg *registry.SchemaRegistry) *SchemaValidator {
	return &SchemaValidator{
		registry:    reg,
		mode:        StrictMode,
		customRules: make(map[string]CustomRule),
	}
}

// SetMode sets the validation mode
func (v *SchemaValidator) SetMode(mode ValidationMode) {
	v.mode = mode
}

// AddCustomRule adds a custom validation rule for a column
func (v *SchemaValidator) AddCustomRule(collectionName, columnName string, rule func(value any) *ValidationError) {
	key := collectionName + "." + columnName
	v.customRules[key] = CustomRule{
		ColumnName: columnName,
		Validator:  rule,
	}
}

// Validate validates data against a collection schema
func (v *SchemaValidator) Validate(collectionName string, data map[string]any) *ValidationErrors {
	errors := &ValidationErrors{}

	// Check if collection exists
	collection, exists := v.registry.Get(collectionName)
	if !exists {
		errors.Errors = append(errors.Errors, ValidationError{
			Field:   "collection",
			Message: fmt.Sprintf("collection '%s' does not exist", collectionName),
			Code:    "COLLECTION_NOT_FOUND",
		})
		return errors
	}

	// Build a map of valid column names
	columnMap := make(map[string]registry.Column)
	for _, col := range collection.Columns {
		columnMap[col.Name] = col
	}

	// Check for unknown fields in strict mode
	if v.mode == StrictMode {
		for fieldName := range data {
			if _, ok := columnMap[fieldName]; !ok {
				errors.Errors = append(errors.Errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("unknown field '%s'", fieldName),
					Code:    "UNKNOWN_FIELD",
				})
			}
		}
	}

	// Validate each column
	for _, col := range collection.Columns {
		value, exists := data[col.Name]

		// Check required fields
		if !exists || value == nil {
			if !col.Nullable && col.DefaultValue == nil {
				errors.Errors = append(errors.Errors, ValidationError{
					Field:   col.Name,
					Message: fmt.Sprintf("required field '%s' is missing", col.Name),
					Code:    "REQUIRED_FIELD",
				})
			}
			continue
		}

		// Validate field type
		if fieldErr := v.ValidateField(col.Name, value, col); fieldErr != nil {
			errors.Errors = append(errors.Errors, *fieldErr)
		}

		// Apply custom rules
		key := collectionName + "." + col.Name
		if rule, ok := v.customRules[key]; ok {
			if customErr := rule.Validator(value); customErr != nil {
				errors.Errors = append(errors.Errors, *customErr)
			}
		}
	}

	if len(errors.Errors) == 0 {
		return nil
	}
	return errors
}

// ValidateField validates a single field value against its column definition
func (v *SchemaValidator) ValidateField(fieldName string, value any, column registry.Column) *ValidationError {
	// Check type
	if err := v.validateType(fieldName, value, column.Type); err != nil {
		return err
	}

	// Additional validation based on type
	switch column.Type {
	case registry.TypeString:
		if str, ok := value.(string); ok {
			if err := v.validateStringConstraints(fieldName, str); err != nil {
				return err
			}
		}
	case registry.TypeDatetime:
		if str, ok := value.(string); ok {
			if err := v.validateDatetime(fieldName, str); err != nil {
				return err
			}
		}
	case registry.TypeDecimal:
		if str, ok := value.(string); ok {
			if err := v.validateDecimal(fieldName, str); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateType validates that a value matches the expected column type
func (v *SchemaValidator) validateType(fieldName string, value any, expectedType registry.ColumnType) *ValidationError {
	switch expectedType {
	case registry.TypeString:
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:        fieldName,
				Message:      fmt.Sprintf("field '%s' must be a string", fieldName),
				ExpectedType: string(expectedType),
				ActualValue:  value,
				Code:         "INVALID_TYPE",
			}
		}

	case registry.TypeInteger:
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// Valid integer types
		case float64:
			// JSON numbers are float64, check if it's a whole number
			if v != float64(int64(v)) {
				return &ValidationError{
					Field:        fieldName,
					Message:      fmt.Sprintf("field '%s' must be an integer, got float", fieldName),
					ExpectedType: string(expectedType),
					ActualValue:  value,
					Code:         "INVALID_TYPE",
				}
			}
		default:
			return &ValidationError{
				Field:        fieldName,
				Message:      fmt.Sprintf("field '%s' must be an integer", fieldName),
				ExpectedType: string(expectedType),
				ActualValue:  value,
				Code:         "INVALID_TYPE",
			}
		}

	case registry.TypeBoolean:
		if _, ok := value.(bool); !ok {
			return &ValidationError{
				Field:        fieldName,
				Message:      fmt.Sprintf("field '%s' must be a boolean", fieldName),
				ExpectedType: string(expectedType),
				ActualValue:  value,
				Code:         "INVALID_TYPE",
			}
		}

	case registry.TypeDatetime:
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:        fieldName,
				Message:      fmt.Sprintf("field '%s' must be a datetime string", fieldName),
				ExpectedType: string(expectedType),
				ActualValue:  value,
				Code:         "INVALID_TYPE",
			}
		}

	case registry.TypeDecimal:
		// Decimal values are passed as strings in JSON
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:        fieldName,
				Message:      fmt.Sprintf("field '%s' must be a decimal string", fieldName),
				ExpectedType: string(expectedType),
				ActualValue:  value,
				Code:         "INVALID_TYPE",
			}
		}

	case registry.TypeJSON:
		// JSON can be any type (object, array, string, number, boolean, null)
		// So we just accept any value here
	}

	return nil
}

// validateStringConstraints validates string-specific constraints
// Note: Since all strings now map to TEXT, there is no length limit
func (v *SchemaValidator) validateStringConstraints(fieldName string, value string) *ValidationError {
	// No length constraints for TEXT type
	return nil
}

// validateDatetime validates datetime string format
func (v *SchemaValidator) validateDatetime(fieldName, value string) *ValidationError {
	// Support multiple common datetime formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, value); err == nil {
			return nil
		}
	}

	return &ValidationError{
		Field:       fieldName,
		Message:     fmt.Sprintf("field '%s' has invalid datetime format", fieldName),
		ActualValue: value,
		Code:        "INVALID_DATETIME",
	}
}

// validateDecimal validates decimal string format and precision
func (v *SchemaValidator) validateDecimal(fieldName, value string) *ValidationError {
	if err := decimal.ValidateDecimalStringForField(fieldName, value, constants.DefaultDecimalScale); err != nil {
		return &ValidationError{
			Field:       fieldName,
			Message:     err.Error(),
			ActualValue: value,
			Code:        "INVALID_DECIMAL",
		}
	}
	return nil
}

// Common custom validation rules

// EmailRule creates a validation rule for email fields
func EmailRule() func(value any) *ValidationError {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	return func(value any) *ValidationError {
		str, ok := value.(string)
		if !ok {
			return nil // Type validation will catch this
		}

		if !emailRegex.MatchString(str) {
			return &ValidationError{
				Field:       "email",
				Message:     "invalid email format",
				ActualValue: str,
				Code:        "INVALID_EMAIL",
			}
		}
		return nil
	}
}

// MinLengthRule creates a validation rule for minimum string length
func MinLengthRule(fieldName string, minLength int) func(value any) *ValidationError {
	return func(value any) *ValidationError {
		str, ok := value.(string)
		if !ok {
			return nil // Type validation will catch this
		}

		if len(str) < minLength {
			return &ValidationError{
				Field:       fieldName,
				Message:     fmt.Sprintf("field '%s' must be at least %d characters", fieldName, minLength),
				ActualValue: len(str),
				Code:        "STRING_TOO_SHORT",
			}
		}
		return nil
	}
}

// MaxLengthRule creates a validation rule for maximum string length
func MaxLengthRule(fieldName string, maxLength int) func(value any) *ValidationError {
	return func(value any) *ValidationError {
		str, ok := value.(string)
		if !ok {
			return nil // Type validation will catch this
		}

		if len(str) > maxLength {
			return &ValidationError{
				Field:       fieldName,
				Message:     fmt.Sprintf("field '%s' must be at most %d characters", fieldName, maxLength),
				ActualValue: len(str),
				Code:        "STRING_TOO_LONG",
			}
		}
		return nil
	}
}

// RangeRule creates a validation rule for numeric range
func RangeRule(fieldName string, min, max float64) func(value any) *ValidationError {
	return func(value any) *ValidationError {
		var num float64
		switch v := value.(type) {
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		case float64:
			num = v
		default:
			return nil // Type validation will catch this
		}

		if num < min || num > max {
			return &ValidationError{
				Field:       fieldName,
				Message:     fmt.Sprintf("field '%s' must be between %v and %v", fieldName, min, max),
				ActualValue: num,
				Code:        "OUT_OF_RANGE",
			}
		}
		return nil
	}
}

// PatternRule creates a validation rule for string pattern matching
func PatternRule(fieldName string, pattern *regexp.Regexp, message string) func(value any) *ValidationError {
	return func(value any) *ValidationError {
		str, ok := value.(string)
		if !ok {
			return nil // Type validation will catch this
		}

		if !pattern.MatchString(str) {
			return &ValidationError{
				Field:       fieldName,
				Message:     message,
				ActualValue: str,
				Code:        "PATTERN_MISMATCH",
			}
		}
		return nil
	}
}

// EnumRule creates a validation rule for enum values
func EnumRule(fieldName string, allowedValues []string) func(value any) *ValidationError {
	valueSet := make(map[string]bool)
	for _, v := range allowedValues {
		valueSet[v] = true
	}

	return func(value any) *ValidationError {
		str, ok := value.(string)
		if !ok {
			return nil // Type validation will catch this
		}

		if !valueSet[str] {
			return &ValidationError{
				Field:       fieldName,
				Message:     fmt.Sprintf("field '%s' must be one of: %v", fieldName, allowedValues),
				ActualValue: str,
				Code:        "INVALID_ENUM",
			}
		}
		return nil
	}
}

// ULIDRule creates a validation rule for ULID fields
func ULIDRule(fieldName string) func(value any) *ValidationError {
	return func(value any) *ValidationError {
		str, ok := value.(string)
		if !ok {
			return nil // Type validation will catch this
		}

		if err := moonulid.Validate(str); err != nil {
			return &ValidationError{
				Field:       fieldName,
				Message:     fmt.Sprintf("field '%s' must be a valid ULID", fieldName),
				ActualValue: str,
				Code:        "INVALID_ULID",
			}
		}
		return nil
	}
}
