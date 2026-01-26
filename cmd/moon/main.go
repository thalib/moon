package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/thalib/moon/internal/config"
	"github.com/thalib/moon/internal/database"
	"github.com/thalib/moon/internal/registry"
	"github.com/thalib/moon/internal/server"
)

func main() {
	fmt.Println("Moon - Dynamic Headless Engine")

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Server will start on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database: %s\n", cfg.Database.ConnectionString)

	// Initialize database driver
	dbConfig := database.Config{
		ConnectionString: cfg.Database.ConnectionString,
		MaxOpenConns:     cfg.Database.MaxOpenConns,
		MaxIdleConns:     cfg.Database.MaxIdleConns,
		ConnMaxLifetime:  time.Duration(cfg.Database.ConnMaxLifetime) * time.Second,
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
