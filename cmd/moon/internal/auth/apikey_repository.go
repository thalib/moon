package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
	moonulid "github.com/thalib/moon/cmd/moon/internal/ulid"
)

// base62Charset is used for generating API keys.
const base62Charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// APIKeyRepository provides database operations for API keys.
type APIKeyRepository struct {
	db database.Driver
}

// NewAPIKeyRepository creates a new API key repository.
func NewAPIKeyRepository(db database.Driver) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// GenerateAPIKey generates a new API key with the moon_live_ prefix.
// Returns the raw key (to be shown to user once) and the hash (to be stored).
func GenerateAPIKey() (rawKey string, keyHash string, err error) {
	// Generate 64 random bytes
	randomBytes := make([]byte, APIKeyLength)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to base62
	chars := make([]byte, APIKeyLength)
	for i, b := range randomBytes {
		chars[i] = base62Charset[int(b)%len(base62Charset)]
	}

	rawKey = APIKeyPrefix + string(chars)
	keyHash = HashAPIKey(rawKey)
	return rawKey, keyHash, nil
}

// HashAPIKey hashes an API key using SHA-256.
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// Create creates a new API key in the database.
func (r *APIKeyRepository) Create(ctx context.Context, apiKey *APIKey) error {
	apiKey.ULID = moonulid.Generate()
	apiKey.CreatedAt = time.Now()

	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = `INSERT INTO apikeys (ulid, name, description, key_hash, role, can_write, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
		err := r.db.QueryRow(ctx, query,
			apiKey.ULID, apiKey.Name, apiKey.Description, apiKey.KeyHash,
			apiKey.Role, apiKey.CanWrite, apiKey.CreatedAt,
		).Scan(&apiKey.ID)
		return err
	default:
		query = `INSERT INTO apikeys (ulid, name, description, key_hash, role, can_write, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
		result, err := r.db.Exec(ctx, query,
			apiKey.ULID, apiKey.Name, apiKey.Description, apiKey.KeyHash,
			apiKey.Role, apiKey.CanWrite, apiKey.CreatedAt,
		)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		apiKey.ID = id
		return nil
	}
}

// GetByID retrieves an API key by internal ID.
func (r *APIKeyRepository) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	query := "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys WHERE id = ?"
	if r.db.Dialect() == database.DialectPostgres {
		query = "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys WHERE id = $1"
	}

	apiKey := &APIKey{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&apiKey.ID, &apiKey.ULID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
		&apiKey.Role, &apiKey.CanWrite, &apiKey.CreatedAt, &apiKey.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return apiKey, nil
}

// GetByULID retrieves an API key by ULID.
func (r *APIKeyRepository) GetByULID(ctx context.Context, ulid string) (*APIKey, error) {
	query := "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys WHERE ulid = ?"
	if r.db.Dialect() == database.DialectPostgres {
		query = "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys WHERE ulid = $1"
	}

	apiKey := &APIKey{}
	err := r.db.QueryRow(ctx, query, ulid).Scan(
		&apiKey.ID, &apiKey.ULID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
		&apiKey.Role, &apiKey.CanWrite, &apiKey.CreatedAt, &apiKey.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return apiKey, nil
}

// GetByHash retrieves an API key by its hash.
func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys WHERE key_hash = ?"
	if r.db.Dialect() == database.DialectPostgres {
		query = "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys WHERE key_hash = $1"
	}

	apiKey := &APIKey{}
	err := r.db.QueryRow(ctx, query, keyHash).Scan(
		&apiKey.ID, &apiKey.ULID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
		&apiKey.Role, &apiKey.CanWrite, &apiKey.CreatedAt, &apiKey.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return apiKey, nil
}

// Update updates an API key in the database.
func (r *APIKeyRepository) Update(ctx context.Context, apiKey *APIKey) error {
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = `UPDATE apikeys SET name = $1, description = $2, role = $3, can_write = $4, last_used_at = $5 WHERE id = $6`
	default:
		query = `UPDATE apikeys SET name = ?, description = ?, role = ?, can_write = ?, last_used_at = ? WHERE id = ?`
	}

	_, err := r.db.Exec(ctx, query,
		apiKey.Name, apiKey.Description, apiKey.Role, apiKey.CanWrite, apiKey.LastUsedAt, apiKey.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last used time for an API key.
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id int64) error {
	now := time.Now()
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = "UPDATE apikeys SET last_used_at = $1 WHERE id = $2"
	default:
		query = "UPDATE apikeys SET last_used_at = ? WHERE id = ?"
	}

	_, err := r.db.Exec(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}
	return nil
}

// Delete deletes an API key from the database.
func (r *APIKeyRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM apikeys WHERE id = ?"
	if r.db.Dialect() == database.DialectPostgres {
		query = "DELETE FROM apikeys WHERE id = $1"
	}

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// List retrieves all API keys.
func (r *APIKeyRepository) List(ctx context.Context) ([]*APIKey, error) {
	query := "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		apiKey := &APIKey{}
		err := rows.Scan(
			&apiKey.ID, &apiKey.ULID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
			&apiKey.Role, &apiKey.CanWrite, &apiKey.CreatedAt, &apiKey.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate API keys: %w", err)
	}

	return keys, nil
}

// APIKeyListOptions contains options for listing API keys.
type APIKeyListOptions struct {
	Limit     int
	AfterULID string
}

// ListPaginated retrieves API keys with pagination.
func (r *APIKeyRepository) ListPaginated(ctx context.Context, opts APIKeyListOptions) ([]*APIKey, error) {
	var query string
	var args []any
	argIdx := 1

	baseSelect := "SELECT id, ulid, name, description, key_hash, role, can_write, created_at, last_used_at FROM apikeys"

	if opts.AfterULID != "" {
		if r.db.Dialect() == database.DialectPostgres {
			query = baseSelect + fmt.Sprintf(" WHERE ulid > $%d ORDER BY ulid ASC", argIdx)
		} else {
			query = baseSelect + " WHERE ulid > ? ORDER BY ulid ASC"
		}
		args = append(args, opts.AfterULID)
		argIdx++
	} else {
		query = baseSelect + " ORDER BY ulid ASC"
	}

	if opts.Limit > 0 {
		if r.db.Dialect() == database.DialectPostgres {
			query += fmt.Sprintf(" LIMIT $%d", argIdx)
		} else {
			query += " LIMIT ?"
		}
		args = append(args, opts.Limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		apiKey := &APIKey{}
		err := rows.Scan(
			&apiKey.ID, &apiKey.ULID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
			&apiKey.Role, &apiKey.CanWrite, &apiKey.CreatedAt, &apiKey.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate API keys: %w", err)
	}

	return keys, nil
}

// NameExists checks if an API key name already exists (optionally excluding an ID).
func (r *APIKeyRepository) NameExists(ctx context.Context, name string, excludeID int64) (bool, error) {
	var query string
	var args []any

	if excludeID > 0 {
		query = "SELECT COUNT(*) FROM apikeys WHERE name = ? AND id != ?"
		args = []any{name, excludeID}
		if r.db.Dialect() == database.DialectPostgres {
			query = "SELECT COUNT(*) FROM apikeys WHERE name = $1 AND id != $2"
		}
	} else {
		query = "SELECT COUNT(*) FROM apikeys WHERE name = ?"
		args = []any{name}
		if r.db.Dialect() == database.DialectPostgres {
			query = "SELECT COUNT(*) FROM apikeys WHERE name = $1"
		}
	}

	var count int64
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check name existence: %w", err)
	}
	return count > 0, nil
}

// UpdateKeyHash updates the key hash for an API key (used during rotation).
func (r *APIKeyRepository) UpdateKeyHash(ctx context.Context, id int64, newHash string) error {
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = "UPDATE apikeys SET key_hash = $1 WHERE id = $2"
	default:
		query = "UPDATE apikeys SET key_hash = ? WHERE id = ?"
	}

	_, err := r.db.Exec(ctx, query, newHash, id)
	if err != nil {
		return fmt.Errorf("failed to update key hash: %w", err)
	}
	return nil
}

// UpdateMetadata updates only name, description, and can_write fields.
func (r *APIKeyRepository) UpdateMetadata(ctx context.Context, apiKey *APIKey) error {
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = `UPDATE apikeys SET name = $1, description = $2, can_write = $3 WHERE id = $4`
	default:
		query = `UPDATE apikeys SET name = ?, description = ?, can_write = ? WHERE id = ?`
	}

	_, err := r.db.Exec(ctx, query, apiKey.Name, apiKey.Description, apiKey.CanWrite, apiKey.ID)
	if err != nil {
		return fmt.Errorf("failed to update API key metadata: %w", err)
	}
	return nil
}
