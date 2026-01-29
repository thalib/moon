// Package database provides database abstraction and connection management.
// It supports multiple database dialects (PostgreSQL, MySQL, SQLite) with
// automatic dialect detection from connection strings.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// DialectType represents the type of database dialect
type DialectType string

const (
	DialectPostgres DialectType = "postgres"
	DialectMySQL    DialectType = "mysql"
	DialectSQLite   DialectType = "sqlite"
)

// Driver defines the interface for database operations
type Driver interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context) error

	// Close closes the database connection
	Close() error

	// Exec executes a query without returning rows
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	// Query executes a query that returns rows
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// QueryRow executes a query that returns at most one row
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row

	// BeginTx starts a new transaction
	BeginTx(ctx context.Context) (*sql.Tx, error)

	// Ping verifies the connection to the database is still alive
	Ping(ctx context.Context) error

	// Dialect returns the database dialect type
	Dialect() DialectType

	// DB returns the underlying *sql.DB instance
	DB() *sql.DB

	// ListTables returns a list of all user tables in the database
	ListTables(ctx context.Context) ([]string, error)

	// GetTableInfo retrieves detailed information about a table
	GetTableInfo(ctx context.Context, tableName string) (*TableInfo, error)

	// TableExists checks if a table exists in the database
	TableExists(ctx context.Context, tableName string) (bool, error)
}

// Config holds database connection configuration
type Config struct {
	ConnectionString string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
}

// baseDriver implements common functionality for all database drivers
type baseDriver struct {
	db      *sql.DB
	dialect DialectType
	dsn     string
	config  Config
}

// Connect establishes a connection to the database
func (d *baseDriver) Connect(ctx context.Context) error {
	var err error
	driverName := string(d.dialect)

	// For SQLite, use a different driver name
	if d.dialect == DialectSQLite {
		driverName = "sqlite"
	}

	d.db, err = sql.Open(driverName, d.dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	d.db.SetMaxOpenConns(d.config.MaxOpenConns)
	d.db.SetMaxIdleConns(d.config.MaxIdleConns)
	d.db.SetConnMaxLifetime(d.config.ConnMaxLifetime)

	// Verify connection
	if err := d.db.PingContext(ctx); err != nil {
		d.db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *baseDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Exec executes a query without returning rows
func (d *baseDriver) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows
func (d *baseDriver) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row
func (d *baseDriver) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a new transaction
func (d *baseDriver) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, nil)
}

// Ping verifies the connection to the database is still alive
func (d *baseDriver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// Dialect returns the database dialect type
func (d *baseDriver) Dialect() DialectType {
	return d.dialect
}

// DB returns the underlying *sql.DB instance
func (d *baseDriver) DB() *sql.DB {
	return d.db
}

// NewDriver creates a new database driver based on the connection string
func NewDriver(config Config) (Driver, error) {
	dialect, dsn, err := detectDialect(config.ConnectionString)
	if err != nil {
		return nil, err
	}

	driver := &baseDriver{
		dialect: dialect,
		dsn:     dsn,
		config:  config,
	}

	return driver, nil
}

// detectDialect detects the database dialect from the connection string
func detectDialect(connectionString string) (DialectType, string, error) {
	if connectionString == "" {
		return "", "", fmt.Errorf("connection string is empty")
	}

	lower := strings.ToLower(connectionString)

	// Check for URL-style connection strings
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return DialectPostgres, connectionString, nil
	}

	if strings.HasPrefix(lower, "mysql://") {
		// Convert mysql:// to MySQL DSN format
		dsn := strings.TrimPrefix(connectionString, "mysql://")
		return DialectMySQL, dsn, nil
	}

	if strings.HasPrefix(lower, "sqlite://") {
		// Extract file path from sqlite:// URL
		dsn := strings.TrimPrefix(connectionString, "sqlite://")
		
		// For in-memory databases, use shared cache mode to allow multiple connections
		// to access the same in-memory database
		if dsn == ":memory:" {
			dsn = "file::memory:?mode=memory&cache=shared"
		}
		
		return DialectSQLite, dsn, nil
	}

	// Check for standard MySQL DSN (user:password@tcp(host:port)/database)
	if strings.Contains(lower, "@tcp(") || strings.Contains(lower, "charset=") {
		return DialectMySQL, connectionString, nil
	}

	// Check for file-based connection strings (SQLite)
	if lower == ":memory:" || strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".sqlite") || strings.HasSuffix(lower, ".sqlite3") {
		return DialectSQLite, connectionString, nil
	}

	// Default to PostgreSQL for backward compatibility with standard DSN format
	if strings.Contains(lower, "host=") || strings.Contains(lower, "dbname=") {
		return DialectPostgres, connectionString, nil
	}

	return "", "", fmt.Errorf("unable to detect database dialect from connection string: %s", connectionString)
}
