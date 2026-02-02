package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
)

// RefreshTokenRepository provides database operations for refresh tokens.
type RefreshTokenRepository struct {
	db database.Driver
}

// NewRefreshTokenRepository creates a new refresh token repository.
func NewRefreshTokenRepository(db database.Driver) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// HashToken hashes a refresh token using SHA-256.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Create creates a new refresh token in the database.
func (r *RefreshTokenRepository) Create(ctx context.Context, token *RefreshToken) error {
	token.CreatedAt = time.Now()
	token.LastUsedAt = time.Now()

	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf(`INSERT INTO %s (user_id, token_hash, expires_at, created_at, last_used_at)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`, constants.TableRefreshTokens)
		err := r.db.QueryRow(ctx, query,
			token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt, token.LastUsedAt,
		).Scan(&token.ID)
		if err != nil {
			return fmt.Errorf("failed to create refresh token: %w", err)
		}
		return nil
	default:
		query = fmt.Sprintf(`INSERT INTO %s (user_id, token_hash, expires_at, created_at, last_used_at)
			VALUES (?, ?, ?, ?, ?)`, constants.TableRefreshTokens)
		result, err := r.db.Exec(ctx, query,
			token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt, token.LastUsedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create refresh token: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get refresh token ID: %w", err)
		}
		token.ID = id
		return nil
	}
}

// GetByHash retrieves a refresh token by its hash.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	query := fmt.Sprintf("SELECT id, user_id, token_hash, expires_at, created_at, last_used_at FROM %s WHERE token_hash = ?", constants.TableRefreshTokens)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT id, user_id, token_hash, expires_at, created_at, last_used_at FROM %s WHERE token_hash = $1", constants.TableRefreshTokens)
	}

	token := &RefreshToken{}
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.CreatedAt, &token.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return token, nil
}

// UpdateLastUsed updates the last used time for a refresh token.
func (r *RefreshTokenRepository) UpdateLastUsed(ctx context.Context, id int64) error {
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf("UPDATE %s SET last_used_at = $1 WHERE id = $2", constants.TableRefreshTokens)
	default:
		query = fmt.Sprintf("UPDATE %s SET last_used_at = ? WHERE id = ?", constants.TableRefreshTokens)
	}

	_, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}
	return nil
}

// Delete deletes a refresh token from the database.
func (r *RefreshTokenRepository) Delete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", constants.TableRefreshTokens)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", constants.TableRefreshTokens)
	}

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// DeleteByHash deletes a refresh token by its hash.
func (r *RefreshTokenRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE token_hash = ?", constants.TableRefreshTokens)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE token_hash = $1", constants.TableRefreshTokens)
	}

	_, err := r.db.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all refresh tokens for a user.
func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE user_id = ?", constants.TableRefreshTokens)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE user_id = $1", constants.TableRefreshTokens)
	}

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user tokens: %w", err)
	}
	return nil
}

// DeleteAllByUserID is an alias for DeleteByUserID for backwards compatibility
func (r *RefreshTokenRepository) DeleteAllByUserID(ctx context.Context, userID int64) error {
	return r.DeleteByUserID(ctx, userID)
}

// DeleteExpired deletes all expired refresh tokens.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE expires_at < ?", constants.TableRefreshTokens)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE expires_at < $1", constants.TableRefreshTokens)
	}

	result, err := r.db.Exec(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return count, nil
}
