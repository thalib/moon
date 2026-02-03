package schema

import (
	"net/url"
	"testing"
)

func TestParseQueryParameter(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected QueryMode
	}{
		{"no_schema_param", "", ModeNone},
		{"schema_no_value", "schema", ModeBoth},
		{"schema_true", "schema=true", ModeBoth},
		{"schema_only", "schema=only", ModeOnly},
		{"schema_false", "schema=false", ModeNone},
		{"schema_ONLY_uppercase", "schema=ONLY", ModeOnly},
		{"schema_invalid_value", "schema=invalid", ModeBoth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			result := ParseQueryParameter(values)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
