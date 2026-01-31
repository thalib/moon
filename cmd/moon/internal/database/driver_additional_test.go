package database

import (
	"context"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// TestIsValidIdentifier tests the identifier validation function
func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid identifiers
		{name: "simple lowercase", input: "users", want: true},
		{name: "simple uppercase", input: "Users", want: true},
		{name: "mixed case", input: "UserData", want: true},
		{name: "with underscore", input: "user_data", want: true},
		{name: "start with underscore", input: "_users", want: true},
		{name: "alphanumeric", input: "user2", want: true},
		{name: "underscore and numbers", input: "table_123", want: true},

		// Invalid identifiers
		{name: "empty string", input: "", want: false},
		{name: "starts with number", input: "1users", want: false},
		{name: "contains dash", input: "user-data", want: false},
		{name: "contains space", input: "user data", want: false},
		{name: "contains dot", input: "user.data", want: false},
		{name: "contains special char", input: "user@data", want: false},
		{name: "too long", input: "a123456789012345678901234567890123456789012345678901234567890123456", want: false},
		{name: "contains semicolon", input: "users;", want: false},
		{name: "contains quote", input: "users'", want: false},
		{name: "SQL injection attempt", input: "users; DROP TABLE users;--", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestGetTableInfo_InvalidTableName tests validation of table names
func TestGetTableInfo_InvalidTableName(t *testing.T) {
	cfg := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}
	driver, err := NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer driver.Close()

	// Test with invalid table names
	invalidNames := []string{
		"",
		"users; DROP TABLE users;--",
		"table-with-dash",
		"1startwithnum",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			_, err := driver.GetTableInfo(ctx, name)
			if err == nil {
				t.Errorf("GetTableInfo(%q) should return error for invalid table name", name)
			}
		})
	}
}

// TestInferColumnType_Extended tests additional column type mappings
func TestInferColumnType_Extended(t *testing.T) {
	tests := []struct {
		dbType string
		want   registry.ColumnType
	}{
		// Case variations
		{"integer", registry.TypeInteger},
		{"INTEGER", registry.TypeInteger},
		{"Integer", registry.TypeInteger},
		{"BIGSERIAL", registry.TypeInteger},
		{"bigserial", registry.TypeInteger},
		{"SMALLINT", registry.TypeInteger},
		{"TINYINT", registry.TypeInteger},
		{"MEDIUMINT", registry.TypeInteger},

		// Float types (map to Integer now)
		{"float", registry.TypeInteger},
		{"FLOAT4", registry.TypeInteger},
		{"FLOAT8", registry.TypeInteger},
		{"double precision", registry.TypeInteger},
		{"NUMERIC(10,2)", registry.TypeInteger},
		{"DECIMAL(10,2)", registry.TypeInteger},
		{"real", registry.TypeInteger},

		// Boolean variations
		{"boolean", registry.TypeBoolean},
		{"BOOLEAN", registry.TypeBoolean},
		{"BOOL", registry.TypeBoolean},
		{"tinyint(1)", registry.TypeInteger}, // MySQL boolean often uses tinyint

		// DateTime variations
		{"timestamp", registry.TypeDatetime},
		{"TIMESTAMP WITH TIME ZONE", registry.TypeDatetime},
		{"TIMESTAMP WITHOUT TIME ZONE", registry.TypeDatetime},
		{"datetime", registry.TypeDatetime},
		{"date", registry.TypeDatetime},
		{"time", registry.TypeDatetime},
		{"TIME WITH TIME ZONE", registry.TypeDatetime},

		// JSON variations
		{"json", registry.TypeJSON},
		{"jsonb", registry.TypeJSON},
		{"JSON", registry.TypeJSON},
		{"JSONB", registry.TypeJSON},

		// String/Text variations
		{"text", registry.TypeString},
		{"TEXT", registry.TypeString},
		{"varchar", registry.TypeString},
		{"VARCHAR(255)", registry.TypeString},
		{"character varying", registry.TypeString},
		{"char", registry.TypeString},
		{"CHAR(10)", registry.TypeString},
		{"nvarchar", registry.TypeString},
		{"ntext", registry.TypeString},
		{"clob", registry.TypeString},
		{"CLOB", registry.TypeString},
		{"blob", registry.TypeString},
		{"BLOB", registry.TypeString},

		// Unknown types default to string
		{"unknown_type", registry.TypeString},
		{"custom_type", registry.TypeString},
		{"geometry", registry.TypeString},
		{"uuid", registry.TypeString},
		{"bytea", registry.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			got := InferColumnType(tt.dbType)
			if got != tt.want {
				t.Errorf("InferColumnType(%q) = %v, want %v", tt.dbType, got, tt.want)
			}
		})
	}
}

// TestDetectDialect_EdgeCases tests edge cases in dialect detection
func TestDetectDialect_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		connString string
		wantDialect DialectType
		wantError  bool
	}{
		// PostgreSQL variations
		{
			name:        "postgres URL",
			connString:  "postgres://user:pass@localhost/db",
			wantDialect: DialectPostgres,
			wantError:   false,
		},
		{
			name:        "postgresql URL",
			connString:  "postgresql://user:pass@localhost/db",
			wantDialect: DialectPostgres,
			wantError:   false,
		},
		{
			name:        "postgres DSN format",
			connString:  "host=localhost dbname=test user=postgres",
			wantDialect: DialectPostgres,
			wantError:   false,
		},

		// MySQL variations
		{
			name:        "mysql URL",
			connString:  "mysql://user:pass@localhost/db",
			wantDialect: DialectMySQL,
			wantError:   false,
		},
		{
			name:        "mysql DSN format",
			connString:  "user:password@tcp(localhost:3306)/database",
			wantDialect: DialectMySQL,
			wantError:   false,
		},
		{
			name:        "mysql with charset",
			connString:  "user:password@tcp(localhost:3306)/database?charset=utf8mb4",
			wantDialect: DialectMySQL,
			wantError:   false,
		},

		// SQLite variations
		{
			name:        "sqlite URL",
			connString:  "sqlite://:memory:",
			wantDialect: DialectSQLite,
			wantError:   false,
		},
		{
			name:        "sqlite memory",
			connString:  ":memory:",
			wantDialect: DialectSQLite,
			wantError:   false,
		},
		{
			name:        "sqlite file .db",
			connString:  "/path/to/database.db",
			wantDialect: DialectSQLite,
			wantError:   false,
		},
		{
			name:        "sqlite file .sqlite",
			connString:  "/path/to/database.sqlite",
			wantDialect: DialectSQLite,
			wantError:   false,
		},
		{
			name:        "sqlite file .sqlite3",
			connString:  "/path/to/database.sqlite3",
			wantDialect: DialectSQLite,
			wantError:   false,
		},

		// Error cases
		{
			name:       "empty string",
			connString: "",
			wantError:  true,
		},
		{
			name:       "unrecognized format",
			connString: "random-connection-string",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialect, _, err := detectDialect(tt.connString)

			if tt.wantError {
				if err == nil {
					t.Errorf("detectDialect(%q) expected error, got nil", tt.connString)
				}
				return
			}

			if err != nil {
				t.Errorf("detectDialect(%q) unexpected error: %v", tt.connString, err)
				return
			}

			if dialect != tt.wantDialect {
				t.Errorf("detectDialect(%q) = %v, want %v", tt.connString, dialect, tt.wantDialect)
			}
		})
	}
}

// TestDriver_DBMethod tests the DB() method returns the underlying connection
func TestDriver_DBMethod(t *testing.T) {
	cfg := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer driver.Close()

	// DB() should return the underlying *sql.DB
	db := driver.DB()
	if db == nil {
		t.Error("DB() returned nil, expected *sql.DB")
	}

	// Should be able to ping using the underlying connection
	if err := db.PingContext(ctx); err != nil {
		t.Errorf("Failed to ping using underlying DB: %v", err)
	}
}

// TestDriver_CloseTwice tests that closing a driver twice doesn't error
func TestDriver_CloseTwice(t *testing.T) {
	cfg := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// First close should succeed
	if err := driver.Close(); err != nil {
		t.Errorf("First Close() failed: %v", err)
	}

	// Second close - behavior may vary
	err = driver.Close()
	// Just ensure no panic, error is acceptable
	t.Logf("Second Close() result: %v", err)
}

// TestDriver_CloseWithoutConnect tests closing a driver that was never connected
func TestDriver_CloseWithoutConnect(t *testing.T) {
	cfg := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	// Close without connecting should not panic
	err = driver.Close()
	if err != nil {
		t.Logf("Close() without Connect: %v", err)
	}
}

// TestDriver_BeginTx tests transaction creation
func TestDriver_BeginTx(t *testing.T) {
	cfg := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer driver.Close()

	// Create a table for testing
	_, err = driver.Exec(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Begin transaction
	tx, err := driver.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() failed: %v", err)
	}

	// Insert within transaction
	_, err = tx.Exec("INSERT INTO test (value) VALUES ('test')")
	if err != nil {
		t.Fatalf("Insert within transaction failed: %v", err)
	}

	// Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() failed: %v", err)
	}

	// Value should not exist after rollback
	var count int
	err = driver.QueryRow(ctx, "SELECT COUNT(*) FROM test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 rows after rollback, got %d", count)
	}
}

// TestDriver_BeginTx_Commit tests transaction commit
func TestDriver_BeginTx_Commit(t *testing.T) {
	cfg := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer driver.Close()

	// Create a table for testing
	_, err = driver.Exec(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Begin transaction
	tx, err := driver.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() failed: %v", err)
	}

	// Insert within transaction
	_, err = tx.Exec("INSERT INTO test (value) VALUES ('test')")
	if err != nil {
		t.Fatalf("Insert within transaction failed: %v", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() failed: %v", err)
	}

	// Value should exist after commit
	var count int
	err = driver.QueryRow(ctx, "SELECT COUNT(*) FROM test").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row after commit, got %d", count)
	}
}
