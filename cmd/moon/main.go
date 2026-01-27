package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/thalib/moon/internal/config"
	"github.com/thalib/moon/internal/database"
	"github.com/thalib/moon/internal/registry"
	"github.com/thalib/moon/internal/server"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "path to configuration file (default: /etc/moon.conf)")
	flag.Parse()

	fmt.Println("Moon - Dynamic Headless Engine")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Server will start on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database: %s (%s)\n", cfg.Database.Connection, cfg.Database.Database)

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
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
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
