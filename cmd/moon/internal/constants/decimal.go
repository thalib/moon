package constants

// Decimal type constants for precision-critical numeric values.
const (
	// DefaultDecimalScale is the default number of decimal places for Decimal fields.
	// Used when scale is not explicitly defined in the schema.
	// Used in: decimal/decimal.go, validation/validator.go
	// Default: 2 decimal places (e.g., "123.45")
	DefaultDecimalScale = 2

	// MaxDecimalScale is the maximum allowed number of decimal places.
	// This is a hard limit enforced during validation.
	// Used in: decimal/decimal.go, validation/validator.go
	// Default: 10 decimal places
	MaxDecimalScale = 10

	// DefaultDecimalPrecision is the default total number of digits for Decimal storage.
	// Used for SQL DECIMAL(p,s) where p is precision and s is scale.
	// Used in: query/builder.go
	// Default: 19 digits total (allows values up to 9,999,999,999,999,999.99)
	DefaultDecimalPrecision = 19
)
