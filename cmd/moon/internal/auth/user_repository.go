package auth

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
	moonulid "github.com/thalib/moon/cmd/moon/internal/ulid"
)

// UserRepository provides database operations for users.
type UserRepository struct {
	db database.Driver
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db database.Driver) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	user.ID = moonulid.Generate()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf(`INSERT INTO %s (id, username, email, password_hash, role, can_write, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING pkid`, constants.TableUsers)
		err := r.db.QueryRow(ctx, query,
			user.ID, user.Username, user.Email, user.PasswordHash,
			user.Role, user.CanWrite, user.CreatedAt, user.UpdatedAt,
		).Scan(&user.PKID)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		return nil
	default:
		query = fmt.Sprintf(`INSERT INTO %s (id, username, email, password_hash, role, can_write, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, constants.TableUsers)
		result, err := r.db.Exec(ctx, query,
			user.ID, user.Username, user.Email, user.PasswordHash,
			user.Role, user.CanWrite, user.CreatedAt, user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		pkid, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get user PKID: %w", err)
		}
		user.PKID = pkid
		return nil
	}
}

// GetByPKID retrieves a user by internal PKID.
func (r *UserRepository) GetByPKID(ctx context.Context, pkid int64) (*User, error) {
	query := fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE pkid = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE pkid = $1", constants.TableUsers)
	}

	user := &User{}
	err := r.db.QueryRow(ctx, query, pkid).Scan(
		&user.PKID, &user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.CanWrite, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetByID retrieves a user by ID (ULID).
func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	query := fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE id = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE id = $1", constants.TableUsers)
	}

	user := &User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.PKID, &user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.CanWrite, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetByUsername retrieves a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE username = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE username = $1", constants.TableUsers)
	}

	user := &User{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.PKID, &user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.CanWrite, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE email = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s WHERE email = $1", constants.TableUsers)
	}

	user := &User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.PKID, &user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.CanWrite, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// Update updates a user in the database.
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf(`UPDATE %s SET username = $1, email = $2, password_hash = $3, role = $4, 
			can_write = $5, updated_at = $6, last_login_at = $7 WHERE pkid = $8`, constants.TableUsers)
	default:
		query = fmt.Sprintf(`UPDATE %s SET username = ?, email = ?, password_hash = ?, role = ?, 
			can_write = ?, updated_at = ?, last_login_at = ? WHERE pkid = ?`, constants.TableUsers)
	}

	_, err := r.db.Exec(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role,
		user.CanWrite, user.UpdatedAt, user.LastLoginAt, user.PKID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// UpdateLastLogin updates the last login time for a user.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userPKID int64) error {
	now := time.Now()
	var query string
	switch r.db.Dialect() {
	case database.DialectPostgres:
		query = fmt.Sprintf("UPDATE %s SET last_login_at = $1, updated_at = $2 WHERE pkid = $3", constants.TableUsers)
	default:
		query = fmt.Sprintf("UPDATE %s SET last_login_at = ?, updated_at = ? WHERE pkid = ?", constants.TableUsers)
	}

	_, err := r.db.Exec(ctx, query, now, now, userPKID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// Delete deletes a user from the database.
func (r *UserRepository) Delete(ctx context.Context, pkid int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE pkid = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE pkid = $1", constants.TableUsers)
	}

	_, err := r.db.Exec(ctx, query, pkid)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", constants.TableUsers)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// Exists checks if a user exists by username or email.
func (r *UserRepository) Exists(ctx context.Context, username, email string) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE username = ? OR email = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE username = $1 OR email = $2", constants.TableUsers)
	}

	var count int64
	err := r.db.QueryRow(ctx, query, username, email).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

// UsernameExists checks if a username already exists (optionally excluding a user PKID).
func (r *UserRepository) UsernameExists(ctx context.Context, username string, excludePKID int64) (bool, error) {
	var query string
	var args []any

	if excludePKID > 0 {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE username = ? AND pkid != ?", constants.TableUsers)
		args = []any{username, excludePKID}
		if r.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE username = $1 AND pkid != $2", constants.TableUsers)
		}
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE username = ?", constants.TableUsers)
		args = []any{username}
		if r.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE username = $1", constants.TableUsers)
		}
	}

	var count int64
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}
	return count > 0, nil
}

// EmailExists checks if an email already exists (optionally excluding a user PKID).
func (r *UserRepository) EmailExists(ctx context.Context, email string, excludePKID int64) (bool, error) {
	var query string
	var args []any

	if excludePKID > 0 {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE email = ? AND pkid != ?", constants.TableUsers)
		args = []any{email, excludePKID}
		if r.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE email = $1 AND pkid != $2", constants.TableUsers)
		}
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE email = ?", constants.TableUsers)
		args = []any{email}
		if r.db.Dialect() == database.DialectPostgres {
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE email = $1", constants.TableUsers)
		}
	}

	var count int64
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return count > 0, nil
}

// CountByRole returns the number of users with a specific role.
func (r *UserRepository) CountByRole(ctx context.Context, role string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE role = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE role = $1", constants.TableUsers)
	}

	var count int64
	err := r.db.QueryRow(ctx, query, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by role: %w", err)
	}
	return count, nil
}

// ListOptions contains options for listing users.
type ListOptions struct {
	Limit     int
	AfterID   string
	RoleFilter string
}

// List retrieves users with pagination and optional filtering.
func (r *UserRepository) List(ctx context.Context, opts ListOptions) ([]*User, error) {
	var query string
	var args []any
	argIdx := 1

	baseSelect := fmt.Sprintf("SELECT pkid, id, username, email, password_hash, role, can_write, created_at, updated_at, last_login_at FROM %s", constants.TableUsers)

	var conditions []string
	if opts.AfterID != "" {
		if r.db.Dialect() == database.DialectPostgres {
			conditions = append(conditions, fmt.Sprintf("id > $%d", argIdx))
		} else {
			conditions = append(conditions, "id > ?")
		}
		args = append(args, opts.AfterID)
		argIdx++
	}

	if opts.RoleFilter != "" {
		if r.db.Dialect() == database.DialectPostgres {
			conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		} else {
			conditions = append(conditions, "role = ?")
		}
		args = append(args, opts.RoleFilter)
		argIdx++
	}

	query = baseSelect
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY id ASC"

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
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(
			&user.PKID, &user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Role, &user.CanWrite, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// DeleteByID deletes a user by their ID (ULID).
func (r *UserRepository) DeleteByID(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", constants.TableUsers)
	if r.db.Dialect() == database.DialectPostgres {
		query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", constants.TableUsers)
	}

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
