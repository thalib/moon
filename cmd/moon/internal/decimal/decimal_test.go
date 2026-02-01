package decimal

import (
	"encoding/json"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/constants"
)

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantStr string
		wantErr bool
	}{
		// Valid inputs
		{name: "integer", input: "10", wantStr: "10.00", wantErr: false},
		{name: "with decimals", input: "10.50", wantStr: "10.50", wantErr: false},
		{name: "price", input: "1299.99", wantStr: "1299.99", wantErr: false},
		{name: "negative", input: "-42.75", wantStr: "-42.75", wantErr: false},
		{name: "small", input: "0.01", wantStr: "0.01", wantErr: false},
		{name: "zero", input: "0", wantStr: "0.00", wantErr: false},
		{name: "zero with decimals", input: "0.00", wantStr: "0.00", wantErr: false},
		{name: "negative zero", input: "-0", wantStr: "0.00", wantErr: false},
		{name: "large number", input: "999999.99", wantStr: "999999.99", wantErr: false},
		{name: "one decimal", input: "10.5", wantStr: "10.50", wantErr: false},
		{name: "whitespace", input: "  10.50  ", wantStr: "10.50", wantErr: false},

		// Invalid inputs
		{name: "non-numeric", input: "abc", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
		{name: "scientific notation", input: "1e10", wantErr: true},
		{name: "scientific notation caps", input: "1E10", wantErr: true},
		{name: "excess precision", input: "10.999", wantErr: true},
		{name: "trailing decimal", input: "10.", wantErr: true},
		{name: "leading decimal", input: ".50", wantErr: true},
		{name: "negative leading decimal", input: "-.50", wantErr: true},
		{name: "comma separator", input: "1,234.56", wantErr: true},
		{name: "multiple decimals", input: "1.2.3", wantErr: true},
		{name: "letters in number", input: "12a34", wantErr: true},
		{name: "currency symbol", input: "$10.00", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDecimal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDecimal(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && d.String() != tt.wantStr {
				t.Errorf("ParseDecimal(%q).String() = %q, want %q", tt.input, d.String(), tt.wantStr)
			}
		})
	}
}

func TestParseDecimalWithScale(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxScale int
		wantStr  string
		wantErr  bool
	}{
		{name: "scale 2 valid", input: "10.99", maxScale: 2, wantStr: "10.99", wantErr: false},
		{name: "scale 2 invalid", input: "10.999", maxScale: 2, wantErr: true},
		{name: "scale 4 valid", input: "10.9999", maxScale: 4, wantStr: "10.9999", wantErr: false},
		{name: "scale 4 too many", input: "10.99999", maxScale: 4, wantErr: true},
		{name: "scale 10 valid", input: "1.1234567890", maxScale: 10, wantStr: "1.1234567890", wantErr: false},
		{name: "scale 10 invalid", input: "1.12345678901", maxScale: 10, wantErr: true},
		{name: "no decimal always valid", input: "100", maxScale: 2, wantStr: "100.00", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDecimalWithScale(tt.input, tt.maxScale)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDecimalWithScale(%q, %d) error = %v, wantErr %v", tt.input, tt.maxScale, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				got := d.StringWithScale(tt.maxScale)
				if got != tt.wantStr {
					t.Errorf("ParseDecimalWithScale(%q, %d).StringWithScale() = %q, want %q", tt.input, tt.maxScale, got, tt.wantStr)
				}
			}
		})
	}
}

func TestDecimalArithmetic(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		a := MustParseDecimal("10.50")
		b := MustParseDecimal("5.25")
		result := a.Add(b)
		if result.String() != "15.75" {
			t.Errorf("Add: got %s, want 15.75", result.String())
		}
	})

	t.Run("Sub", func(t *testing.T) {
		a := MustParseDecimal("10.50")
		b := MustParseDecimal("5.25")
		result := a.Sub(b)
		if result.String() != "5.25" {
			t.Errorf("Sub: got %s, want 5.25", result.String())
		}
	})

	t.Run("Mul", func(t *testing.T) {
		a := MustParseDecimal("10.00")
		b := MustParseDecimal("5.50")
		result := a.Mul(b)
		if result.String() != "55.00" {
			t.Errorf("Mul: got %s, want 55.00", result.String())
		}
	})

	t.Run("Div", func(t *testing.T) {
		a := MustParseDecimal("10.50")
		b := MustParseDecimal("5.25")
		result, err := a.Div(b)
		if err != nil {
			t.Fatalf("Div: unexpected error: %v", err)
		}
		if result.String() != "2.00" {
			t.Errorf("Div: got %s, want 2.00", result.String())
		}
	})

	t.Run("Div by zero", func(t *testing.T) {
		a := MustParseDecimal("10.50")
		zero := Zero()
		_, err := a.Div(zero)
		if err == nil {
			t.Error("Div by zero: expected error, got nil")
		}
	})

	t.Run("Complex calculation", func(t *testing.T) {
		// (100.00 + 50.00) * 0.10 = 15.00
		a := MustParseDecimal("100.00")
		b := MustParseDecimal("50.00")
		rate := MustParseDecimal("0.10")

		sum := a.Add(b)
		result := sum.Mul(rate)
		if result.String() != "15.00" {
			t.Errorf("Complex: got %s, want 15.00", result.String())
		}
	})
}

func TestDecimalComparison(t *testing.T) {
	a := MustParseDecimal("10.50")
	b := MustParseDecimal("5.25")
	c := MustParseDecimal("10.50")

	t.Run("Equal", func(t *testing.T) {
		if !a.Equal(c) {
			t.Errorf("Equal: 10.50 should equal 10.50")
		}
		if a.Equal(b) {
			t.Errorf("Equal: 10.50 should not equal 5.25")
		}
	})

	t.Run("Less", func(t *testing.T) {
		if !b.Less(a) {
			t.Errorf("Less: 5.25 should be less than 10.50")
		}
		if a.Less(b) {
			t.Errorf("Less: 10.50 should not be less than 5.25")
		}
	})

	t.Run("Greater", func(t *testing.T) {
		if !a.Greater(b) {
			t.Errorf("Greater: 10.50 should be greater than 5.25")
		}
		if b.Greater(a) {
			t.Errorf("Greater: 5.25 should not be greater than 10.50")
		}
	})

	t.Run("Compare", func(t *testing.T) {
		if a.Compare(c) != 0 {
			t.Errorf("Compare: 10.50 vs 10.50 should be 0")
		}
		if a.Compare(b) != 1 {
			t.Errorf("Compare: 10.50 vs 5.25 should be 1")
		}
		if b.Compare(a) != -1 {
			t.Errorf("Compare: 5.25 vs 10.50 should be -1")
		}
	})
}

func TestDecimalJSON(t *testing.T) {
	t.Run("Marshal", func(t *testing.T) {
		d := MustParseDecimal("123.45")
		data, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		if string(data) != `"123.45"` {
			t.Errorf("Marshal: got %s, want \"123.45\"", string(data))
		}
	})

	t.Run("Unmarshal", func(t *testing.T) {
		var d Decimal
		err := json.Unmarshal([]byte(`"123.45"`), &d)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if d.String() != "123.45" {
			t.Errorf("Unmarshal: got %s, want 123.45", d.String())
		}
	})

	t.Run("Unmarshal invalid", func(t *testing.T) {
		var d Decimal
		err := json.Unmarshal([]byte(`"invalid"`), &d)
		if err == nil {
			t.Error("Unmarshal invalid: expected error, got nil")
		}
	})

	t.Run("Unmarshal null", func(t *testing.T) {
		var d Decimal
		err := json.Unmarshal([]byte(`null`), &d)
		if err != nil {
			t.Fatalf("Unmarshal null error: %v", err)
		}
		if !d.IsZero() {
			t.Errorf("Unmarshal null: expected zero, got %s", d.String())
		}
	})

	t.Run("Marshal in struct", func(t *testing.T) {
		type Product struct {
			Name  string  `json:"name"`
			Price Decimal `json:"price"`
		}
		p := Product{Name: "Widget", Price: MustParseDecimal("199.99")}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("Marshal struct error: %v", err)
		}
		expected := `{"name":"Widget","price":"199.99"}`
		if string(data) != expected {
			t.Errorf("Marshal struct: got %s, want %s", string(data), expected)
		}
	})

	t.Run("Unmarshal in struct", func(t *testing.T) {
		type Product struct {
			Name  string  `json:"name"`
			Price Decimal `json:"price"`
		}
		var p Product
		err := json.Unmarshal([]byte(`{"name":"Widget","price":"199.99"}`), &p)
		if err != nil {
			t.Fatalf("Unmarshal struct error: %v", err)
		}
		if p.Name != "Widget" || p.Price.String() != "199.99" {
			t.Errorf("Unmarshal struct: got %+v", p)
		}
	})
}

func TestDecimalScan(t *testing.T) {
	t.Run("Scan string", func(t *testing.T) {
		var d Decimal
		err := d.Scan("123.45")
		if err != nil {
			t.Fatalf("Scan string error: %v", err)
		}
		if d.String() != "123.45" {
			t.Errorf("Scan string: got %s, want 123.45", d.String())
		}
	})

	t.Run("Scan bytes", func(t *testing.T) {
		var d Decimal
		err := d.Scan([]byte("123.45"))
		if err != nil {
			t.Fatalf("Scan bytes error: %v", err)
		}
		if d.String() != "123.45" {
			t.Errorf("Scan bytes: got %s, want 123.45", d.String())
		}
	})

	t.Run("Scan int64", func(t *testing.T) {
		var d Decimal
		err := d.Scan(int64(12345))
		if err != nil {
			t.Fatalf("Scan int64 error: %v", err)
		}
		if d.String() != "12345.00" {
			t.Errorf("Scan int64: got %s, want 12345.00", d.String())
		}
	})

	t.Run("Scan float64", func(t *testing.T) {
		var d Decimal
		err := d.Scan(float64(123.45))
		if err != nil {
			t.Fatalf("Scan float64 error: %v", err)
		}
		if d.String() != "123.45" {
			t.Errorf("Scan float64: got %s, want 123.45", d.String())
		}
	})

	t.Run("Scan nil", func(t *testing.T) {
		var d Decimal
		err := d.Scan(nil)
		if err != nil {
			t.Fatalf("Scan nil error: %v", err)
		}
		if !d.IsZero() {
			t.Errorf("Scan nil: expected zero, got %s", d.String())
		}
	})

	t.Run("Scan unsupported type", func(t *testing.T) {
		var d Decimal
		err := d.Scan(true)
		if err == nil {
			t.Error("Scan unsupported type: expected error, got nil")
		}
	})
}

func TestDecimalValue(t *testing.T) {
	d := MustParseDecimal("123.45")
	v, err := d.Value()
	if err != nil {
		t.Fatalf("Value error: %v", err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value: expected string, got %T", v)
	}
	if s != "123.45" {
		t.Errorf("Value: got %s, want 123.45", s)
	}
}

func TestDecimalHelpers(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		z := Zero()
		if !z.IsZero() {
			t.Error("Zero: IsZero should be true")
		}
		if z.String() != "0.00" {
			t.Errorf("Zero: got %s, want 0.00", z.String())
		}
	})

	t.Run("New", func(t *testing.T) {
		d := New(1, 2) // 0.5
		if d.String() != "0.50" {
			t.Errorf("New(1,2): got %s, want 0.50", d.String())
		}
	})

	t.Run("Sign", func(t *testing.T) {
		pos := MustParseDecimal("10.00")
		neg := MustParseDecimal("-10.00")
		zero := Zero()

		if pos.Sign() != 1 {
			t.Errorf("Sign positive: got %d, want 1", pos.Sign())
		}
		if neg.Sign() != -1 {
			t.Errorf("Sign negative: got %d, want -1", neg.Sign())
		}
		if zero.Sign() != 0 {
			t.Errorf("Sign zero: got %d, want 0", zero.Sign())
		}
	})

	t.Run("Neg", func(t *testing.T) {
		d := MustParseDecimal("10.00")
		n := d.Neg()
		if n.String() != "-10.00" {
			t.Errorf("Neg: got %s, want -10.00", n.String())
		}
	})

	t.Run("Abs", func(t *testing.T) {
		d := MustParseDecimal("-10.00")
		a := d.Abs()
		if a.String() != "10.00" {
			t.Errorf("Abs: got %s, want 10.00", a.String())
		}
	})

	t.Run("Float64", func(t *testing.T) {
		d := MustParseDecimal("123.45")
		f := d.Float64()
		if f != 123.45 {
			t.Errorf("Float64: got %f, want 123.45", f)
		}
	})
}

func TestValidateDecimalString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		maxScale int
		wantErr  bool
	}{
		{name: "valid", value: "123.45", maxScale: 2, wantErr: false},
		{name: "invalid", value: "abc", maxScale: 2, wantErr: true},
		{name: "excess scale", value: "123.456", maxScale: 2, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDecimalString(tt.value, tt.maxScale)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDecimalString(%q, %d) error = %v, wantErr %v", tt.value, tt.maxScale, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDecimalStringForField(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		value       string
		maxScale    int
		wantErr     bool
		errContains string
	}{
		{name: "valid", fieldName: "price", value: "123.45", maxScale: 2, wantErr: false},
		{name: "empty", fieldName: "price", value: "", maxScale: 2, wantErr: true, errContains: "price"},
		{name: "not a number", fieldName: "amount", value: "abc", maxScale: 2, wantErr: true, errContains: "amount"},
		{name: "excess scale", fieldName: "total", value: "123.456", maxScale: 2, wantErr: true, errContains: "total"},
		{name: "scientific notation", fieldName: "value", value: "1e10", maxScale: 2, wantErr: true, errContains: "scientific notation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDecimalStringForField(tt.fieldName, tt.value, tt.maxScale)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDecimalStringForField(%q, %q, %d) error = %v, wantErr %v", tt.fieldName, tt.value, tt.maxScale, err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateDecimalStringForField error should contain %q, got %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestConstantsUsed(t *testing.T) {
	// Verify that constants are correctly used
	if constants.DefaultDecimalScale != 2 {
		t.Errorf("DefaultDecimalScale: got %d, want 2", constants.DefaultDecimalScale)
	}
	if constants.MaxDecimalScale != 10 {
		t.Errorf("MaxDecimalScale: got %d, want 10", constants.MaxDecimalScale)
	}

	// Test that ParseDecimal uses DefaultDecimalScale
	_, err := ParseDecimal("10.999") // 3 decimal places should fail with default scale of 2
	if err == nil {
		t.Error("ParseDecimal should enforce DefaultDecimalScale of 2")
	}

	// Test that scale beyond max is rejected
	longDecimal := "1.12345678901" // 11 decimal places
	_, err = ParseDecimalWithScale(longDecimal, constants.MaxDecimalScale)
	if err == nil {
		t.Error("ParseDecimalWithScale should reject scale > MaxDecimalScale")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
