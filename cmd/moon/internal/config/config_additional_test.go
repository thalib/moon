package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestValidate_InvalidPort_Zero tests validation with zero port
func TestValidate_InvalidPort_Zero(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 0
jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for port 0, got nil")
	}
}

// TestValidate_InvalidPort_Negative tests validation with negative port
func TestValidate_InvalidPort_Negative(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: -1
jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for negative port, got nil")
	}
}

// TestValidate_PortBoundary tests validation at port boundaries
func TestValidate_PortBoundary(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		wantError bool
	}{
		{"port 1 (valid)", 1, false},
		{"port 65535 (valid)", 65535, false},
		{"port 65536 (invalid)", 65536, true},
		{"port 80 (valid)", 80, false},
		{"port 443 (valid)", 443, false},
		{"port 8080 (valid)", 8080, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			content := fmt.Sprintf(`server:
  port: %d
jwt:
  secret: test-secret
`, tt.port)

			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			_, err := Load(configPath)
			gotError := err != nil

			if gotError != tt.wantError {
				t.Errorf("port %d: error = %v, wantError = %v", tt.port, err, tt.wantError)
			}
		})
	}
}

// TestValidate_RelativeDatabasePath tests that relative SQLite paths are converted to absolute
func TestValidate_RelativeDatabasePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `database:
  connection: sqlite
  database: relative/path/db.sqlite
jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should be converted to absolute path
	if !filepath.IsAbs(cfg.Database.Database) {
		t.Errorf("Expected absolute path, got relative: %s", cfg.Database.Database)
	}
}

// TestValidate_PostgresConnectionNotNormalized tests that non-SQLite paths aren't normalized
func TestValidate_PostgresConnectionNotNormalized(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `database:
  connection: postgres
  database: mydb
jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Postgres database name should not be converted to absolute path
	if cfg.Database.Database != "mydb" {
		t.Errorf("Expected 'mydb', got '%s'", cfg.Database.Database)
	}
}

// TestValidate_EmptyLoggingPath tests default logging path
func TestValidate_EmptyLoggingPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `jwt:
  secret: test-secret
logging:
  path: ""
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should use default logging path
	if cfg.Logging.Path != Defaults.Logging.Path {
		t.Errorf("Expected default logging path '%s', got '%s'", Defaults.Logging.Path, cfg.Logging.Path)
	}
}

// TestLoad_RecoveryConfig tests recovery configuration
func TestLoad_RecoveryConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `jwt:
  secret: test-secret
recovery:
  auto_repair: false
  drop_orphans: true
  check_timeout: 10
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Recovery.AutoRepair {
		t.Error("Expected AutoRepair to be false")
	}

	if !cfg.Recovery.DropOrphans {
		t.Error("Expected DropOrphans to be true")
	}

	if cfg.Recovery.CheckTimeout != 10 {
		t.Errorf("Expected CheckTimeout to be 10, got %d", cfg.Recovery.CheckTimeout)
	}
}

// TestLoad_RecoveryDefaults tests default recovery configuration
func TestLoad_RecoveryDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Defaults should be applied
	if !cfg.Recovery.AutoRepair {
		t.Error("Expected AutoRepair default to be true")
	}

	if cfg.Recovery.DropOrphans {
		t.Error("Expected DropOrphans default to be false")
	}
}

// TestAPIKeyConfig tests API key configuration
func TestAPIKeyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `jwt:
  secret: test-secret
apikey:
  enabled: true
  header: X-Custom-API-Key
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.APIKey.Enabled {
		t.Error("Expected APIKey.Enabled to be true")
	}

	if cfg.APIKey.Header != "X-Custom-API-Key" {
		t.Errorf("Expected APIKey.Header 'X-Custom-API-Key', got '%s'", cfg.APIKey.Header)
	}
}

// TestAPIKeyDefaultHeader tests default API key header
func TestAPIKeyDefaultHeader(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `jwt:
  secret: test-secret
apikey:
  enabled: true
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.APIKey.Header != Defaults.APIKey.Header {
		t.Errorf("Expected default APIKey.Header '%s', got '%s'", Defaults.APIKey.Header, cfg.APIKey.Header)
	}
}

// TestJWTExpiry tests JWT expiry configuration
func TestJWTExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `jwt:
  secret: test-secret
  expiry: 86400
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.JWT.Expiry != 86400 {
		t.Errorf("Expected JWT.Expiry 86400, got %d", cfg.JWT.Expiry)
	}
}

// TestJWTDefaultExpiry tests default JWT expiry
func TestJWTDefaultExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.JWT.Expiry != Defaults.JWT.Expiry {
		t.Errorf("Expected default JWT.Expiry %d, got %d", Defaults.JWT.Expiry, cfg.JWT.Expiry)
	}
}
