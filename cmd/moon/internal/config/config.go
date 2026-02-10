// Package config provides configuration management for the Moon application.
// It uses YAML-only configuration with centralized defaults and no environment
// variable overrides, following the principles defined in SPEC.md.
package config

import (
	"fmt"
	"log"
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
		Connection         string
		Database           string
		User               string
		Password           string
		Host               string
		QueryTimeout       int
		SlowQueryThreshold int
	}
	Logging struct {
		Path            string
		RedactSensitive bool
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
		Endpoints        []CORSEndpointConfig
	}
	Pagination struct {
		DefaultPageSize int
		MaxPageSize     int
	}
	Limits struct {
		MaxCollections          int
		MaxColumnsPerCollection int
		MaxFiltersPerRequest    int
		MaxSortFieldsPerRequest int
	}
	Batch struct {
		MaxSize         int
		MaxPayloadBytes int
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
		Connection         string
		Database           string
		User               string
		Password           string
		Host               string
		QueryTimeout       int
		SlowQueryThreshold int
	}{
		Connection:         "sqlite",
		Database:           "/opt/moon/sqlite.db",
		User:               "",
		Password:           "",
		Host:               "0.0.0.0",
		QueryTimeout:       30,  // 30 seconds
		SlowQueryThreshold: 500, // 500 milliseconds
	},
	Logging: struct {
		Path            string
		RedactSensitive bool
	}{
		Path:            "/var/log/moon",
		RedactSensitive: true,
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
		Header:  "X-API-KEY", // DEPRECATED: Retained for config parsing only. Not used in code (removed in PRD-059).
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
		Endpoints        []CORSEndpointConfig
	}{
		Enabled:          false, // Disabled by default for security
		AllowedOrigins:   []string{},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           3600, // 1 hour
		Endpoints: []CORSEndpointConfig{
			{
				Path:             "/health",
				PatternType:      "exact",
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: false,
				BypassAuth:       true,
			},
			{
				Path:             "/doc/", // Prefix pattern matches /doc, /doc/, /doc/llms-full.txt, etc.
				PatternType:      "prefix",
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: false,
				BypassAuth:       true,
			},
			// Data endpoints - Dynamic collection access (requires authentication)
			// These defaults allow CORS preflight but require actual requests to authenticate
			{
				Path:             "*:list",
				PatternType:      "suffix",
				AllowedOrigins:   []string{},
				AllowedMethods:   []string{"GET", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
				BypassAuth:       false,
			},
			{
				Path:             "*:get",
				PatternType:      "suffix",
				AllowedOrigins:   []string{},
				AllowedMethods:   []string{"GET", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
				BypassAuth:       false,
			},
			{
				Path:             "*:schema",
				PatternType:      "suffix",
				AllowedOrigins:   []string{},
				AllowedMethods:   []string{"GET", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
				BypassAuth:       false,
			},
			{
				Path:             "*:create",
				PatternType:      "suffix",
				AllowedOrigins:   []string{},
				AllowedMethods:   []string{"POST", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
				BypassAuth:       false,
			},
			{
				Path:             "*:update",
				PatternType:      "suffix",
				AllowedOrigins:   []string{},
				AllowedMethods:   []string{"POST", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
				BypassAuth:       false,
			},
			{
				Path:             "*:destroy",
				PatternType:      "suffix",
				AllowedOrigins:   []string{},
				AllowedMethods:   []string{"POST", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
				BypassAuth:       false,
			},
		},
	},
	Pagination: struct {
		DefaultPageSize int
		MaxPageSize     int
	}{
		DefaultPageSize: 15,  // Default page size
		MaxPageSize:     200, // Maximum page size
	},
	Limits: struct {
		MaxCollections          int
		MaxColumnsPerCollection int
		MaxFiltersPerRequest    int
		MaxSortFieldsPerRequest int
	}{
		MaxCollections:          1000,
		MaxColumnsPerCollection: 100,
		MaxFiltersPerRequest:    20,
		MaxSortFieldsPerRequest: 5,
	},
	Batch: struct {
		MaxSize         int
		MaxPayloadBytes int
	}{
		MaxSize:         50,
		MaxPayloadBytes: 2097152, // 2 MB
	},
	ConfigPath: "/etc/moon.conf",
}

// AppConfig holds the application configuration.
// It is designed to be immutable after initialization.
type AppConfig struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	APIKey     APIKeyConfig     `mapstructure:"apikey"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Recovery   RecoveryConfig   `mapstructure:"recovery"`
	CORS       CORSConfig       `mapstructure:"cors"`
	Pagination PaginationConfig `mapstructure:"pagination"`
	Limits     LimitsConfig     `mapstructure:"limits"`
	Batch      BatchConfig      `mapstructure:"batch"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port   int    `mapstructure:"port"`
	Host   string `mapstructure:"host"`
	Prefix string `mapstructure:"prefix"`
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Connection         string `mapstructure:"connection"`           // database type: sqlite, postgres, mysql
	Database           string `mapstructure:"database"`             // database file/name
	User               string `mapstructure:"user"`                 // database user
	Password           string `mapstructure:"password"`             // database password
	Host               string `mapstructure:"host"`                 // database host
	QueryTimeout       int    `mapstructure:"query_timeout"`        // query timeout in seconds
	SlowQueryThreshold int    `mapstructure:"slow_query_threshold"` // slow query threshold in milliseconds
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Path                      string   `mapstructure:"path"`                        // log directory path
	RedactSensitive           bool     `mapstructure:"redact_sensitive"`            // redact sensitive data in logs
	AdditionalSensitiveFields []string `mapstructure:"additional_sensitive_fields"` // additional fields to redact
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
	Header  string `mapstructure:"header"` // Deprecated: Now always uses Authorization Bearer
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
	Enabled          bool                 `mapstructure:"enabled"`           // enable CORS support (default: false for security)
	AllowedOrigins   []string             `mapstructure:"allowed_origins"`   // list of allowed origins (e.g., ["https://example.com"])
	AllowedMethods   []string             `mapstructure:"allowed_methods"`   // list of allowed HTTP methods
	AllowedHeaders   []string             `mapstructure:"allowed_headers"`   // list of allowed request headers
	AllowCredentials bool                 `mapstructure:"allow_credentials"` // allow credentials (cookies, auth headers)
	MaxAge           int                  `mapstructure:"max_age"`           // preflight cache duration in seconds
	Endpoints        []CORSEndpointConfig `mapstructure:"endpoints"`         // endpoint-specific CORS registration (PRD-058)
}

// CORSEndpointConfig represents a single CORS endpoint registration
type CORSEndpointConfig struct {
	Path             string   `mapstructure:"path"`              // endpoint path or pattern
	PatternType      string   `mapstructure:"pattern_type"`      // pattern matching type: exact, prefix, suffix, contains
	AllowedOrigins   []string `mapstructure:"allowed_origins"`   // allowed origins for this endpoint
	AllowedMethods   []string `mapstructure:"allowed_methods"`   // allowed HTTP methods for this endpoint
	AllowedHeaders   []string `mapstructure:"allowed_headers"`   // allowed request headers for this endpoint
	AllowCredentials bool     `mapstructure:"allow_credentials"` // allow credentials for this endpoint
	BypassAuth       bool     `mapstructure:"bypass_auth"`       // skip authentication for this endpoint (default: false)
}

// PaginationConfig holds pagination settings for list endpoints.
type PaginationConfig struct {
	DefaultPageSize int `mapstructure:"default_page_size"` // default number of records per page
	MaxPageSize     int `mapstructure:"max_page_size"`     // maximum allowed page size
}

// LimitsConfig holds system limits for schema and query constraints.
type LimitsConfig struct {
	MaxCollections          int `mapstructure:"max_collections"`             // maximum number of collections
	MaxColumnsPerCollection int `mapstructure:"max_columns_per_collection"`  // maximum columns per collection
	MaxFiltersPerRequest    int `mapstructure:"max_filters_per_request"`     // maximum filter parameters per request
	MaxSortFieldsPerRequest int `mapstructure:"max_sort_fields_per_request"` // maximum sort fields per request
}

// BatchConfig holds batch operation configuration (PRD-064)
type BatchConfig struct {
	MaxSize         int `mapstructure:"max_size"`          // maximum number of items per batch request
	MaxPayloadBytes int `mapstructure:"max_payload_bytes"` // maximum payload size in bytes
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
	v.SetDefault("database.query_timeout", Defaults.Database.QueryTimeout)
	v.SetDefault("database.slow_query_threshold", Defaults.Database.SlowQueryThreshold)
	v.SetDefault("logging.path", Defaults.Logging.Path)
	v.SetDefault("logging.redact_sensitive", Defaults.Logging.RedactSensitive)
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
	v.SetDefault("cors.endpoints", Defaults.CORS.Endpoints)
	v.SetDefault("pagination.default_page_size", Defaults.Pagination.DefaultPageSize)
	v.SetDefault("pagination.max_page_size", Defaults.Pagination.MaxPageSize)
	v.SetDefault("limits.max_collections", Defaults.Limits.MaxCollections)
	v.SetDefault("limits.max_columns_per_collection", Defaults.Limits.MaxColumnsPerCollection)
	v.SetDefault("limits.max_filters_per_request", Defaults.Limits.MaxFiltersPerRequest)
	v.SetDefault("limits.max_sort_fields_per_request", Defaults.Limits.MaxSortFieldsPerRequest)
	v.SetDefault("batch.max_size", Defaults.Batch.MaxSize)
	v.SetDefault("batch.max_payload_bytes", Defaults.Batch.MaxPayloadBytes)

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
	// Apply database defaults for new fields
	if cfg.Database.QueryTimeout <= 0 {
		cfg.Database.QueryTimeout = Defaults.Database.QueryTimeout
	}
	if cfg.Database.SlowQueryThreshold <= 0 {
		cfg.Database.SlowQueryThreshold = Defaults.Database.SlowQueryThreshold
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

	// Validate pagination configuration (PRD-046)
	if cfg.Pagination.DefaultPageSize <= 0 {
		cfg.Pagination.DefaultPageSize = Defaults.Pagination.DefaultPageSize
	}
	if cfg.Pagination.MaxPageSize <= 0 {
		cfg.Pagination.MaxPageSize = Defaults.Pagination.MaxPageSize
	}
	// Ensure default_page_size <= max_page_size
	if cfg.Pagination.DefaultPageSize > cfg.Pagination.MaxPageSize {
		return fmt.Errorf("pagination.default_page_size (%d) cannot exceed pagination.max_page_size (%d)",
			cfg.Pagination.DefaultPageSize, cfg.Pagination.MaxPageSize)
	}

	// Validate limits configuration (PRD-048)
	if cfg.Limits.MaxCollections <= 0 {
		cfg.Limits.MaxCollections = Defaults.Limits.MaxCollections
	}
	if cfg.Limits.MaxColumnsPerCollection <= 0 {
		cfg.Limits.MaxColumnsPerCollection = Defaults.Limits.MaxColumnsPerCollection
	}
	if cfg.Limits.MaxFiltersPerRequest <= 0 {
		cfg.Limits.MaxFiltersPerRequest = Defaults.Limits.MaxFiltersPerRequest
	}
	if cfg.Limits.MaxSortFieldsPerRequest <= 0 {
		cfg.Limits.MaxSortFieldsPerRequest = Defaults.Limits.MaxSortFieldsPerRequest
	}

	// Validate batch configuration (apply defaults if missing or zero)
	if cfg.Batch.MaxSize <= 0 {
		cfg.Batch.MaxSize = Defaults.Batch.MaxSize
	}
	if cfg.Batch.MaxPayloadBytes <= 0 {
		cfg.Batch.MaxPayloadBytes = Defaults.Batch.MaxPayloadBytes
	}

	// Validate CORS endpoint configuration (PRD-058)
	if err := validateCORSEndpoints(&cfg.CORS); err != nil {
		return fmt.Errorf("CORS configuration validation failed: %w", err)
	}

	return nil
}

// validateCORSEndpoints validates CORS endpoint configuration
func validateCORSEndpoints(cors *CORSConfig) error {
	validPatternTypes := map[string]bool{
		"exact":    true,
		"prefix":   true,
		"suffix":   true,
		"contains": true,
	}

	for i, endpoint := range cors.Endpoints {
		// Validate path is non-empty
		if endpoint.Path == "" {
			return fmt.Errorf("cors.endpoints[%d]: path cannot be empty", i)
		}

		// Validate pattern_type
		if endpoint.PatternType == "" {
			// Default to "exact" if not specified
			cors.Endpoints[i].PatternType = "exact"
		} else if !validPatternTypes[endpoint.PatternType] {
			return fmt.Errorf("cors.endpoints[%d]: invalid pattern_type '%s', must be one of: exact, prefix, suffix, contains", i, endpoint.PatternType)
		}

		// allowed_origins can be empty - it will fall back to global allowed_origins
		// This allows for flexible endpoint-specific configurations
		// Note: We skip all origin-specific validations for empty origins since
		// the global configuration will be used at runtime and validated separately
		if len(endpoint.AllowedOrigins) == 0 {
			// Still log bypass_auth for audit trail even without explicit origins
			if endpoint.BypassAuth {
				log.Printf("INFO: CORS endpoint registered with authentication bypass: %s (%s pattern)", endpoint.Path, endpoint.PatternType)
			}
			continue
		}

		// Check for wildcard mixed with specific origins
		hasWildcard := false
		hasSpecific := false
		for _, origin := range endpoint.AllowedOrigins {
			if origin == "*" {
				hasWildcard = true
			} else {
				hasSpecific = true
			}
		}
		if hasWildcard && hasSpecific {
			return fmt.Errorf("cors.endpoints[%d]: cannot mix wildcard '*' with specific origins", i)
		}

		// Warn if allow_credentials with wildcard origin (unusual but not invalid)
		if endpoint.AllowCredentials && hasWildcard {
			log.Printf("WARN: cors.endpoints[%d] (%s): allow_credentials=true with wildcard origin '*' is not recommended and may not work in modern browsers", i, endpoint.Path)
		}

		// Warn if bypass_auth with non-wildcard origins (unusual configuration)
		if endpoint.BypassAuth && !hasWildcard {
			log.Printf("WARN: cors.endpoints[%d] (%s): bypass_auth=true with restricted origins is an unusual configuration", i, endpoint.Path)
		}

		// Log authentication bypass for audit trail
		if endpoint.BypassAuth {
			log.Printf("INFO: CORS endpoint registered with authentication bypass: %s (%s pattern)", endpoint.Path, endpoint.PatternType)
		}
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
