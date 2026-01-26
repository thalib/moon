package database

import (
	"context"
	"testing"
	"time"
)

func TestDetectDialect_Postgres(t *testing.T) {
	tests := []struct {
		name           string
		connStr        string
		expectedDialect DialectType
	}{
		{
			name:            "postgres:// URL",
			connStr:         "postgres://user:pass@localhost:5432/dbname",
			expectedDialect: DialectPostgres,
		},
		{
			name:            "postgresql:// URL",
			connStr:         "postgresql://user:pass@localhost/db",
			expectedDialect: DialectPostgres,
		},
		{
			name:            "PostgreSQL DSN",
			connStr:         "host=localhost port=5432 user=postgres dbname=test sslmode=disable",
			expectedDialect: DialectPostgres,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialect, _, err := detectDialect(tt.connStr)
			if err != nil {
				t.Fatalf("detectDialect() error = %v", err)
			}
			if dialect != tt.expectedDialect {
				t.Errorf("Expected dialect %v, got %v", tt.expectedDialect, dialect)
			}
		})
	}
}

func TestDetectDialect_MySQL(t *testing.T) {
	tests := []struct {
		name           string
		connStr        string
		expectedDialect DialectType
	}{
		{
			name:            "mysql:// URL",
			connStr:         "mysql://user:pass@localhost:3306/dbname",
			expectedDialect: DialectMySQL,
		},
		{
			name:            "MySQL DSN with tcp",
			connStr:         "user:password@tcp(localhost:3306)/dbname",
			expectedDialect: DialectMySQL,
		},
		{
			name:            "MySQL DSN with charset",
			connStr:         "user:pass@/dbname?charset=utf8mb4",
			expectedDialect: DialectMySQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialect, _, err := detectDialect(tt.connStr)
			if err != nil {
				t.Fatalf("detectDialect() error = %v", err)
			}
			if dialect != tt.expectedDialect {
				t.Errorf("Expected dialect %v, got %v", tt.expectedDialect, dialect)
			}
		})
	}
}

func TestDetectDialect_SQLite(t *testing.T) {
	tests := []struct {
		name           string
		connStr        string
		expectedDialect DialectType
	}{
		{
			name:            "sqlite:// URL",
			connStr:         "sqlite://test.db",
			expectedDialect: DialectSQLite,
		},
		{
			name:            ".db file",
			connStr:         "test.db",
			expectedDialect: DialectSQLite,
		},
		{
			name:            ".sqlite file",
			connStr:         "data.sqlite",
			expectedDialect: DialectSQLite,
		},
		{
			name:            ".sqlite3 file",
			connStr:         "moon.sqlite3",
			expectedDialect: DialectSQLite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialect, _, err := detectDialect(tt.connStr)
			if err != nil {
				t.Fatalf("detectDialect() error = %v", err)
			}
			if dialect != tt.expectedDialect {
				t.Errorf("Expected dialect %v, got %v", tt.expectedDialect, dialect)
			}
		})
	}
}

func TestDetectDialect_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
	}{
		{
			name:    "empty string",
			connStr: "",
		},
		{
			name:    "unsupported protocol",
			connStr: "oracle://user:pass@localhost/db",
		},
		{
			name:    "random text",
			connStr: "invalid connection string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := detectDialect(tt.connStr)
			if err == nil {
				t.Error("Expected error for invalid connection string, got nil")
			}
		})
	}
}

func TestNewDriver(t *testing.T) {
	config := Config{
		ConnectionString: "sqlite://test.db",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(config)
	if err != nil {
		t.Fatalf("NewDriver() error = %v", err)
	}

	if driver == nil {
		t.Fatal("Expected non-nil driver")
	}

	if driver.Dialect() != DialectSQLite {
		t.Errorf("Expected dialect %v, got %v", DialectSQLite, driver.Dialect())
	}
}

func TestNewDriver_InvalidConnection(t *testing.T) {
	config := Config{
		ConnectionString: "",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	_, err := NewDriver(config)
	if err == nil {
		t.Error("Expected error for empty connection string, got nil")
	}
}

func TestDriver_SQLite_Operations(t *testing.T) {
	// Use in-memory SQLite for testing
	config := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(config)
	if err != nil {
		t.Fatalf("NewDriver() error = %v", err)
	}

	ctx := context.Background()

	// Test Connect
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer driver.Close()

	// Test Ping
	if err := driver.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}

	// Test Exec - Create table
	createTable := `CREATE TABLE test_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE
	)`
	_, err = driver.Exec(ctx, createTable)
	if err != nil {
		t.Fatalf("Exec(CREATE TABLE) error = %v", err)
	}

	// Test Exec - Insert
	_, err = driver.Exec(ctx, "INSERT INTO test_users (name, email) VALUES (?, ?)", "John Doe", "john@example.com")
	if err != nil {
		t.Fatalf("Exec(INSERT) error = %v", err)
	}

	// Test QueryRow
	var name, email string
	row := driver.QueryRow(ctx, "SELECT name, email FROM test_users WHERE id = ?", 1)
	if err := row.Scan(&name, &email); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}

	if name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", name)
	}

	if email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", email)
	}

	// Test Query
	rows, err := driver.Query(ctx, "SELECT name, email FROM test_users")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var n, e string
		if err := rows.Scan(&n, &e); err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestDriver_Transaction(t *testing.T) {
	config := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(config)
	if err != nil {
		t.Fatalf("NewDriver() error = %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer driver.Close()

	// Create table
	_, err = driver.Exec(ctx, "CREATE TABLE test_tx (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("Create table error = %v", err)
	}

	// Test successful transaction
	tx, err := driver.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO test_tx (value) VALUES (?)", "test1")
	if err != nil {
		t.Fatalf("Transaction Exec() error = %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// Verify data was committed
	var value string
	row := driver.QueryRow(ctx, "SELECT value FROM test_tx WHERE id = 1")
	if err := row.Scan(&value); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}

	if value != "test1" {
		t.Errorf("Expected value 'test1', got '%s'", value)
	}

	// Test rollback
	tx, err = driver.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO test_tx (value) VALUES (?)", "test2")
	if err != nil {
		t.Fatalf("Transaction Exec() error = %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	// Verify data was not committed
	rows, err := driver.Query(ctx, "SELECT COUNT(*) FROM test_tx")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 row after rollback, got %d", count)
	}
}

func TestDriver_Close(t *testing.T) {
	config := Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Minute * 5,
	}

	driver, err := NewDriver(config)
	if err != nil {
		t.Fatalf("NewDriver() error = %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Close should not return error
	if err := driver.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Calling Close again should be safe
	if err := driver.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}
