package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create an empty config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := `jwt:
  secret: test-secret-key-for-testing
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check default values
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}

	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Expected default max_open_conns 25, got %d", cfg.Database.MaxOpenConns)
	}
}

func TestLoad_FromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 9090
  host: 127.0.0.1
database:
  connection_string: postgres://user:pass@localhost/db
  max_open_conns: 50
jwt:
  secret: my-super-secret-key
  expiration: 7200
apikey:
  enabled: true
  header: X-Custom-Key
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Server.Host)
	}

	if cfg.Database.ConnectionString != "postgres://user:pass@localhost/db" {
		t.Errorf("Expected postgres connection string, got %s", cfg.Database.ConnectionString)
	}

	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("Expected max_open_conns 50, got %d", cfg.Database.MaxOpenConns)
	}

	if cfg.JWT.Secret != "my-super-secret-key" {
		t.Errorf("Expected JWT secret, got %s", cfg.JWT.Secret)
	}

	if cfg.JWT.Expiration != 7200 {
		t.Errorf("Expected JWT expiration 7200, got %d", cfg.JWT.Expiration)
	}

	if !cfg.APIKey.Enabled {
		t.Error("Expected APIKey.Enabled to be true")
	}

	if cfg.APIKey.Header != "X-Custom-Key" {
		t.Errorf("Expected API key header X-Custom-Key, got %s", cfg.APIKey.Header)
	}
}

func TestLoad_EnvironmentOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 8080
jwt:
  secret: file-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set environment variables
	os.Setenv("MOON_SERVER_PORT", "3000")
	os.Setenv("MOON_JWT_SECRET", "env-secret")
	defer os.Unsetenv("MOON_SERVER_PORT")
	defer os.Unsetenv("MOON_JWT_SECRET")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Environment variables should override file config
	if cfg.Server.Port != 3000 {
		t.Errorf("Expected port 3000 from env, got %d", cfg.Server.Port)
	}

	if cfg.JWT.Secret != "env-secret" {
		t.Errorf("Expected JWT secret from env, got %s", cfg.JWT.Secret)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 8080
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Ensure JWT secret is not in environment
	os.Unsetenv("MOON_JWT_SECRET")

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing JWT secret, got nil")
	}

	if err != nil && err.Error() != "config validation failed: JWT secret is required (set MOON_JWT_SECRET)" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 99999
jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

func TestLoad_NoConfigFile_SpecifiedPath(t *testing.T) {
	// Try to load with non-existent config file path specified
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Expected error when specific config file is missing")
	}
}

func TestLoad_NoConfigFile_SearchPath(t *testing.T) {
	// Reset global config for this test
	globalConfig = nil

	// Set required environment variables for the search test
	os.Setenv("MOON_JWT_SECRET", "test-secret")
	defer os.Unsetenv("MOON_JWT_SECRET")

	// Try to load with empty path (searches for config files)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() should not fail when searching for config files: %v", err)
	}

	// Should use default values
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}

	// JWT secret should be loaded from environment
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("Expected JWT secret from env, got %s", cfg.JWT.Secret)
	}
}

func TestLoad_DefaultSQLiteConnection(t *testing.T) {
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

	// Should default to SQLite
	if cfg.Database.ConnectionString != "sqlite://moon.db" {
		t.Errorf("Expected default SQLite connection, got %s", cfg.Database.ConnectionString)
	}
}

func TestGet_WithoutLoad(t *testing.T) {
	// Reset global config
	globalConfig = nil

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling Get() before Load()")
		}
	}()

	Get()
}

func TestGet_AfterLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 8888
jwt:
  secret: test-secret
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Get should return the loaded config
	cfg := Get()
	if cfg.Server.Port != 8888 {
		t.Errorf("Expected port 8888 from Get(), got %d", cfg.Server.Port)
	}
}
