package config

import (
	"testing"
)

// TestValidateCORSEndpoints tests CORS endpoint validation
func TestValidateCORSEndpoints(t *testing.T) {
	tests := []struct {
		name        string
		cors        CORSConfig
		wantError   bool
		errContains string
		description string
	}{
		{
			name: "valid_exact_pattern",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/health",
						PatternType:    "exact",
						AllowedOrigins: []string{"*"},
					},
				},
			},
			wantError:   false,
			description: "Valid exact pattern should pass",
		},
		{
			name: "valid_prefix_pattern",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/doc/*",
						PatternType:    "prefix",
						AllowedOrigins: []string{"*"},
					},
				},
			},
			wantError:   false,
			description: "Valid prefix pattern should pass",
		},
		{
			name: "valid_suffix_pattern",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "*.json",
						PatternType:    "suffix",
						AllowedOrigins: []string{"*"},
					},
				},
			},
			wantError:   false,
			description: "Valid suffix pattern should pass",
		},
		{
			name: "valid_contains_pattern",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/public/",
						PatternType:    "contains",
						AllowedOrigins: []string{"*"},
					},
				},
			},
			wantError:   false,
			description: "Valid contains pattern should pass",
		},
		{
			name: "empty_path",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "",
						PatternType:    "exact",
						AllowedOrigins: []string{"*"},
					},
				},
			},
			wantError:   true,
			errContains: "path cannot be empty",
			description: "Empty path should fail validation",
		},
		{
			name: "invalid_pattern_type",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/health",
						PatternType:    "regex",
						AllowedOrigins: []string{"*"},
					},
				},
			},
			wantError:   true,
			errContains: "invalid pattern_type",
			description: "Invalid pattern type should fail validation",
		},
		{
			name: "empty_allowed_origins",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/health",
						PatternType:    "exact",
						AllowedOrigins: []string{},
					},
				},
			},
			wantError:   false,
			errContains: "",
			description: "Empty allowed_origins is valid - falls back to global origins",
		},
		{
			name: "wildcard_mixed_with_specific",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/health",
						PatternType:    "exact",
						AllowedOrigins: []string{"*", "https://example.com"},
					},
				},
			},
			wantError:   true,
			errContains: "cannot mix wildcard",
			description: "Wildcard mixed with specific origins should fail validation",
		},
		{
			name: "default_pattern_type",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/health",
						PatternType:    "", // Empty, should default to "exact"
						AllowedOrigins: []string{"*"},
					},
				},
			},
			wantError:   false,
			description: "Missing pattern_type should default to exact",
		},
		{
			name: "multiple_valid_endpoints",
			cors: CORSConfig{
				Endpoints: []CORSEndpointConfig{
					{
						Path:           "/health",
						PatternType:    "exact",
						AllowedOrigins: []string{"*"},
					},
					{
						Path:           "/doc/*",
						PatternType:    "prefix",
						AllowedOrigins: []string{"*"},
					},
					{
						Path:           "/webhooks/*",
						PatternType:    "prefix",
						AllowedOrigins: []string{"https://partner.example.com"},
					},
				},
			},
			wantError:   false,
			description: "Multiple valid endpoints should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCORSEndpoints(&tt.cors)

			if tt.wantError {
				if err == nil {
					t.Errorf("validateCORSEndpoints() error = nil, want error containing %q for %s", tt.errContains, tt.description)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("validateCORSEndpoints() error = %v, want error containing %q for %s", err, tt.errContains, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("validateCORSEndpoints() error = %v, want nil for %s", err, tt.description)
				}
			}

			// Check that empty pattern_type was defaulted to "exact"
			if !tt.wantError && tt.cors.Endpoints[0].PatternType == "" {
				// After validation, it should be set to "exact"
				// Note: This test checks the behavior described in FR-1.3
			}
		})
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestCORSEndpointConfig_DefaultPatternType tests that pattern_type defaults to exact
func TestCORSEndpointConfig_DefaultPatternType(t *testing.T) {
	cors := CORSConfig{
		Endpoints: []CORSEndpointConfig{
			{
				Path:           "/health",
				PatternType:    "", // Empty, should be set to "exact"
				AllowedOrigins: []string{"*"},
			},
		},
	}

	err := validateCORSEndpoints(&cors)
	if err != nil {
		t.Errorf("validateCORSEndpoints() error = %v, want nil", err)
	}

	if cors.Endpoints[0].PatternType != "exact" {
		t.Errorf("PatternType = %v, want 'exact' (default)", cors.Endpoints[0].PatternType)
	}
}
