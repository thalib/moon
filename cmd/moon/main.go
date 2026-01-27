package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thalib/moon/internal/config"
	"github.com/thalib/moon/internal/daemon"
	"github.com/thalib/moon/internal/database"
	"github.com/thalib/moon/internal/logging"
	"github.com/thalib/moon/internal/preflight"
	"github.com/thalib/moon/internal/registry"
	"github.com/thalib/moon/internal/server"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "path to configuration file (default: /etc/moon.conf)")
	daemonMode := flag.Bool("daemon", false, "run in daemon mode (background)")
	daemonShort := flag.Bool("d", false, "run in daemon mode (background) - shorthand")
	flag.Parse()

	// Check if daemon mode is enabled (either flag)
	isDaemon := *daemonMode || *daemonShort

	fmt.Println("Moon - Dynamic Headless Engine")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Run preflight checks before any other initialization
	fmt.Println("Running preflight checks...")
	if err := runPreflightChecks(cfg, isDaemon); err != nil {
		fmt.Fprintf(os.Stderr, "Preflight checks failed: %v\n", err)
		os.Exit(1)
	}

	// Handle daemon mode
	if isDaemon {
		fmt.Println("Starting in daemon mode...")

		// Daemonize the process
		daemonCfg := daemon.DefaultConfig()
		if err := daemon.Daemonize(daemonCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to daemonize: %v\n", err)
			os.Exit(1)
		}

		// Write PID file (after daemonization, in child process)
		if err := daemon.WritePIDFile(daemonCfg.PIDFile); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write PID file: %v\n", err)
			os.Exit(1)
		}

		// Setup cleanup on exit
		defer daemon.RemovePIDFile(daemonCfg.PIDFile)

		// Initialize file-based logging for daemon mode
		logFile := filepath.Join(cfg.Logging.Path, "main.log")
		logging.Init(logging.LoggerConfig{
			Level:       logging.LevelInfo,
			Format:      "json",
			FilePath:    logFile,
			ServiceName: "moon",
		})

		logging.Info("Moon daemon started")
		logging.Infof("PID file: %s", daemonCfg.PIDFile)
		logging.Infof("Log file: %s", logFile)
	} else {
		// Console mode - log to stdout
		logging.Init(logging.LoggerConfig{
			Level:       logging.LevelInfo,
			Format:      "console",
			ServiceName: "moon",
		})
	}

	// Log configuration summary
	logConfigSummary(cfg)

	fmt.Printf("Server will start on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database: %s (%s)\n", cfg.Database.Connection, cfg.Database.Database)
	if isDaemon {
		fmt.Printf("Log file: %s/main.log\n", cfg.Logging.Path)
	}

	// Initialize database driver
	dbConfig := database.Config{
		ConnectionString: buildConnectionString(cfg.Database),
	}

	driver, err := database.NewDriver(dbConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create database driver: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close()

	fmt.Printf("Connected to %s database\n", driver.Dialect())

	// Initialize schema registry
	reg := registry.NewSchemaRegistry()

	// Create and start HTTP server
	srv := server.New(cfg, driver, reg)

	fmt.Println("Starting HTTP server...")
	if err := srv.Run(); err != nil {
		if isDaemon {
			logging.Errorf("Server error: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}

	if isDaemon {
		logging.Info("Server stopped gracefully")
	}
	fmt.Println("Server stopped gracefully")
}

// buildConnectionString creates a database connection string from DatabaseConfig
func buildConnectionString(db config.DatabaseConfig) string {
	switch db.Connection {
	case "sqlite":
		return fmt.Sprintf("sqlite://%s", db.Database)
	case "postgres":
		if db.User != "" && db.Password != "" {
			return fmt.Sprintf("postgres://%s:%s@%s/%s", db.User, db.Password, db.Host, db.Database)
		}
		return fmt.Sprintf("postgres://%s/%s", db.Host, db.Database)
	case "mysql":
		if db.User != "" && db.Password != "" {
			return fmt.Sprintf("mysql://%s:%s@%s/%s", db.User, db.Password, db.Host, db.Database)
		}
		return fmt.Sprintf("mysql://%s/%s", db.Host, db.Database)
	default:
		// Default to SQLite
		return fmt.Sprintf("sqlite://%s", db.Database)
	}
}

// runPreflightChecks validates and creates required files and directories
func runPreflightChecks(cfg *config.AppConfig, isDaemon bool) error {
	var checks []preflight.FileCheck

	// Check logging directory
	checks = append(checks, preflight.FileCheck{
		Path:      cfg.Logging.Path,
		IsDir:     true,
		Required:  true,
		FailFatal: true,
	})

	// For SQLite, check database file parent directory and create if needed
	// Database path is already normalized to absolute in config.validate()
	if cfg.Database.Connection == config.Defaults.Database.Connection {
		dbDir := filepath.Dir(cfg.Database.Database)
		checks = append(checks, preflight.FileCheck{
			Path:      dbDir,
			IsDir:     true,
			Required:  true,
			FailFatal: true,
		})
	}

	// Run validation
	results, err := preflight.ValidateAndCreate(checks)
	if err != nil {
		return err
	}

	// Log results
	for _, result := range results {
		if result.Created {
			fmt.Printf("✓ Created: %s\n", result.Path)
		} else if result.Exists {
			fmt.Printf("✓ Verified: %s\n", result.Path)
		}
		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "✗ Error with %s: %v\n", result.Path, result.Error)
		}
	}

	// If daemon mode, truncate the log file to start fresh
	if isDaemon {
		logFile := filepath.Join(cfg.Logging.Path, "main.log")
		fmt.Printf("Truncating log file: %s\n", logFile)
		if err := preflight.CreateOrTruncateFile(logFile); err != nil {
			return fmt.Errorf("failed to truncate log file: %w", err)
		}
	}

	return nil
}

// logConfigSummary logs the loaded configuration for debugging
func logConfigSummary(cfg *config.AppConfig) {
	logging.Info("=== Configuration Summary ===")
	logging.Infof("Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
	logging.Infof("Database Type: %s", cfg.Database.Connection)
	logging.Infof("Database: %s", cfg.Database.Database)
	if cfg.Database.User != "" {
		logging.Infof("Database User: %s", cfg.Database.User)
	}
	if cfg.Database.Host != "" && cfg.Database.Connection != "sqlite" {
		logging.Infof("Database Host: %s", cfg.Database.Host)
	}
	logging.Infof("Logging Path: %s", cfg.Logging.Path)
	logging.Infof("JWT Expiry: %d seconds", cfg.JWT.Expiry)
	logging.Infof("API Key Enabled: %v", cfg.APIKey.Enabled)
	if cfg.APIKey.Enabled {
		logging.Infof("API Key Header: %s", cfg.APIKey.Header)
	}
	logging.Info("============================")
}

