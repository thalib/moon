package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// AppConfig holds the application configuration.
// It is designed to be immutable after initialization.
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	APIKey   APIKeyConfig   `mapstructure:"apikey"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	ConnectionString string `mapstructure:"connection_string"`
	MaxOpenConns     int    `mapstructure:"max_open_conns"`
	MaxIdleConns     int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime  int    `mapstructure:"conn_max_lifetime"` // in seconds
}

// JWTConfig holds JWT authentication configuration.
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"` // in seconds
}

// APIKeyConfig holds API key configuration.
type APIKeyConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Header  string `mapstructure:"header"`
}

var globalConfig *AppConfig

// Load initializes and loads the application configuration.
// It reads from config files (YAML/TOML) and environment variables.
// Environment variables take precedence over file-based configuration.
func Load(configPath string) (*AppConfig, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	v := viper.New()

	// Set default values
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 300)
	v.SetDefault("jwt.expiration", 3600)
	v.SetDefault("apikey.enabled", false)
	v.SetDefault("apikey.header", "X-API-Key")

	// Configure Viper to read from config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
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
		// If we're searching for config files and none found, that's OK
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Enable environment variable override
	v.SetEnvPrefix("MOON")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind environment variables for keys we care about
	// This is needed because Unmarshal doesn't automatically pick up env vars
	v.BindEnv("server.port")
	v.BindEnv("server.host")
	v.BindEnv("database.connection_string")
	v.BindEnv("database.max_open_conns")
	v.BindEnv("database.max_idle_conns")
	v.BindEnv("database.conn_max_lifetime")
	v.BindEnv("jwt.secret")
	v.BindEnv("jwt.expiration")
	v.BindEnv("apikey.enabled")
	v.BindEnv("apikey.header")

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

	// Database connection string is optional (defaults to SQLite)
	// But if provided, it must not be empty
	if cfg.Database.ConnectionString == "" {
		// Set default SQLite connection
		cfg.Database.ConnectionString = "sqlite://moon.db"
	}

	// JWT secret is required for authentication
	// In production, this should be set via environment variable
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required (set MOON_JWT_SECRET)")
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
