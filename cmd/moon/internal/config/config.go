// Package config provides configuration management for the Moon application.
// It uses YAML-only configuration with centralized defaults and no environment
// variable overrides, following the principles defined in SPEC.md.
package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	// VersionMajor is the major version number
	VersionMajor = 1
	// VersionMinor is the minor version number
	VersionMinor = 99
	// RootMessage is the plain text response for the root endpoint.
	RootMessage = "Moon is running."
)

// Version returns the version string in format {major}.{minor}
func Version() string {
	return fmt.Sprintf("%d.%d", VersionMajor, VersionMinor)
}

// Defaults contains all default configuration values
// centralized in one place to avoid hardcoded literals
var Defaults = struct {
	Server struct {
		Port   int
		Host   string
		Prefix string
	}
	Database struct {
		Connection string
		Database   string
		User       string
		Password   string
		Host       string
	}
	Logging struct {
		Path string
	}
	JWT struct {
		Expiry        int
		AccessExpiry  int
		RefreshExpiry int
	}
	APIKey struct {
		Enabled bool
		Header  string
	}
	Auth struct {
		RateLimit struct {
			UserRPM       int
			APIKeyRPM     int
			LoginAttempts int
			LoginWindow   int
		}
	}
	Recovery struct {
		AutoRepair   bool
		DropOrphans  bool
		CheckTimeout int
	}
	CORS struct {
		Enabled          bool
		AllowedOrigins   []string
		AllowedMethods   []string
		AllowedHeaders   []string
		AllowCredentials bool
		MaxAge           int
	}
	ConfigPath string
}{
	Server: struct {
		Port   int
		Host   string
		Prefix string
	}{
		Port:   6006,
		Host:   "0.0.0.0",
		Prefix: "",
	},
	Database: struct {
		Connection string
		Database   string
		User       string
		Password   string
		Host       string
	}{
		Connection: "sqlite",
		Database:   "/opt/moon/sqlite.db",
		User:       "",
		Password:   "",
		Host:       "0.0.0.0",
	},
	Logging: struct {
		Path string
	}{
		Path: "/var/log/moon",
	},
	JWT: struct {
		Expiry        int
		AccessExpiry  int
		RefreshExpiry int
	}{
		Expiry:        3600,
		AccessExpiry:  3600,   // 1 hour
		RefreshExpiry: 604800, // 7 days
	},
	APIKey: struct {
		Enabled bool
		Header  string
	}{
		Enabled: false,
		Header:  "X-API-KEY",
	},
	Auth: struct {
		RateLimit struct {
			UserRPM       int
			APIKeyRPM     int
			LoginAttempts int
			LoginWindow   int
		}
	}{
		RateLimit: struct {
			UserRPM       int
			APIKeyRPM     int
			LoginAttempts int
			LoginWindow   int
		}{
			UserRPM:       100,
			APIKeyRPM:     1000,
			LoginAttempts: 5,
			LoginWindow:   900, // 15 minutes
		},
	},
	Recovery: struct {
		AutoRepair   bool
		DropOrphans  bool
		CheckTimeout int
	}{
		AutoRepair:   true,
		DropOrphans:  false,
		CheckTimeout: 5,
	},
	CORS: struct {
		Enabled          bool
		AllowedOrigins   []string
		AllowedMethods   []string
		AllowedHeaders   []string
		AllowCredentials bool
		MaxAge           int
	}{
		Enabled:          false, // Disabled by default for security
		AllowedOrigins:   []string{},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           3600, // 1 hour
	},
	ConfigPath: "/etc/moon.conf",
}

// AppConfig holds the application configuration.
// It is designed to be immutable after initialization.
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	APIKey   APIKeyConfig   `mapstructure:"apikey"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Recovery RecoveryConfig `mapstructure:"recovery"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port   int    `mapstructure:"port"`
	Host   string `mapstructure:"host"`
	Prefix string `mapstructure:"prefix"`
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Connection string `mapstructure:"connection"` // database type: sqlite, postgres, mysql
	Database   string `mapstructure:"database"`   // database file/name
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	Host       string `mapstructure:"host"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Path string `mapstructure:"path"` // log directory path
}

// JWTConfig holds JWT authentication configuration.
type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	Expiry        int    `mapstructure:"expiry"`         // in seconds (deprecated, use access_expiry)
	AccessExpiry  int    `mapstructure:"access_expiry"`  // access token expiry in seconds
	RefreshExpiry int    `mapstructure:"refresh_expiry"` // refresh token expiry in seconds
}

// APIKeyConfig holds API key configuration.
type APIKeyConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Header  string `mapstructure:"header"`
}

// AuthConfig holds authentication and rate limiting configuration.
type AuthConfig struct {
	BootstrapAdmin BootstrapAdminConfig `mapstructure:"bootstrap_admin"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
}

// BootstrapAdminConfig holds bootstrap admin user configuration.
type BootstrapAdminConfig struct {
	Username string `mapstructure:"username"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	UserRPM       int `mapstructure:"user_rpm"`       // requests per minute for authenticated users
	APIKeyRPM     int `mapstructure:"apikey_rpm"`     // requests per minute for API keys
	LoginAttempts int `mapstructure:"login_attempts"` // max login attempts before lockout
	LoginWindow   int `mapstructure:"login_window"`   // lockout window in seconds
}

// RecoveryConfig holds database recovery and consistency check configuration.
type RecoveryConfig struct {
	AutoRepair   bool `mapstructure:"auto_repair"`   // automatically repair inconsistencies
	DropOrphans  bool `mapstructure:"drop_orphans"`  // drop orphaned tables (admin-controlled)
	CheckTimeout int  `mapstructure:"check_timeout"` // consistency check timeout in seconds
}

// CORSConfig holds CORS (Cross-Origin Resource Sharing) configuration.
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`           // enable CORS support (default: false for security)
	AllowedOrigins   []string `mapstructure:"allowed_origins"`   // list of allowed origins (e.g., ["https://example.com"])
	AllowedMethods   []string `mapstructure:"allowed_methods"`   // list of allowed HTTP methods
	AllowedHeaders   []string `mapstructure:"allowed_headers"`   // list of allowed request headers
	AllowCredentials bool     `mapstructure:"allow_credentials"` // allow credentials (cookies, auth headers)
	MaxAge           int      `mapstructure:"max_age"`           // preflight cache duration in seconds
}

var globalConfig *AppConfig

// Load initializes and loads the application configuration.
// It reads from YAML config files only.
// No environment variable overrides are supported (YAML-only approach).
func Load(configPath string) (*AppConfig, error) {
	v := viper.New()

	// Set default values from centralized Defaults struct
	v.SetDefault("server.port", Defaults.Server.Port)
	v.SetDefault("server.host", Defaults.Server.Host)
	v.SetDefault("server.prefix", Defaults.Server.Prefix)
	v.SetDefault("database.connection", Defaults.Database.Connection)
	v.SetDefault("database.database", Defaults.Database.Database)
	v.SetDefault("database.user", Defaults.Database.User)
	v.SetDefault("database.password", Defaults.Database.Password)
	v.SetDefault("database.host", Defaults.Database.Host)
	v.SetDefault("logging.path", Defaults.Logging.Path)
	v.SetDefault("jwt.expiry", Defaults.JWT.Expiry)
	v.SetDefault("jwt.access_expiry", Defaults.JWT.AccessExpiry)
	v.SetDefault("jwt.refresh_expiry", Defaults.JWT.RefreshExpiry)
	v.SetDefault("apikey.enabled", Defaults.APIKey.Enabled)
	v.SetDefault("apikey.header", Defaults.APIKey.Header)
	v.SetDefault("auth.rate_limit.user_rpm", Defaults.Auth.RateLimit.UserRPM)
	v.SetDefault("auth.rate_limit.apikey_rpm", Defaults.Auth.RateLimit.APIKeyRPM)
	v.SetDefault("auth.rate_limit.login_attempts", Defaults.Auth.RateLimit.LoginAttempts)
	v.SetDefault("auth.rate_limit.login_window", Defaults.Auth.RateLimit.LoginWindow)
	v.SetDefault("recovery.auto_repair", Defaults.Recovery.AutoRepair)
	v.SetDefault("recovery.drop_orphans", Defaults.Recovery.DropOrphans)
	v.SetDefault("recovery.check_timeout", Defaults.Recovery.CheckTimeout)
	v.SetDefault("cors.enabled", Defaults.CORS.Enabled)
	v.SetDefault("cors.allowed_origins", Defaults.CORS.AllowedOrigins)
	v.SetDefault("cors.allowed_methods", Defaults.CORS.AllowedMethods)
	v.SetDefault("cors.allowed_headers", Defaults.CORS.AllowedHeaders)
	v.SetDefault("cors.allow_credentials", Defaults.CORS.AllowCredentials)
	v.SetDefault("cors.max_age", Defaults.CORS.MaxAge)

	// Configure Viper to read from YAML config file only
	// Explicitly disable TOML support
	v.SetConfigType("yaml")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Use default config path
		v.SetConfigFile(Defaults.ConfigPath)
	}

	// Read config file (optional - continue if file doesn't exist)
	if err := v.ReadInConfig(); err != nil {
		// If a specific config file was requested but not found, that's an error
		if configPath != "" {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				return nil, fmt.Errorf("config file not found: %s", configPath)
			}
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// If using default path and file not found, that's OK - use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration into struct
	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Store in global variable for thread-safe read-only access
	globalConfig = &cfg

	return &cfg, nil
}

// validate checks that required configuration fields are present.
func validate(cfg *AppConfig) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	// Normalize prefix: add leading slash if missing, preserve trailing slash
	if cfg.Server.Prefix != "" && !strings.HasPrefix(cfg.Server.Prefix, "/") {
		cfg.Server.Prefix = "/" + cfg.Server.Prefix
	}

	// Apply default database values if not provided
	if cfg.Database.Connection == "" {
		cfg.Database.Connection = Defaults.Database.Connection
	}
	if cfg.Database.Database == "" {
		cfg.Database.Database = Defaults.Database.Database
	}

	// For SQLite, normalize database path to absolute
	if cfg.Database.Connection == Defaults.Database.Connection && !filepath.IsAbs(cfg.Database.Database) {
		absPath, err := filepath.Abs(cfg.Database.Database)
		if err != nil {
			return fmt.Errorf("failed to resolve database path: %w", err)
		}
		cfg.Database.Database = absPath
	}

	// Apply default logging path if not provided
	if cfg.Logging.Path == "" {
		cfg.Logging.Path = Defaults.Logging.Path
	}

	// JWT secret is required for authentication
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required (set in config file under jwt.secret)")
	}

	return nil
}

// Get returns the global configuration instance.
// This is thread-safe as the config is immutable after Load().
func Get() *AppConfig {
	if globalConfig == nil {
		panic("configuration not loaded - call config.Load() first")
	}
	return globalConfig
}
