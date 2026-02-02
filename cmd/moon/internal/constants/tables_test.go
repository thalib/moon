package constants

import "testing"

// TestTableConstants verifies system table name constants.
func TestTableConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"Users table", TableUsers, "moon_users"},
		{"Refresh tokens table", TableRefreshTokens, "moon_refresh_tokens"},
		{"API keys table", TableAPIKeys, "moon_apikeys"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

// TestSystemTables verifies SystemTables list contains all expected tables.
func TestSystemTables(t *testing.T) {
	expectedTables := []string{
		"moon_users",
		"moon_refresh_tokens",
		"moon_apikeys",
	}

	if len(SystemTables) != len(expectedTables) {
		t.Errorf("Expected %d system tables, got %d", len(expectedTables), len(SystemTables))
	}

	// Check that all expected tables are present
	tableMap := make(map[string]bool)
	for _, table := range SystemTables {
		tableMap[table] = true
	}

	for _, expected := range expectedTables {
		if !tableMap[expected] {
			t.Errorf("Expected system table %q not found", expected)
		}
	}
}

// TestIsSystemTable verifies the IsSystemTable function.
func TestIsSystemTable(t *testing.T) {
	tests := []struct {
		name       string
		tableName  string
		wantSystem bool
	}{
		{"Users table is system", "moon_users", true},
		{"Refresh tokens table is system", "moon_refresh_tokens", true},
		{"API keys table is system", "moon_apikeys", true},
		{"Regular table is not system", "products", false},
		{"Regular table with moon prefix is not system", "moon_products", false},
		{"Empty string is not system", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSystemTable(tt.tableName)
			if got != tt.wantSystem {
				t.Errorf("IsSystemTable(%q) = %v, want %v", tt.tableName, got, tt.wantSystem)
			}
		})
	}
}
