package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a minimal config file
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

	// Check default values from Defaults struct
	if cfg.Server.Port != Defaults.Server.Port {
		t.Errorf("Expected default port %d, got %d", Defaults.Server.Port, cfg.Server.Port)
	}

	if cfg.Server.Host != Defaults.Server.Host {
		t.Errorf("Expected default host %s, got %s", Defaults.Server.Host, cfg.Server.Host)
	}

	if cfg.Database.Connection != Defaults.Database.Connection {
		t.Errorf("Expected default connection %s, got %s", Defaults.Database.Connection, cfg.Database.Connection)
	}

	if cfg.Database.Database != Defaults.Database.Database {
		t.Errorf("Expected default database %s, got %s", Defaults.Database.Database, cfg.Database.Database)
	}

	if cfg.Logging.Path != Defaults.Logging.Path {
		t.Errorf("Expected default logging path %s, got %s", Defaults.Logging.Path, cfg.Logging.Path)
	}
}

func TestLoad_FromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 9090
  host: 127.0.0.1
database:
  connection: postgres
  database: testdb
  user: testuser
  password: testpass
  host: localhost
logging:
  path: /tmp/logs
jwt:
  secret: my-super-secret-key
  expiry: 7200
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

	if cfg.Database.Connection != "postgres" {
		t.Errorf("Expected postgres connection, got %s", cfg.Database.Connection)
	}

	if cfg.Database.Database != "testdb" {
		t.Errorf("Expected database testdb, got %s", cfg.Database.Database)
	}

	if cfg.Database.User != "testuser" {
		t.Errorf("Expected user testuser, got %s", cfg.Database.User)
	}

	if cfg.Logging.Path != "/tmp/logs" {
		t.Errorf("Expected logging path /tmp/logs, got %s", cfg.Logging.Path)
	}

	if cfg.JWT.Secret != "my-super-secret-key" {
		t.Errorf("Expected JWT secret, got %s", cfg.JWT.Secret)
	}

	if cfg.JWT.Expiry != 7200 {
		t.Errorf("Expected JWT expiry 7200, got %d", cfg.JWT.Expiry)
	}

	if !cfg.APIKey.Enabled {
		t.Error("Expected APIKey.Enabled to be true")
	}

	if cfg.APIKey.Header != "X-Custom-Key" {
		t.Errorf("Expected API key header X-Custom-Key, got %s", cfg.APIKey.Header)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  port: 6006
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing JWT secret, got nil")
	}

	if err != nil && err.Error() != "config validation failed: JWT secret is required (set in config file under jwt.secret)" {
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

func TestLoad_NoConfigFile_DefaultPath(t *testing.T) {
	// Reset global config for this test
	globalConfig = nil

	// Try to load with empty path (uses default path which likely doesn't exist)
	_, err := Load("")
	if err == nil {
		// If default config exists, that's fine - skip this test
		t.Skip("Default config file exists, skipping test")
	}

	// Should get error about missing config file
	if err == nil {
		t.Fatal("Expected error when default config file is missing")
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
	if cfg.Database.Connection != Defaults.Database.Connection {
		t.Errorf("Expected default SQLite connection %s, got %s", Defaults.Database.Connection, cfg.Database.Connection)
	}

	if cfg.Database.Database != Defaults.Database.Database {
		t.Errorf("Expected default database path %s, got %s", Defaults.Database.Database, cfg.Database.Database)
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

func TestLoad_CustomConfigFlag(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom-config.yaml")

	content := `server:
  port: 7777
jwt:
  secret: custom-secret
`

	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(customPath)
	if err != nil {
		t.Fatalf("Load() with custom path failed: %v", err)
	}

	if cfg.Server.Port != 7777 {
		t.Errorf("Expected port 7777 from custom config, got %d", cfg.Server.Port)
	}

	if cfg.JWT.Secret != "custom-secret" {
		t.Errorf("Expected custom-secret, got %s", cfg.JWT.Secret)
	}
}

func TestDefaults(t *testing.T) {
	// Verify that Defaults struct has correct values
	if Defaults.Server.Port != 6006 {
		t.Errorf("Expected default port 6006, got %d", Defaults.Server.Port)
	}

	if Defaults.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host 0.0.0.0, got %s", Defaults.Server.Host)
	}

	if Defaults.Database.Connection != "sqlite" {
		t.Errorf("Expected default connection sqlite, got %s", Defaults.Database.Connection)
	}

	if Defaults.Database.Database != "/opt/moon/sqlite.db" {
		t.Errorf("Expected default database /opt/moon/sqlite.db, got %s", Defaults.Database.Database)
	}

	if Defaults.Logging.Path != "/var/log/moon" {
		t.Errorf("Expected default logging path /var/log/moon, got %s", Defaults.Logging.Path)
	}

	if Defaults.JWT.Expiry != 3600 {
		t.Errorf("Expected default JWT expiry 3600, got %d", Defaults.JWT.Expiry)
	}

	if Defaults.APIKey.Header != "X-API-KEY" {
		t.Errorf("Expected default API key header X-API-KEY, got %s", Defaults.APIKey.Header)
	}

	if Defaults.ConfigPath != "/etc/moon.conf" {
		t.Errorf("Expected default config path /etc/moon.conf, got %s", Defaults.ConfigPath)
	}
}

