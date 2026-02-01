# PRD-044-A: Core Authentication (JWT + API Key + Database Schema)

## Overview

Implement the foundational authentication infrastructure for Moon, including database schema, JWT token management, API key management, bootstrap admin creation, and core authentication middleware. This is the first phase of the authentication system and provides the base for all other auth features.

This PRD implements:
- Database schema for users, refresh tokens, and API keys
- JWT access and refresh token generation/validation
- API key generation and validation
- Bootstrap admin account creation
- Authentication middleware (JWT and API key)
- Configuration loading and validation
- Core authentication endpoints: `/auth:login`, `/auth:logout`, `/auth:refresh`, `/auth:me`

### Problem Statement

Moon requires mandatory authentication for all operations. The system must support both interactive users (JWT) and machine-to-machine integrations (API keys) with role-based access control. Authentication is always enabled and cannot be disabled.

**BREAKING CHANGE:** This introduces mandatory authentication. All API clients must be updated to authenticate.

### Dependencies

- Existing: `config` package, `database` package, `ulid` package, `constants` package
- New: `golang.org/x/crypto/bcrypt` (standard library extension)
- New: `github.com/golang-jwt/jwt/v5` (JWT library)

---

## Requirements

### FR-1: Database Schema

**FR-1.1: Users Table**
```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ulid TEXT UNIQUE NOT NULL,
  username TEXT UNIQUE NOT NULL,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('admin', 'user')),
  can_write BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_login_at TIMESTAMP
);

CREATE INDEX idx_users_ulid ON users(ulid);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
```

**FR-1.2: Refresh Tokens Table**
```sql
CREATE TABLE refresh_tokens (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  token_hash TEXT UNIQUE NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);
```

**FR-1.3: API Keys Table**
```sql
CREATE TABLE apikeys (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ulid TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  key_hash TEXT UNIQUE NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('admin', 'user')),
  can_write BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP
);

CREATE INDEX idx_apikeys_ulid ON apikeys(ulid);
CREATE INDEX idx_apikeys_hash ON apikeys(key_hash);
```

**FR-1.4: Schema Creation**
- Create tables on startup if they don't exist
- Support SQLite, PostgreSQL, and MySQL
- Use dialect-specific SQL generation
- Log schema creation events

### FR-2: Configuration

**FR-2.1: Auth Configuration Removal**
- Remove `auth.enabled` flag - authentication is always enabled
- Remove conditional logic for auth toggle

**FR-2.2: JWT Configuration (Required)**
Update existing `JWTConfig`:
```go
type JWTConfig struct {
    Secret        string `mapstructure:"secret"`        // REQUIRED
    AccessExpiry  int    `mapstructure:"access_expiry"` // seconds
    RefreshExpiry int    `mapstructure:"refresh_expiry"` // seconds
}

type BootstrapAdmin struct {
    Username string `mapstructure:"username"`
    Email    string `mapstructure:"email"`
    Password string `mapstructure:"password"`
}
```

**FR-2.3: Configuration Defaults**
```go
JWT: struct {
    Secret        string
    AccessExpiry  int
    RefreshExpiry int
}{
    Secret:        "",  // REQUIRED - server won't start without it
    AccessExpiry:  3600,     // 1 hour
    RefreshExpiry: 604800,   // 7 days
},
BootstrapAdmin: nil,  // REQUIRED on first startup
```

**FR-2.4: Configuration Validation**
- Validate `jwt.access_expiry > 0`
- Validate `jwt.refresh_expiry > jwt.access_expiry`
- Validate bootstrap admin fields if present

### FR-3: User Model

**FR-3.1: User Struct**
```go
type User struct {
    ID           int       `db:"id" json:"-"`
    ULID         string    `db:"ulid" json:"id"`
    Username     string    `db:"username" json:"username"`
    Email        string    `db:"email" json:"email"`
    PasswordHash string    `db:"password_hash" json:"-"`
    Role         string    `db:"role" json:"role"`
    CanWrite     bool      `db:"can_write" json:"can_write"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
    UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
    LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
}
```

**FR-3.2: Password Hashing**
- Use `golang.org/x/crypto/bcrypt`
- Cost factor: 12 (constant in `constants` package)
- Functions:
  - `HashPassword(password string) (string, error)`
  - `ComparePassword(hash, password string) error`

**FR-3.3: User Repository**
```go
type UserRepository interface {
    Create(user *User) error
    GetByID(ulid string) (*User, error)
    GetByUsername(username string) (*User, error)
    GetByEmail(email string) (*User, error)
    Update(user *User) error
    Delete(ulid string) error
    List(limit int, cursor string, role string) ([]*User, string, error)
    CountAdmins() (int, error)
    UpdateLastLogin(ulid string) error
}
```

### FR-4: JWT Token Management

**FR-4.1: Token Types**
```go
type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    TokenType    string `json:"token_type"`
}

type UserClaims struct {
    UserID   string `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    CanWrite bool   `json:"can_write"`
    jwt.RegisteredClaims
}
```

**FR-4.2: Token Generation**
- `GenerateTokenPair(user *User, secret string, accessExpiry, refreshExpiry int) (*TokenPair, error)`
- Access token: JWT with user claims
- Refresh token: JWT with minimal claims
- Both signed with HS256 algorithm

**FR-4.3: Token Validation**
- `ValidateAccessToken(token, secret string) (*UserClaims, error)`
- `ValidateRefreshToken(token, secret string) (string, error)` // returns user ULID
- Check signature validity
- Check expiration
- Check nbf (not before)
- Allow clock skew tolerance (30 seconds)

**FR-4.4: Refresh Token Storage**
```go
type RefreshToken struct {
    ID         int       `db:"id"`
    UserID     int       `db:"user_id"`
    TokenHash  string    `db:"token_hash"`
    ExpiresAt  time.Time `db:"expires_at"`
    CreatedAt  time.Time `db:"created_at"`
    LastUsedAt *time.Time `db:"last_used_at"`
}

type RefreshTokenRepository interface {
    Create(rt *RefreshToken) error
    GetByHash(hash string) (*RefreshToken, error)
    Invalidate(hash string) error
    InvalidateAllForUser(userID int) error
    CleanupExpired() error
}
```

- Store SHA-256 hash of refresh token
- Single-use tokens: invalidate after successful refresh

### FR-5: API Key Management

**FR-5.1: API Key Struct**
```go
type APIKey struct {
    ID          int       `db:"id" json:"-"`
    ULID        string    `db:"ulid" json:"id"`
    Name        string    `db:"name" json:"name"`
    Description string    `db:"description" json:"description,omitempty"`
    KeyHash     string    `db:"key_hash" json:"-"`
    Role        string    `db:"role" json:"role"`
    CanWrite    bool      `db:"can_write" json:"can_write"`
    CreatedAt   time.Time `db:"created_at" json:"created_at"`
    LastUsedAt  *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
}
```

**FR-5.2: Key Generation**
- Format: `moon_live_` prefix + 64 characters
- Character set: base62 (alphanumeric + `-` + `_`)
- Use `crypto/rand` for generation
- Function: `GenerateAPIKey() (string, error)`

**FR-5.3: Key Hashing**
- Use SHA-256 for storage
- Function: `HashAPIKey(key string) string`
- Compare in constant time to prevent timing attacks

**FR-5.4: API Key Repository**
```go
type APIKeyRepository interface {
    Create(key *APIKey) error
    GetByID(ulid string) (*APIKey, error)
    GetByHash(hash string) (*APIKey, error)
    Update(key *APIKey) error
    Delete(ulid string) error
    List(limit int, cursor string) ([]*APIKey, string, error)
    UpdateLastUsed(ulid string) error
}
```

### FR-6: Bootstrap Admin

**FR-6.1: Bootstrap Logic (Required)**
On server startup:
1. Count admin users in database
2. If admin count == 0 and bootstrap config present:
   - Create admin user from config
   - Hash bootstrap password
   - Log success: "Bootstrap admin created: {email}"
3. If admin count == 0 and NO bootstrap config:
   - **ERROR:** Server fails to start with message: "No admin user exists. Provide auth.bootstrap_admin configuration."
4. If admin count > 0:
   - Skip bootstrap silently

**FR-6.2: Bootstrap Security**
- Never store bootstrap config after first use
- Recommend removing from config file
- Log reminder to change password

### FR-7: Authentication Middleware

**FR-7.1: Middleware Structure**
```go
func AuthMiddleware(config *AuthConfig, jwtSecret string, userRepo UserRepository, apiKeyRepo APIKeyRepository) func(http.Handler) http.Handler
```

**FR-7.2: Authentication Flow**
1. Check path against unprotected list (`/health`, `/doc/*`, `/auth:login`, `/auth:refresh`)
2. If unprotected, skip authentication
3. Extract JWT from `Authorization: Bearer` header OR API key from `X-API-Key` header
4. If both present, prioritize JWT
5. If neither present, return 401
6. Validate credentials
7. Load user/key info
8. Add to request context
9. Call next handler

**FR-7.3: Context Keys**
```go
const (
    ContextKeyUserClaims = "user_claims"
    ContextKeyAPIKeyInfo = "apikey_info"
)
```

**FR-7.4: Helper Functions**
```go
func GetUserFromContext(ctx context.Context) (*UserClaims, bool)
func GetAPIKeyFromContext(ctx context.Context) (*APIKey, bool)
func GetAuthEntity(ctx context.Context) (isUser bool, userID string, role string, canWrite bool)
```

### FR-8: Core Auth Endpoints

**FR-8.1: POST /auth:login**
- Request: `{"username": "...", "password": "..."}`
- Validate username and password presence
- Lookup user by username OR email
- Compare password hash
- Generate token pair
- Store refresh token
- Update last_login_at
- Return tokens + user info
- Rate limit: 5 attempts per 15 min per IP/username

**FR-8.2: POST /auth:logout**
- Require: `Authorization: Bearer <token>` header
- Request: `{"refresh_token": "..."}`
- Validate access token
- Invalidate refresh token
- Return success message

**FR-8.3: POST /auth:refresh**
- Request: `{"refresh_token": "..."}`
- Validate refresh token
- Check expiration
- Check if token already used
- Invalidate old refresh token
- Generate new token pair
- Store new refresh token
- Return new tokens

**FR-8.4: GET /auth:me**
- Require: `Authorization: Bearer <token>` header
- Extract user from token
- Return user info (excluding password_hash)

**FR-8.5: POST /auth:me**
- Require: `Authorization: Bearer <token>` header
- Support two operations:
  1. Update email: `{"email": "new@example.com"}`
  2. Change password: `{"current_password": "...", "new_password": "..."}`
- For password change:
  - Validate current password
  - Validate new password against policy
  - Update password hash
  - Invalidate all refresh tokens
  - Return success
- For email change:
  - Validate email format
  - Check email not already taken
  - Update email
  - Return updated user

---

## Acceptance Criteria

### AC-1: Database Schema

**Verification:**
- [ ] Tables created on startup
- [ ] Indexes created correctly
- [ ] Foreign key constraints enforced
- [ ] Works on SQLite, PostgreSQL, MySQL

**Test:**
```go
// Test schema creation
db := setupTestDB()
err := CreateAuthSchema(db)
assert.NoError(t, err)

// Verify tables exist
tables := []string{"users", "refresh_tokens", "apikeys"}
for _, table := range tables {
    exists := tableExists(db, table)
    assert.True(t, exists)
}
```

### AC-2: Configuration Loading

**Verification:**
- [ ] Auth config loaded correctly
- [ ] Defaults applied when fields missing
- [ ] Validation catches invalid values
- [ ] JWT secret required when auth enabled

**Test:**
```go
// Test valid config
cfg, err := LoadConfig("testdata/auth-enabled.yaml")
assert.NoError(t, err)
assert.True(t, cfg.Auth.Enabled)
assert.Equal(t, 3600, cfg.JWT.AccessExpiry)

// Test missing JWT secret
cfg, err = LoadConfig("testdata/auth-no-secret.yaml")
assert.Error(t, err)
assert.Contains(t, err.Error(), "jwt.secret")
```

### AC-3: Password Hashing

**Verification:**
- [ ] Passwords hashed with bcrypt
- [ ] Cost factor is 12
- [ ] Same password produces different hashes
- [ ] Comparison works correctly

**Test:**
```go
password := "SecurePass123"

// Hash password
hash, err := HashPassword(password)
assert.NoError(t, err)
assert.NotEqual(t, password, hash)

// Verify password
err = ComparePassword(hash, password)
assert.NoError(t, err)

// Wrong password fails
err = ComparePassword(hash, "WrongPass")
assert.Error(t, err)

// Same password produces different hashes
hash2, err := HashPassword(password)
assert.NoError(t, err)
assert.NotEqual(t, hash, hash2)
```

### AC-4: JWT Token Generation

**Verification:**
- [ ] Access token generated with user claims
- [ ] Refresh token generated
- [ ] Tokens signed correctly
- [ ] Expiration set correctly

**Test:**
```go
user := &User{
    ULID:     "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    Username: "testuser",
    Role:     "user",
    CanWrite: false,
}

secret := "test-secret-key-min-32-chars-long"
pair, err := GenerateTokenPair(user, secret, 3600, 604800)
assert.NoError(t, err)
assert.NotEmpty(t, pair.AccessToken)
assert.NotEmpty(t, pair.RefreshToken)
assert.Equal(t, 3600, pair.ExpiresIn)
assert.Equal(t, "Bearer", pair.TokenType)
```

### AC-5: JWT Token Validation

**Verification:**
- [ ] Valid tokens pass validation
- [ ] Expired tokens rejected
- [ ] Invalid signature rejected
- [ ] Claims extracted correctly

**Test:**
```go
// Generate valid token
token, _ := GenerateAccessToken(user, secret, 3600)

// Validate valid token
claims, err := ValidateAccessToken(token, secret)
assert.NoError(t, err)
assert.Equal(t, user.ULID, claims.UserID)
assert.Equal(t, user.Role, claims.Role)

// Validate with wrong secret
_, err = ValidateAccessToken(token, "wrong-secret")
assert.Error(t, err)

// Validate expired token
expiredToken, _ := GenerateAccessToken(user, secret, -1)
_, err = ValidateAccessToken(expiredToken, secret)
assert.Error(t, err)
```

### AC-6: API Key Generation

**Verification:**
- [ ] Keys have correct format (prefix + 64 chars)
- [ ] Keys are cryptographically random
- [ ] Keys are unique
- [ ] Character set is base62

**Test:**
```go
key1, err := GenerateAPIKey()
assert.NoError(t, err)
assert.True(t, strings.HasPrefix(key1, "moon_live_"))
assert.Equal(t, 74, len(key1)) // prefix (10) + 64 chars

// Generate multiple keys - should be unique
key2, _ := GenerateAPIKey()
key3, _ := GenerateAPIKey()
assert.NotEqual(t, key1, key2)
assert.NotEqual(t, key2, key3)
assert.NotEqual(t, key1, key3)

// Validate character set
validChars := regexp.MustCompile(`^moon_live_[a-zA-Z0-9_-]{64}$`)
assert.True(t, validChars.MatchString(key1))
```

### AC-7: Bootstrap Admin

**Verification:**
- [ ] Admin created when none exists
- [ ] Bootstrap skipped when admin exists
- [ ] Warning logged when no admin and no config
- [ ] Bootstrap password hashed

**Test:**
```bash
# Test bootstrap creation
# Remove test database
rm /tmp/moon-test.db

# Start with bootstrap config
./moon --config testdata/bootstrap.yaml

# Check logs
grep "Bootstrap admin created" /var/log/moon/main.log

# Verify admin in database
sqlite3 /tmp/moon-test.db "SELECT username, role FROM users WHERE role='admin';"
# Output: admin|admin

# Restart server - bootstrap should be skipped
./moon --config testdata/bootstrap.yaml
grep "Admin user already exists" /var/log/moon/main.log
```

### AC-8: Authentication Middleware

**Verification:**
- [ ] JWT authentication works
- [ ] API key authentication works
- [ ] JWT takes precedence when both present
- [ ] 401 on missing/invalid credentials
- [ ] User/key info added to context

**Test:**
```go
// Test JWT authentication
req := httptest.NewRequest("GET", "/protected", nil)
req.Header.Set("Authorization", "Bearer "+validJWT)

recorder := httptest.NewRecorder()
handler := AuthMiddleware(config, secret, userRepo, keyRepo)(http.HandlerFunc(protectedHandler))
handler.ServeHTTP(recorder, req)

assert.Equal(t, 200, recorder.Code)

// Test API key authentication
req = httptest.NewRequest("GET", "/protected", nil)
req.Header.Set("X-API-Key", validAPIKey)

recorder = httptest.NewRecorder()
handler.ServeHTTP(recorder, req)

assert.Equal(t, 200, recorder.Code)

// Test no credentials
req = httptest.NewRequest("GET", "/protected", nil)
recorder = httptest.NewRecorder()
handler.ServeHTTP(recorder, req)

assert.Equal(t, 401, recorder.Code)
```

### AC-9: Login Endpoint

**Verification:**
- [ ] Valid credentials return tokens
- [ ] Invalid credentials return 401
- [ ] Missing fields return 400
- [ ] last_login_at updated
- [ ] Refresh token stored in database

**Test:**
```bash
# Valid login
curl -X POST http://localhost:6006/auth:login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"SecurePass123"}'

# Expected response (200):
# {
#   "access_token": "eyJhbGc...",
#   "refresh_token": "eyJhbGc...",
#   "expires_in": 3600,
#   "token_type": "Bearer",
#   "user": {
#     "id": "01ARZ...",
#     "username": "admin",
#     "role": "admin",
#     "can_write": true
#   }
# }

# Invalid credentials
curl -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"wrong"}'
# Expected: 401 {"error":{"code":"INVALID_CREDENTIALS",...}}
```

### AC-10: Refresh Endpoint

**Verification:**
- [ ] Valid refresh token returns new token pair
- [ ] Old refresh token invalidated
- [ ] Expired refresh token rejected
- [ ] Already-used refresh token rejected

**Test:**
```bash
# Login to get refresh token
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"SecurePass123"}')

REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.refresh_token')

# Refresh tokens
curl -X POST http://localhost:6006/auth:refresh \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"

# Expected: 200 with new token pair

# Try to use same refresh token again
curl -X POST http://localhost:6006/auth:refresh \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"

# Expected: 401 (token already used)
```

### AC-11: Logout Endpoint

**Verification:**
- [ ] Logout invalidates refresh token
- [ ] Requires valid access token
- [ ] Returns success message

**Test:**
```bash
# Login
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"SecurePass123"}')

ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.refresh_token')

# Logout
curl -X POST http://localhost:6006/auth:logout \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"

# Expected: 200 {"message":"Logged out successfully"}

# Try to refresh with logged out token
curl -X POST http://localhost:6006/auth:refresh \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"

# Expected: 401 (token invalidated)
```

### AC-12: No Regression

**Verification:**
- [ ] All existing tests pass
- [ ] No compilation warnings
- [ ] With auth disabled, system behaves as before
- [ ] Performance unchanged for non-auth operations

**Test:**
```bash
# Run full test suite
go test ./... -v

# Test with auth disabled
AUTH_ENABLED=false go test ./...

# Compare performance
go test -bench=. ./internal/handlers > before.txt
# (implement auth)
go test -bench=. ./internal/handlers > after.txt
# Compare benchmark results
```

---

## Implementation Checklist

- [ ] Create `cmd/moon/internal/auth` package
- [ ] Define database schema constants
- [ ] Implement schema creation logic (dialect-specific)
- [ ] Add auth config structures to `config` package
- [ ] Update `config.Defaults` with auth defaults
- [ ] Implement password hashing functions
- [ ] Implement User model and repository
- [ ] Implement RefreshToken model and repository
- [ ] Implement APIKey model and repository
- [ ] Implement JWT token generation
- [ ] Implement JWT token validation
- [ ] Implement API key generation
- [ ] Implement API key hashing
- [ ] Implement bootstrap admin logic
- [ ] Implement authentication middleware
- [ ] Implement POST /auth:login handler
- [ ] Implement POST /auth:logout handler
- [ ] Implement POST /auth:refresh handler
- [ ] Implement GET /auth:me handler
- [ ] Implement POST /auth:me handler
- [ ] Add auth routes to server setup
- [ ] Write unit tests for password hashing
- [ ] Write unit tests for JWT operations
- [ ] Write unit tests for API key operations
- [ ] Write unit tests for repositories
- [ ] Write integration tests for auth endpoints
- [ ] Update samples/moon.conf with auth section
- [ ] Document configuration in comments

---

## Related PRDs

- [PRD-044: Authentication System](044-authentication-system.md) - Parent PRD
- [PRD-002: Configuration Loader](002-configuration-loader.md) - Configuration architecture
- [PRD-003: Database Driver Abstraction](003-database-driver-abstraction.md) - Database interface

---

## References

- [SPEC_AUTH.md](../SPEC_AUTH.md) - Authentication specification
- [SPEC.md](../SPEC.md) - System architecture
- [Go crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) - Password hashing
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt) - JWT library
