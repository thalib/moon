package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/thalib/moon/cmd/moon/internal/constants"
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
	apiKey.ID = moonulid.Generate()
	apiKey.CreatedAt = time.Now()

	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf(`INSERT INTO %s (id, name, description, key_hash, role, can_write, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING pkid`, constants.TableAPIKeys)
		err := r.db.QueryRow(ctx, query,
			apiKey.ID, apiKey.Name, apiKey.Description, apiKey.KeyHash,
			apiKey.Role, apiKey.CanWrite, apiKey.CreatedAt,
		).Scan(&apiKey.PKID)
		if err != nil {
			return fmt.Errorf("failed to create API key: %w", err)
		}
		return nil
	default:
		query = fmt.Sprintf(`INSERT INTO %s (id, name, description, key_hash, role, can_write, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, constants.TableAPIKeys)
		result, err := r.db.Exec(ctx, query,
			apiKey.ID, apiKey.Name, apiKey.Description, apiKey.KeyHash,
			apiKey.Role, apiKey.CanWrite, apiKey.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create API key: %w", err)
		}
		pkid, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get API key ID: %w", err)
		}
		apiKey.PKID = pkid
		return nil
	}
}

// GetByPKID retrieves an API key by internal primary key ID.
func (r *APIKeyRepository) GetByPKID(ctx context.Context, pkid int64) (*APIKey, error) {
	query := fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s WHERE pkid = ?", constants.TableAPIKeys)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s WHERE pkid = $1", constants.TableAPIKeys)
	}

	apiKey := &APIKey{}
	err := r.db.QueryRow(ctx, query, pkid).Scan(
		&apiKey.PKID, &apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
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

// GetByID retrieves an API key by ID (ULID).
func (r *APIKeyRepository) GetByID(ctx context.Context, id string) (*APIKey, error) {
	query := fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s WHERE id = ?", constants.TableAPIKeys)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s WHERE id = $1", constants.TableAPIKeys)
	}

	apiKey := &APIKey{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&apiKey.PKID, &apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
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
	query := fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s WHERE key_hash = ?", constants.TableAPIKeys)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s WHERE key_hash = $1", constants.TableAPIKeys)
	}

	apiKey := &APIKey{}
	err := r.db.QueryRow(ctx, query, keyHash).Scan(
		&apiKey.PKID, &apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
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
		query = fmt.Sprintf(`UPDATE %s SET name = $1, description = $2, role = $3, can_write = $4, last_used_at = $5 WHERE pkid = $6`, constants.TableAPIKeys)
	default:
		query = fmt.Sprintf(`UPDATE %s SET name = ?, description = ?, role = ?, can_write = ?, last_used_at = ? WHERE pkid = ?`, constants.TableAPIKeys)
	}

	_, err := r.db.Exec(ctx, query,
		apiKey.Name, apiKey.Description, apiKey.Role, apiKey.CanWrite, apiKey.LastUsedAt, apiKey.PKID,
	)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last used time for an API key.
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, pkid int64) error {
	now := time.Now()
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf("UPDATE %s SET last_used_at = $1 WHERE pkid = $2", constants.TableAPIKeys)
	default:
		query = fmt.Sprintf("UPDATE %s SET last_used_at = ? WHERE pkid = ?", constants.TableAPIKeys)
	}

	_, err := r.db.Exec(ctx, query, now, pkid)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}
	return nil
}

// Delete deletes an API key from the database.
func (r *APIKeyRepository) Delete(ctx context.Context, pkid int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE pkid = ?", constants.TableAPIKeys)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE pkid = $1", constants.TableAPIKeys)
	}

	_, err := r.db.Exec(ctx, query, pkid)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// List retrieves all API keys.
func (r *APIKeyRepository) List(ctx context.Context) ([]*APIKey, error) {
	query := fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s ORDER BY created_at DESC", constants.TableAPIKeys)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		apiKey := &APIKey{}
		err := rows.Scan(
			&apiKey.PKID, &apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
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
	Limit   int
	AfterID string
}

// ListPaginated retrieves API keys with pagination.
func (r *APIKeyRepository) ListPaginated(ctx context.Context, opts APIKeyListOptions) ([]*APIKey, error) {
	var query string
	var args []any
	argIdx := 1

	baseSelect := fmt.Sprintf("SELECT pkid, id, name, description, key_hash, role, can_write, created_at, last_used_at FROM %s", constants.TableAPIKeys)

	if opts.AfterID != "" {
		if r.db.Dialect() == database.DialectPostgres {
			query = baseSelect + fmt.Sprintf(" WHERE id > $%d ORDER BY id ASC", argIdx)
		} else {
			query = baseSelect + " WHERE id > ? ORDER BY id ASC"
		}
		args = append(args, opts.AfterID)
		argIdx++
	} else {
		query = baseSelect + " ORDER BY id ASC"
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
			&apiKey.PKID, &apiKey.ID, &apiKey.Name, &apiKey.Description, &apiKey.KeyHash,
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

// NameExists checks if an API key name already exists (optionally excluding a primary key ID).
func (r *APIKeyRepository) NameExists(ctx context.Context, name string, excludePKID int64) (bool, error) {
	var query string
	var args []any

	if excludePKID > 0 {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE name = ? AND pkid != ?", constants.TableAPIKeys)
		args = []any{name, excludePKID}
		if r.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE name = $1 AND pkid != $2", constants.TableAPIKeys)
		}
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE name = ?", constants.TableAPIKeys)
		args = []any{name}
		if r.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE name = $1", constants.TableAPIKeys)
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
func (r *APIKeyRepository) UpdateKeyHash(ctx context.Context, pkid int64, newHash string) error {
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf("UPDATE %s SET key_hash = $1 WHERE pkid = $2", constants.TableAPIKeys)
	default:
		query = fmt.Sprintf("UPDATE %s SET key_hash = ? WHERE pkid = ?", constants.TableAPIKeys)
	}

	_, err := r.db.Exec(ctx, query, newHash, pkid)
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
		query = fmt.Sprintf(`UPDATE %s SET name = $1, description = $2, can_write = $3 WHERE pkid = $4`, constants.TableAPIKeys)
	default:
		query = fmt.Sprintf(`UPDATE %s SET name = ?, description = ?, can_write = ? WHERE pkid = ?`, constants.TableAPIKeys)
	}

	_, err := r.db.Exec(ctx, query, apiKey.Name, apiKey.Description, apiKey.CanWrite, apiKey.PKID)
	if err != nil {
		return fmt.Errorf("failed to update API key metadata: %w", err)
	}
	return nil
}
