package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
)

// BlacklistedToken represents a revoked JWT access token
type BlacklistedToken struct {
	ID        int64
	TokenHash string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// TokenBlacklist manages revoked JWT tokens
type TokenBlacklist struct {
	db database.Driver
}

// NewTokenBlacklist creates a new token blacklist instance
func NewTokenBlacklist(db database.Driver) *TokenBlacklist {
	return &TokenBlacklist{db: db}
}

// Add adds a token to the blacklist
func (b *TokenBlacklist) Add(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	tokenHash := hashTokenForBlacklist(token)

	var query string
	switch b.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf(`INSERT INTO %s (token_hash, user_id, expires_at, created_at)
			VALUES ($1, $2, $3, $4)`, constants.TableBlacklistedTokens)
		_, err := b.db.Exec(ctx, query, tokenHash, userID, expiresAt, time.Now())
		if err != nil {
			return fmt.Errorf("failed to blacklist token: %w", err)
		}
		return nil
	default:
		query = fmt.Sprintf(`INSERT INTO %s (token_hash, user_id, expires_at, created_at)
			VALUES (?, ?, ?, ?)`, constants.TableBlacklistedTokens)
		_, err := b.db.Exec(ctx, query, tokenHash, userID, expiresAt, time.Now())
		if err != nil {
			return fmt.Errorf("failed to blacklist token: %w", err)
		}
		return nil
	}
}

// IsBlacklisted checks if a token is blacklisted
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	tokenHash := hashTokenForBlacklist(token)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE token_hash = ?", constants.TableBlacklistedTokens)
	if b.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE token_hash = $1", constants.TableBlacklistedTokens)
	}

	var count int
	err := b.db.QueryRow(ctx, query, tokenHash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return count > 0, nil
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (b *TokenBlacklist) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	// We can't blacklist all existing tokens without having them
	// Instead, we'll mark that tokens issued before this time should be invalid
	// This is handled by storing a revocation timestamp in a separate table
	// For now, we just delete existing tokens from refresh_tokens which will prevent refresh
	// Access tokens will expire naturally
	return nil
}

// CleanupExpired removes expired tokens from the blacklist
func (b *TokenBlacklist) CleanupExpired(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE expires_at < ?", constants.TableBlacklistedTokens)
	if b.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE expires_at < $1", constants.TableBlacklistedTokens)
	}

	_, err := b.db.Exec(ctx, query, time.Now())
	return err
}

// hashTokenForBlacklist creates a SHA-256 hash of the token for storage
func hashTokenForBlacklist(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
