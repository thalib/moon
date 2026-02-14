package auth

import (
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
)

// Schema SQL statements for authentication tables.
// These are used during database initialization.

// GetSchemaSQL returns the SQL statements to create auth tables for the given dialect.
func GetSchemaSQL(dialect database.DialectType) []string {
	switch dialect {
	case database.DialectPostgres:
		return getPostgresSchema()
	case database.DialectMySQL:
		return getMySQLSchema()
	default:
		return getSQLiteSchema()
	}
}

func getSQLiteSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS ` + constants.TableUsers + ` (
			pkid INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			can_write INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_login_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_users_id ON ` + constants.TableUsers + `(id)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_users_username ON ` + constants.TableUsers + `(username)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_users_email ON ` + constants.TableUsers + `(email)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableRefreshTokens + ` (
			pkid INTEGER PRIMARY KEY AUTOINCREMENT,
			user_pkid INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			last_used_at DATETIME NOT NULL,
			FOREIGN KEY (user_pkid) REFERENCES ` + constants.TableUsers + `(pkid) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_refresh_tokens_token_hash ON ` + constants.TableRefreshTokens + `(token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_refresh_tokens_user_pkid ON ` + constants.TableRefreshTokens + `(user_pkid)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_refresh_tokens_expires_at ON ` + constants.TableRefreshTokens + `(expires_at)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableAPIKeys + ` (
			pkid INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			key_hash TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL DEFAULT 'user',
			can_write INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			last_used_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_apikeys_id ON ` + constants.TableAPIKeys + `(id)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_apikeys_key_hash ON ` + constants.TableAPIKeys + `(key_hash)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableBlacklistedTokens + ` (
			pkid INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash TEXT NOT NULL UNIQUE,
			user_pkid INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (user_pkid) REFERENCES ` + constants.TableUsers + `(pkid) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_blacklisted_tokens_token_hash ON ` + constants.TableBlacklistedTokens + `(token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_blacklisted_tokens_expires_at ON ` + constants.TableBlacklistedTokens + `(expires_at)`,
	}
}

func getPostgresSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS ` + constants.TableUsers + ` (
			pkid BIGSERIAL PRIMARY KEY,
			id VARCHAR(26) NOT NULL UNIQUE,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'user',
			can_write BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_login_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_users_id ON ` + constants.TableUsers + `(id)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_users_username ON ` + constants.TableUsers + `(username)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_users_email ON ` + constants.TableUsers + `(email)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableRefreshTokens + ` (
			pkid BIGSERIAL PRIMARY KEY,
			user_pkid BIGINT NOT NULL REFERENCES ` + constants.TableUsers + `(pkid) ON DELETE CASCADE,
			token_hash VARCHAR(64) NOT NULL UNIQUE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL,
			last_used_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_refresh_tokens_token_hash ON ` + constants.TableRefreshTokens + `(token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_refresh_tokens_user_pkid ON ` + constants.TableRefreshTokens + `(user_pkid)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_refresh_tokens_expires_at ON ` + constants.TableRefreshTokens + `(expires_at)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableAPIKeys + ` (
			pkid BIGSERIAL PRIMARY KEY,
			id VARCHAR(26) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			key_hash VARCHAR(64) NOT NULL UNIQUE,
			role VARCHAR(50) NOT NULL DEFAULT 'user',
			can_write BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL,
			last_used_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_apikeys_id ON ` + constants.TableAPIKeys + `(id)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_apikeys_key_hash ON ` + constants.TableAPIKeys + `(key_hash)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableBlacklistedTokens + ` (
			pkid BIGSERIAL PRIMARY KEY,
			token_hash VARCHAR(64) NOT NULL UNIQUE,
			user_pkid BIGINT NOT NULL REFERENCES ` + constants.TableUsers + `(pkid) ON DELETE CASCADE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_blacklisted_tokens_token_hash ON ` + constants.TableBlacklistedTokens + `(token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_moon_blacklisted_tokens_expires_at ON ` + constants.TableBlacklistedTokens + `(expires_at)`,
	}
}

func getMySQLSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS ` + constants.TableUsers + ` (
			pkid BIGINT AUTO_INCREMENT PRIMARY KEY,
			id VARCHAR(26) NOT NULL UNIQUE,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'user',
			can_write BOOLEAN NOT NULL DEFAULT true,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_login_at DATETIME,
			INDEX idx_moon_users_id (id),
			INDEX idx_moon_users_username (username),
			INDEX idx_moon_users_email (email)
		)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableRefreshTokens + ` (
			pkid BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_pkid BIGINT NOT NULL,
			token_hash VARCHAR(64) NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			last_used_at DATETIME NOT NULL,
			INDEX idx_moon_refresh_tokens_token_hash (token_hash),
			INDEX idx_moon_refresh_tokens_user_pkid (user_pkid),
			INDEX idx_moon_refresh_tokens_expires_at (expires_at),
			FOREIGN KEY (user_pkid) REFERENCES ` + constants.TableUsers + `(pkid) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableAPIKeys + ` (
			pkid BIGINT AUTO_INCREMENT PRIMARY KEY,
			id VARCHAR(26) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			key_hash VARCHAR(64) NOT NULL UNIQUE,
			role VARCHAR(50) NOT NULL DEFAULT 'user',
			can_write BOOLEAN NOT NULL DEFAULT true,
			created_at DATETIME NOT NULL,
			last_used_at DATETIME,
			INDEX idx_moon_apikeys_id (id),
			INDEX idx_moon_apikeys_key_hash (key_hash)
		)`,

		`CREATE TABLE IF NOT EXISTS ` + constants.TableBlacklistedTokens + ` (
			pkid BIGINT AUTO_INCREMENT PRIMARY KEY,
			token_hash VARCHAR(64) NOT NULL UNIQUE,
			user_pkid BIGINT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			INDEX idx_moon_blacklisted_tokens_token_hash (token_hash),
			INDEX idx_moon_blacklisted_tokens_expires_at (expires_at),
			FOREIGN KEY (user_pkid) REFERENCES ` + constants.TableUsers + `(pkid) ON DELETE CASCADE
		)`,
	}
}
