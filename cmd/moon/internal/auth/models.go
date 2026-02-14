package auth

import (
	"time"
)

// User represents a user in the system.
type User struct {
	PKID         int64      `json:"-"`
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	CanWrite     bool       `json:"can_write"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// UserRole defines the available user roles.
type UserRole string

const (
	// RoleAdmin has full access to the system.
	RoleAdmin UserRole = "admin"
	// RoleUser has limited access to the system.
	RoleUser UserRole = "user"
	// RoleReadOnly has read-only access to the system.
	RoleReadOnly UserRole = "readonly"
)

// ValidRoles returns the list of valid user roles.
func ValidRoles() []UserRole {
	return []UserRole{RoleAdmin, RoleUser, RoleReadOnly}
}

// IsValidRole checks if a role string is valid.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if string(r) == role {
			return true
		}
	}
	return false
}

// RefreshToken represents a refresh token for JWT authentication.
type RefreshToken struct {
	PKID       int64     `json:"-"`
	UserPKID   int64     `json:"-"`
	TokenHash  string    `json:"-"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// IsExpired checks if the refresh token has expired.
func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// APIKey represents an API key for programmatic access.
type APIKey struct {
	PKID        int64      `json:"-"`
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	KeyHash     string     `json:"-"`
	Role        string     `json:"role"`
	CanWrite    bool       `json:"can_write"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// APIKeyPrefix is the prefix for generated API keys.
const APIKeyPrefix = "moon_live_"

// APIKeyLength is the length of the random portion of API keys (64 chars base62).
const APIKeyLength = 64
