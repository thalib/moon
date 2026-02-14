package auth

import (
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/database"
)

func TestGetSchemaSQL_SQLite(t *testing.T) {
	stmts := GetSchemaSQL(database.DialectSQLite)

	if len(stmts) == 0 {
		t.Fatal("GetSchemaSQL(SQLite) returned empty statements")
	}

	// Check for expected tables
	var hasUsers, hasRefreshTokens, hasAPIKeys bool
	for _, stmt := range stmts {
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_users") {
			hasUsers = true
		}
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_refresh_tokens") {
			hasRefreshTokens = true
		}
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_apikeys") {
			hasAPIKeys = true
		}
	}

	if !hasUsers {
		t.Error("Missing users table in SQLite schema")
	}
	if !hasRefreshTokens {
		t.Error("Missing refresh_tokens table in SQLite schema")
	}
	if !hasAPIKeys {
		t.Error("Missing apikeys table in SQLite schema")
	}
}

func TestGetSchemaSQL_Postgres(t *testing.T) {
	stmts := GetSchemaSQL(database.DialectPostgres)

	if len(stmts) == 0 {
		t.Fatal("GetSchemaSQL(Postgres) returned empty statements")
	}

	// Check for expected tables
	var hasUsers, hasRefreshTokens, hasAPIKeys bool
	for _, stmt := range stmts {
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_users") {
			hasUsers = true
		}
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_refresh_tokens") {
			hasRefreshTokens = true
		}
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_apikeys") {
			hasAPIKeys = true
		}
	}

	if !hasUsers {
		t.Error("Missing users table in Postgres schema")
	}
	if !hasRefreshTokens {
		t.Error("Missing refresh_tokens table in Postgres schema")
	}
	if !hasAPIKeys {
		t.Error("Missing apikeys table in Postgres schema")
	}

	// Check for Postgres-specific types
	found := false
	for _, stmt := range stmts {
		if contains(stmt, "BIGSERIAL") || contains(stmt, "TIMESTAMP") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Postgres schema should use BIGSERIAL or TIMESTAMP types")
	}
}

func TestGetSchemaSQL_MySQL(t *testing.T) {
	stmts := GetSchemaSQL(database.DialectMySQL)

	if len(stmts) == 0 {
		t.Fatal("GetSchemaSQL(MySQL) returned empty statements")
	}

	// Check for expected tables
	var hasUsers, hasRefreshTokens, hasAPIKeys bool
	for _, stmt := range stmts {
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_users") {
			hasUsers = true
		}
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_refresh_tokens") {
			hasRefreshTokens = true
		}
		if contains(stmt, "CREATE TABLE IF NOT EXISTS moon_apikeys") {
			hasAPIKeys = true
		}
	}

	if !hasUsers {
		t.Error("Missing users table in MySQL schema")
	}
	if !hasRefreshTokens {
		t.Error("Missing refresh_tokens table in MySQL schema")
	}
	if !hasAPIKeys {
		t.Error("Missing apikeys table in MySQL schema")
	}

	// Check for MySQL-specific syntax
	found := false
	for _, stmt := range stmts {
		if contains(stmt, "AUTO_INCREMENT") {
			found = true
			break
		}
	}
	if !found {
		t.Error("MySQL schema should use AUTO_INCREMENT")
	}
}

func TestGetSchemaSQL_ForeignKeys(t *testing.T) {
	dialects := []database.DialectType{
		database.DialectSQLite,
		database.DialectPostgres,
		database.DialectMySQL,
	}

	for _, dialect := range dialects {
		stmts := GetSchemaSQL(dialect)

		found := false
		for _, stmt := range stmts {
			if contains(stmt, "FOREIGN KEY") || contains(stmt, "REFERENCES moon_users(pkid)") {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Schema for %s should have foreign key constraint", dialect)
		}
	}
}

func TestGetSchemaSQL_Indexes(t *testing.T) {
	dialects := []database.DialectType{
		database.DialectSQLite,
		database.DialectPostgres,
	}

	for _, dialect := range dialects {
		stmts := GetSchemaSQL(dialect)

		indexCount := 0
		for _, stmt := range stmts {
			if contains(stmt, "CREATE INDEX") {
				indexCount++
			}
		}

		if indexCount < 5 {
			t.Errorf("Schema for %s should have at least 5 indexes, got %d", dialect, indexCount)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
