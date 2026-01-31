// Package decimal provides exact numeric handling for precision-critical values
// such as price, amount, weight, tax, and quantity. It uses Go's math/big.Rat
// for arbitrary precision arithmetic while maintaining a string-based API.
package decimal

import (
	"database/sql/driver"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/constants"
)

// decimalRegex validates decimal string format (e.g., "123.45", "-10.50", "0.01")
var decimalRegex = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// Decimal represents an exact decimal number backed by math/big.Rat.
// It provides arbitrary precision arithmetic without floating-point errors.
type Decimal struct {
	rat *big.Rat
}

// Zero returns a zero-valued Decimal.
func Zero() Decimal {
	return Decimal{rat: big.NewRat(0, 1)}
}

// New creates a new Decimal from numerator and denominator.
func New(numerator, denominator int64) Decimal {
	if denominator == 0 {
		return Zero()
	}
	return Decimal{rat: big.NewRat(numerator, denominator)}
}

// ParseDecimal parses a string into a Decimal with scale validation.
func ParseDecimal(s string) (Decimal, error) {
	return ParseDecimalWithScale(s, constants.DefaultDecimalScale)
}

// ParseDecimalWithScale parses a string into a Decimal with custom scale validation.
func ParseDecimalWithScale(s string, maxScale int) (Decimal, error) {
	if s == "" {
		return Zero(), fmt.Errorf("invalid decimal value: empty string")
	}

	// Trim whitespace
	s = strings.TrimSpace(s)

	// Check for scientific notation (not allowed)
	if strings.ContainsAny(s, "eE") {
		return Zero(), fmt.Errorf("invalid decimal value '%s': scientific notation not supported", s)
	}

	// Validate format with regex
	if !decimalRegex.MatchString(s) {
		return Zero(), fmt.Errorf("invalid decimal value '%s': not a valid number", s)
	}

	// Check for malformed decimals like "10." or ".50"
	if strings.HasSuffix(s, ".") {
		return Zero(), fmt.Errorf("invalid decimal value '%s': trailing decimal point", s)
	}
	if strings.HasPrefix(s, ".") || (len(s) > 1 && strings.HasPrefix(s, "-.")) {
		return Zero(), fmt.Errorf("invalid decimal value '%s': leading decimal point", s)
	}

	// Check decimal scale
	if idx := strings.Index(s, "."); idx >= 0 {
		scale := len(s) - idx - 1
		if scale > maxScale {
			return Zero(), fmt.Errorf("decimal precision exceeded: maximum scale is %d, got %d", maxScale, scale)
		}
	}

	// Parse with big.Rat
	rat := new(big.Rat)
	if _, ok := rat.SetString(s); !ok {
		return Zero(), fmt.Errorf("invalid decimal value '%s': parse error", s)
	}

	return Decimal{rat: rat}, nil
}

// MustParseDecimal parses a string into a Decimal and panics on error.
// Use only for test fixtures or static values known to be valid.
func MustParseDecimal(s string) Decimal {
	d, err := ParseDecimal(s)
	if err != nil {
		panic(err)
	}
	return d
}

// String returns the canonical string representation of the Decimal.
func (d Decimal) String() string {
	return d.StringWithScale(constants.DefaultDecimalScale)
}

// StringWithScale returns the string representation with a specific scale.
func (d Decimal) StringWithScale(scale int) string {
	if d.rat == nil {
		return formatWithScale(big.NewRat(0, 1), scale)
	}
	return formatWithScale(d.rat, scale)
}

// formatWithScale formats a big.Rat to a string with fixed scale.
func formatWithScale(rat *big.Rat, scale int) string {
	// Get the float representation
	f, _ := rat.Float64()

	// Format with the required scale
	format := fmt.Sprintf("%%.%df", scale)
	return fmt.Sprintf(format, f)
}

// Float64 returns the float64 approximation of the Decimal.
// Use with caution as this may lose precision.
func (d Decimal) Float64() float64 {
	if d.rat == nil {
		return 0
	}
	f, _ := d.rat.Float64()
	return f
}

// IsZero returns true if the Decimal is zero.
func (d Decimal) IsZero() bool {
	if d.rat == nil {
		return true
	}
	return d.rat.Sign() == 0
}

// Sign returns the sign of the Decimal (-1 for negative, 0 for zero, +1 for positive).
func (d Decimal) Sign() int {
	if d.rat == nil {
		return 0
	}
	return d.rat.Sign()
}

// Neg returns the negation of the Decimal.
func (d Decimal) Neg() Decimal {
	if d.rat == nil {
		return Zero()
	}
	result := new(big.Rat).Neg(d.rat)
	return Decimal{rat: result}
}

// Abs returns the absolute value of the Decimal.
func (d Decimal) Abs() Decimal {
	if d.rat == nil {
		return Zero()
	}
	result := new(big.Rat).Abs(d.rat)
	return Decimal{rat: result}
}

// Add returns the sum of d and other.
func (d Decimal) Add(other Decimal) Decimal {
	a := d.rat
	b := other.rat
	if a == nil {
		a = big.NewRat(0, 1)
	}
	if b == nil {
		b = big.NewRat(0, 1)
	}
	result := new(big.Rat).Add(a, b)
	return Decimal{rat: result}
}

// Sub returns the difference of d and other.
func (d Decimal) Sub(other Decimal) Decimal {
	a := d.rat
	b := other.rat
	if a == nil {
		a = big.NewRat(0, 1)
	}
	if b == nil {
		b = big.NewRat(0, 1)
	}
	result := new(big.Rat).Sub(a, b)
	return Decimal{rat: result}
}

// Mul returns the product of d and other.
func (d Decimal) Mul(other Decimal) Decimal {
	a := d.rat
	b := other.rat
	if a == nil {
		a = big.NewRat(0, 1)
	}
	if b == nil {
		b = big.NewRat(0, 1)
	}
	result := new(big.Rat).Mul(a, b)
	return Decimal{rat: result}
}

// Div returns the quotient of d and other.
// Returns an error if other is zero.
func (d Decimal) Div(other Decimal) (Decimal, error) {
	if other.IsZero() {
		return Zero(), fmt.Errorf("division by zero")
	}
	a := d.rat
	b := other.rat
	if a == nil {
		a = big.NewRat(0, 1)
	}
	result := new(big.Rat).Quo(a, b)
	return Decimal{rat: result}, nil
}

// Equal returns true if d equals other.
func (d Decimal) Equal(other Decimal) bool {
	return d.Compare(other) == 0
}

// Less returns true if d is less than other.
func (d Decimal) Less(other Decimal) bool {
	return d.Compare(other) < 0
}

// Greater returns true if d is greater than other.
func (d Decimal) Greater(other Decimal) bool {
	return d.Compare(other) > 0
}

// Compare compares d and other.
// Returns -1 if d < other, 0 if d == other, +1 if d > other.
func (d Decimal) Compare(other Decimal) int {
	a := d.rat
	b := other.rat
	if a == nil {
		a = big.NewRat(0, 1)
	}
	if b == nil {
		b = big.NewRat(0, 1)
	}
	return a.Cmp(b)
}

// MarshalJSON implements json.Marshaler.
// Decimals are serialized as JSON strings to preserve precision.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
// Decimals are deserialized from JSON strings.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		d.rat = big.NewRat(0, 1)
		return nil
	}

	// Remove quotes
	s := strings.Trim(string(data), `"`)

	parsed, err := ParseDecimal(s)
	if err != nil {
		return err
	}

	d.rat = parsed.rat
	return nil
}

// Scan implements sql.Scanner for database reads.
func (d *Decimal) Scan(value any) error {
	if value == nil {
		d.rat = big.NewRat(0, 1)
		return nil
	}

	switch v := value.(type) {
	case string:
		parsed, err := ParseDecimalWithScale(v, constants.MaxDecimalScale)
		if err != nil {
			return fmt.Errorf("failed to scan decimal: %w", err)
		}
		d.rat = parsed.rat
		return nil
	case []byte:
		parsed, err := ParseDecimalWithScale(string(v), constants.MaxDecimalScale)
		if err != nil {
			return fmt.Errorf("failed to scan decimal: %w", err)
		}
		d.rat = parsed.rat
		return nil
	case int64:
		d.rat = big.NewRat(v, 1)
		return nil
	case float64:
		d.rat = new(big.Rat).SetFloat64(v)
		return nil
	default:
		return fmt.Errorf("unsupported type for decimal scan: %T", value)
	}
}

// Value implements driver.Valuer for database writes.
// Returns the string representation for database storage.
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// ValidateDecimalString validates a decimal string without parsing it.
// Returns an error if the string is not a valid decimal with the given scale.
func ValidateDecimalString(s string, maxScale int) error {
	_, err := ParseDecimalWithScale(s, maxScale)
	return err
}

// ValidateDecimalStringForField validates a decimal string for a specific field.
// Returns an error with the field name included in the message.
func ValidateDecimalStringForField(fieldName, value string, maxScale int) error {
	if value == "" {
		return fmt.Errorf("invalid decimal value for field '%s': empty string", fieldName)
	}

	value = strings.TrimSpace(value)

	if strings.ContainsAny(value, "eE") {
		return fmt.Errorf("invalid decimal value for field '%s': scientific notation not supported", fieldName)
	}

	if !decimalRegex.MatchString(value) {
		return fmt.Errorf("invalid decimal value for field '%s': '%s' is not a valid number", fieldName, value)
	}

	if strings.HasSuffix(value, ".") {
		return fmt.Errorf("invalid decimal value for field '%s': trailing decimal point", fieldName)
	}
	if strings.HasPrefix(value, ".") || (len(value) > 1 && strings.HasPrefix(value, "-.")) {
		return fmt.Errorf("invalid decimal value for field '%s': leading decimal point", fieldName)
	}

	if idx := strings.Index(value, "."); idx >= 0 {
		scale := len(value) - idx - 1
		if scale > maxScale {
			return fmt.Errorf("decimal precision exceeded for field '%s': maximum scale is %d, got %d", fieldName, maxScale, scale)
		}
	}

	return nil
}
