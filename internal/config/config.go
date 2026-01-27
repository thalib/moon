package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Defaults contains all default configuration values
// centralized in one place to avoid hardcoded literals
var Defaults = struct {
	Server struct {
		Port int
		Host string
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
		Expiry int
	}
	APIKey struct {
		Enabled bool
		Header  string
	}
	ConfigPath string
}{
	Server: struct {
		Port int
		Host string
	}{
		Port: 6006,
		Host: "0.0.0.0",
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
		Expiry int
	}{
		Expiry: 3600,
	},
	APIKey: struct {
		Enabled bool
		Header  string
	}{
		Enabled: false,
		Header:  "X-API-KEY",
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
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
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
	Secret string `mapstructure:"secret"`
	Expiry int    `mapstructure:"expiry"` // in seconds
}

// APIKeyConfig holds API key configuration.
type APIKeyConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Header  string `mapstructure:"header"`
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
	v.SetDefault("database.connection", Defaults.Database.Connection)
	v.SetDefault("database.database", Defaults.Database.Database)
	v.SetDefault("database.user", Defaults.Database.User)
	v.SetDefault("database.password", Defaults.Database.Password)
	v.SetDefault("database.host", Defaults.Database.Host)
	v.SetDefault("logging.path", Defaults.Logging.Path)
	v.SetDefault("jwt.expiry", Defaults.JWT.Expiry)
	v.SetDefault("apikey.enabled", Defaults.APIKey.Enabled)
	v.SetDefault("apikey.header", Defaults.APIKey.Header)

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

	// Apply default database values if not provided
	if cfg.Database.Connection == "" {
		cfg.Database.Connection = Defaults.Database.Connection
	}
	if cfg.Database.Database == "" {
		cfg.Database.Database = Defaults.Database.Database
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
