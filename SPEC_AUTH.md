# Authentication Design

**Rules** to Follow for this document

- Be written from the user's perspective (outside-in: user → server)
- Focus on simplicity and clarity
- Define only what is needed for real use, not hypothetical features
- Use sections and bullet points unless told to be used
- Keep roles and permissions minimal and easy to manage
- **AIP-136 Custom Actions:** APIs use a colon separator (`:`) to distinguish between the resource and the action, providing a predictable and AI-friendly interface.

## Overview

Moon's authentication system provides two authentication methods:

1. **JWT-based authentication** for interactive users (web/mobile applications)
2. **API Key authentication** for machine-to-machine integrations

Both methods support role-based access control (RBAC) with three roles: `admin`, `user`, and `readonly`.

## Access Types

### User Login (JWT-based)

**Purpose:** For interactive users accessing the system via web or mobile clients.

**Token Types:**

- **Access Token:** Short-lived (configurable, default 1 hour)
- **Refresh Token:** Longer-lived (7 days), single-use

**Authentication Header:**

```
Authorization: Bearer <access_token>
```

**Flow:**

1. User sends credentials to `POST /auth:login`
2. Server validates credentials and returns both access and refresh tokens
3. Client stores tokens securely (httpOnly cookies or secure storage)
4. Client includes access token in `Authorization: Bearer <token>` header for all requests
5. Before access token expires, client calls `POST /auth:refresh` with refresh token
6. Server validates refresh token, invalidates it, and issues new token pair
7. If refresh token expires or is invalid, user must re-authenticate via `/auth:login`

**Token Properties:**

- **JWT Algorithm:** HS256 (HMAC with SHA-256)
- Access tokens are stateless (JWT claims validated cryptographically)
- Refresh tokens are single-use and invalidated after use
- Multiple concurrent sessions supported (each gets separate refresh token)
- Logout only invalidates current session's refresh token
- **Token Blacklist:** In-database blacklist for revoked tokens (logout, password changes)

**JWT Claims Structure:**

Access tokens contain the following claims:
- `user_id`: User's ULID identifier (string, from `id` column)
- `username`: User's username (string)
- `email`: User's email address (string)
- `role`: User's role (`admin`, `user`, or `readonly`)
- `can_write`: Write permission flag (boolean)
- Standard JWT claims: `iss`, `exp`, `iat`, `sub`

**Rate Limits:**

- Standard requests: 100 requests/minute per user
- Login attempts: 5 attempts per 15 minutes per IP/username

### API Key Access

**Purpose:** For machine-to-machine integrations, automation, and service accounts.

**Key Properties:**

- **Prefix:** All keys start with `moon_live_` for easy identification
- Long-lived credentials with no expiration
- Must be manually rotated or revoked
- Minimum 64 characters after prefix (base62: alphanumeric + `-` + `_`)
- Total length: ~74 characters (`moon_live_` + 64 chars)
- Stored as SHA-256 hashes in database
- Each key assigned a role (`admin`, `user`, or `readonly`)
- **Usage Tracking:** `last_used_at` timestamp updated on each request

**Authentication Header:**

```
Authorization: Bearer <api_key>
```

**Flow:**

1. Admin creates API key via `POST /apikeys:create` (specifying name, role, and optional description)
2. Server generates cryptographically secure key with `moon_live_` prefix, returns it once
3. Service stores key securely and includes it in `Authorization: Bearer` header for all requests
4. Keys can be rotated via `POST /apikeys:update` or destroyed via `POST /apikeys:destroy`

**Key Management:**

- API key value returned only once during creation - must be stored securely
- Subsequent API calls only return key metadata (id, name, role, created_at)
- Keys stored as SHA-256 hashes; original value never retrievable
- Admin can list all keys and their metadata via `/apikeys:list`

**Rate Limits:**

- 1000 requests/minute per API key

### Unified Authentication Header

**Standard:**

Both JWT tokens and API keys use the same `Authorization: Bearer` header format:

```
Authorization: Bearer <TOKEN>
```

**Token Type Detection:**

The server automatically detects the token type:
- **JWT tokens:** Three base64-encoded segments separated by dots (e.g., `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`)
- **API keys:** Start with `moon_live_` prefix (e.g., `moon_live_abc123...`)

**Examples:**

```bash
# JWT authentication
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  https://api.moon.example.com/data/users:list

# API key authentication (same header format)
curl -H "Authorization: Bearer moon_live_abc123..." \
  https://api.moon.example.com/data/users:list
```

### Authentication Priority

**Precedence Rules:**

- If no `Authorization: Bearer` header is present on protected endpoint, return `401 Unauthorized`
- If invalid/expired credentials provided, return `401 Unauthorized`
- If valid credentials but insufficient permissions, return `403 Forbidden`

**Configuration Control:**

- JWT authentication: Always enabled (controlled via `jwt.secret` in config)
- API Key authentication: Opt-in (controlled via `apikey.enabled` in config)

## Roles and Permissions

### Role Definitions

**admin Role:**

- Full system access
- Can manage users and API keys
- Can create, read, update, and delete all collections
- Can create, read, update, and delete all data in any collection
- Can access all aggregation and query endpoints
- **Admin Override:** The `can_write` flag is ignored for admin role (always has write access)

**user Role:**

- **Write-enabled by default** (`can_write: true`)
- Can read collections metadata
- Can read data from all collections
- Can use query, filter, and aggregation endpoints
- **Write access controlled per-user via `can_write` flag:**
  - When `can_write: true` (default), user can create, update, and delete data
  - When `can_write: false`, user can only read data
- **Cannot** manage collections schema (create/update/destroy collections)
- **Cannot** manage users or API keys

**readonly Role:**

- **Read-only access** (enforced regardless of `can_write` flag)
- Can read collections metadata
- Can read data from all collections
- Can use query, filter, and aggregation endpoints
- **Cannot** write data even if `can_write` flag is set to true
- **Cannot** manage collections schema
- **Cannot** manage users or API keys

### Permission Matrix

| Action | Admin | User (can_write: false) | User (can_write: true) | Readonly |
|--------|-------|-------------------------|------------------------|----------|
| Manage users/apikeys | ✓ | ✗ | ✗ | ✗ |
| Create/update/delete collections | ✓ | ✗ | ✗ | ✗ |
| Read collections metadata | ✓ | ✓ | ✓ | ✓ |
| Read data from collections | ✓ | ✓ | ✓ | ✓ |
| Create/update/delete data | ✓ | ✗ | ✓ | ✗ |
| Query/filter/aggregate data | ✓ | ✓ | ✓ | ✓ |

## Security Configuration

### Password Policy

**Requirements:**

- Minimum 8 characters (configurable per deployment)
- Must include: uppercase letter, lowercase letter, number
- Optionally require special characters (configurable, default: not required)
- **Supported special characters:** 30+ standard special characters including `!@#$%^&*()_+-=[]{}|;:,.<>?`
- No common passwords or dictionary words (implementation recommended)

**Enforcement Contexts:**

The password policy is applied and validated in the following scenarios:
- User creation (via `POST /users:create`)
- Password change (via `POST /auth:me` with `password` field)
- Password reset (via `POST /users:update` with `action: reset_password`)

**Validation:** All password policy violations return detailed error messages indicating which requirements were not met (e.g., "Password must contain at least one uppercase letter").

**Storage:**

- All passwords hashed using `bcrypt` (Go standard: `golang.org/x/crypto/bcrypt`)
- Cost factor: 12 (recommended for balance of security and performance)
- Salts automatically generated per password

**Password Changes:**

- Password change forces immediate re-authentication
- All active sessions (refresh tokens) invalidated
- User must login again with new password

### Password Reset

**Admin-Initiated Reset:**

- Only admins can reset user passwords (no self-service)
- Admin calls `POST /users:update?id={user_id}` with `{"action": "reset_password", "new_password": "..."}`
- All user's existing sessions immediately invalidated
- User notified of password reset (if email configured)
- User must authenticate with new password

**Security Considerations:**

- No "forgot password" self-service (reduces attack surface)
- Admins must authenticate before resetting passwords
- All password reset actions logged for audit trail

### Validation Constraints

**User Constraints:**

- **Email:** Must be valid RFC-compliant email format
- **Username:** Unique across all users
- **Role:** Must be one of: `admin`, `user`, `readonly`
- **Default can_write:** `true` for user role, `false` for readonly role

**API Key Constraints:**

- **Name:** 3-100 characters, must be unique
- **Description:** Maximum 500 characters (optional)
- **Role:** Must be one of: `admin`, `user`, `readonly`
- **Default can_write:** `false` for all roles
- **Key Format:** `moon_live_` prefix + 64 base62 characters

**Last Admin Protection:**

- System prevents deleting or demoting the last admin user
- Ensures at least one admin always exists for system management
- Admin cannot modify their own role to prevent self-lockout

**Cascade Delete Behavior:**

- Deleting a user automatically removes all associated refresh tokens
- Ensures no orphaned sessions remain after user deletion
- Implemented at application level, not database foreign key cascade

### Rate Limiting

**Per-User (JWT):**

- 100 requests per minute per user
- Failed login attempts: 5 attempts per 15 minutes per IP/username combination
- Returns `429 Too Many Requests` when limit exceeded
- Counter resets after time window expires

**Per-API Key:**

- 1000 requests per minute per key
- Returns `429 Too Many Requests` when limit exceeded
- No failed attempt limit (keys are either valid or invalid)

**Implementation:**

- Uses in-memory rate limit tracking (token bucket or sliding window)
- Rate limits applied after authentication, before authorization
- Rate limit headers included in responses:
  - `X-RateLimit-Limit`: Maximum requests allowed
  - `X-RateLimit-Remaining`: Requests remaining in current window
  - `X-RateLimit-Reset`: Unix timestamp when limit resets

### Session Management

**JWT Sessions:**

- Multiple concurrent sessions allowed per user
- Each login creates new refresh token (stored in database)
- Each device/client maintains separate session
- Logout invalidates only current session's refresh token
- Other sessions remain active until they expire or logout

**Refresh Token Storage:**

- Stored in database with: user_pkid, token_hash, expires_at, created_at, last_used_at
- Tokens are single-use: invalidated immediately after successful refresh
- New refresh token issued with each successful refresh
- **Expired Token Cleanup:** Expired tokens should be purged from the database periodically via a scheduled cleanup job (implementation recommended but not automatic)

**Token Invalidation:**

- **User logout:** Invalidates current session's refresh token only
- **Password change:** Invalidates all user's refresh tokens (forced re-login)
- **Admin revoke:** Admin can revoke all user sessions via `POST /users:update?id={user_id}` with `{"action": "revoke_sessions"}`
- **API keys:** Remain valid until explicitly destroyed via `/apikeys:destroy`

### First Admin Account Bootstrap

**On First Startup:**

1. Check if any admin users exist in database
2. If no admin exists:
   - Check config file for `auth.bootstrap_admin` section:

     ```yaml
     auth:
       bootstrap_admin:
         username: "admin"
         email: "admin@example.com"
         password: "change-me-on-first-login"
     ```

   - If bootstrap config present, create admin user from config
   - If bootstrap config absent, server logs warning and requires manual admin creation via direct database access or setup script
3. If admin already exists, skip bootstrap

**Security Notes:**

- Bootstrap password should be changed immediately after first login
- Bootstrap config should be removed from config file after first startup
- Never commit bootstrap credentials to version control

### Security Best Practices

**Transport Security:**

- All endpoints require HTTPS in production (HTTP redirect recommended)
- TLS 1.2 or higher required
- Strong cipher suites only (no weak ciphers)

**Token Storage:**

- JWTs stored in httpOnly cookies (web) or secure storage (mobile)
- Never store tokens in localStorage or sessionStorage (XSS vulnerability)
- API keys stored securely in environment variables or secret management systems

**Secrets Management:**

- API keys never logged in application logs
- API keys never returned in list/get responses (only metadata)
- JWT secrets stored in config file with restricted file permissions (chmod 600)
- Database passwords stored in config file with restricted file permissions

**CORS Configuration:**

- Configured via config file: `security.cors.allowed_origins`
- Supports wildcards for development only: `["http://localhost:*"]`
- Production should specify exact origins: `["https://app.example.com"]`
- Credentials allowed only for specified origins (no wildcard with credentials)

**Audit Logging:**

- All authentication attempts logged (success and failure)
- All admin actions logged (user/apikey management)
- All rate limit violations logged
- Logs include: timestamp, user/key identifier, action, IP address, user agent
- Sensitive data (passwords, tokens) never logged

## Server Implementation

### Middleware Execution Order

The middleware pipeline executes in strict order to ensure security and correctness:

1. **CORS Middleware:** Handle cross-origin requests, set appropriate headers
2. **Rate Limiting Middleware:** Check and enforce rate limits before processing request
3. **Authentication Middleware:** Extract and validate JWT or API key, add user/key info to context
4. **Authorization Middleware:** Check role and permissions against required access level
5. **Request Validation Middleware:** Validate request body, query params, and path params
6. **Logging Middleware:** Log request details (method, path, status, duration)
7. **Handler:** Execute business logic
8. **Error Handling Middleware:** Catch panics and format error responses

**Implementation Notes:**

- Authentication middleware checks for both JWT and API key (JWT takes precedence)
- Authorization middleware reads user/key info from context (populated by authentication)
- Rate limiting uses user ID (JWT) or API key ID for per-entity limits
- All middleware supports graceful error handling and proper HTTP status codes

### Database Schema

**users Table:**

```sql
CREATE TABLE users (
  pkid INTEGER PRIMARY KEY AUTOINCREMENT,  -- Internal ID (not exposed)
  id TEXT UNIQUE NOT NULL,                 -- ULID, exposed as "id" in API
  username TEXT UNIQUE NOT NULL,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,             -- bcrypt hash
  role TEXT NOT NULL,                      -- "admin" or "user"
  can_write BOOLEAN DEFAULT FALSE,         -- Write permission for user role
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_login_at TIMESTAMP
);

CREATE INDEX idx_users_id ON users(id);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
```

**refresh_tokens Table:**

```sql
CREATE TABLE refresh_tokens (
  pkid INTEGER PRIMARY KEY AUTOINCREMENT,
  user_pkid INTEGER NOT NULL,             -- Foreign key to users.pkid
  token_hash TEXT UNIQUE NOT NULL,        -- SHA-256 hash of refresh token
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP,
  FOREIGN KEY (user_pkid) REFERENCES users(pkid) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user_pkid ON refresh_tokens(user_pkid);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);
```

**apikeys Table:**

```sql
CREATE TABLE apikeys (
  pkid INTEGER PRIMARY KEY AUTOINCREMENT,
  id TEXT UNIQUE NOT NULL,                -- ULID, exposed as "id" in API
  name TEXT NOT NULL,
  description TEXT,
  key_hash TEXT UNIQUE NOT NULL,          -- SHA-256 hash of API key
  role TEXT NOT NULL,                     -- "admin" or "user"
  can_write BOOLEAN DEFAULT FALSE,        -- Write permission for user role
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP
);

CREATE INDEX idx_apikeys_id ON apikeys(id);
CREATE INDEX idx_apikeys_hash ON apikeys(key_hash);
```

**rate_limits Table (Optional - In-Memory Recommended):**

```sql
CREATE TABLE rate_limits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  entity_id TEXT NOT NULL,                -- User ULID or API key ULID
  entity_type TEXT NOT NULL,              -- "user" or "apikey"
  window_start TIMESTAMP NOT NULL,
  request_count INTEGER DEFAULT 0,
  PRIMARY KEY (entity_id, window_start)
);

CREATE INDEX idx_rate_limits_entity ON rate_limits(entity_id, window_start);
```

## API Endpoints

All auth endpoints follow the AIP-136 custom actions pattern (resource:action).

### Authentication Endpoints (Public)

#### POST /auth:login

**Purpose:** Authenticate user and receive access + refresh tokens

**Request:**

```json
{
  "username": "user@example.com",
  "password": "SecurePass123"
}
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "user": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "username": "user@example.com",
    "email": "user@example.com",
    "role": "user",
    "can_write": false
  }
}
```

**Error Responses:**

- `400 Bad Request`: Missing or invalid fields
- `401 Unauthorized`: Invalid credentials
- `429 Too Many Requests`: Too many failed login attempts

---

#### POST /auth:logout

**Purpose:** Invalidate current session's refresh token

**Headers:**

```
Authorization: Bearer <access_token>
```

**Request:**

```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response (200 OK):**

```json
{
  "message": "Logged out successfully"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `400 Bad Request`: Missing refresh token

---

#### POST /auth:refresh

**Purpose:** Exchange refresh token for new access + refresh token pair

**Request:**

```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid, expired, or already-used refresh token
- `400 Bad Request`: Missing refresh token

---

#### GET /auth:me

**Purpose:** Get current authenticated user information

**Headers:**

```
Authorization: Bearer <access_token>
```

**Response (200 OK):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "user@example.com",
  "email": "user@example.com",
  "role": "user",
  "can_write": false,
  "created_at": "2024-01-15T10:30:00Z",
  "last_login_at": "2024-01-16T14:20:00Z"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token

---

#### POST /auth:me

**Purpose:** Update current user's profile (email, password)

**Headers:**

```
Authorization: Bearer <access_token>
```

**Request (update email):**

```json
{
  "email": "newemail@example.com"
}
```

**Request (change password):**

```json
{
  "current_password": "OldPass123",
  "new_password": "NewSecurePass456"
}
```

**Response (200 OK):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "user@example.com",
  "email": "newemail@example.com",
  "role": "user",
  "can_write": false,
  "updated_at": "2024-01-16T15:00:00Z"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid access token or incorrect current password
- `400 Bad Request`: Invalid email format or password doesn't meet policy
- `409 Conflict`: Email already in use

**Notes:**

- Password change invalidates all refresh tokens (forces re-login)
- Email change requires re-verification (if email verification enabled)

---

### User Management Endpoints (Admin Only)

#### GET /users:list

**Purpose:** List all users with pagination

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `limit` (optional, default: 50, max: 100)
- `after` (optional): Cursor for pagination (user ULID)
- `role` (optional): Filter by role ("admin" or "user")

**Response (200 OK):**

```json
{
  "users": [
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin",
      "can_write": true,
      "created_at": "2024-01-10T08:00:00Z",
      "last_login_at": "2024-01-16T09:00:00Z"
    },
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FBX",
      "username": "user1",
      "email": "user1@example.com",
      "role": "user",
      "can_write": false,
      "created_at": "2024-01-12T10:30:00Z",
      "last_login_at": "2024-01-15T14:20:00Z"
    }
  ],
  "next_cursor": "01ARZ3NDEKTSV4RRFFQ69G5FBX"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role

---

#### GET /users:get

**Purpose:** Get specific user by ID

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `id` (required): User ULID

**Response (200 OK):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "user1",
  "email": "user1@example.com",
  "role": "user",
  "can_write": false,
  "created_at": "2024-01-12T10:30:00Z",
  "updated_at": "2024-01-15T11:00:00Z",
  "last_login_at": "2024-01-15T14:20:00Z"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: User does not exist

---

#### POST /users:create

**Purpose:** Create new user (admin only - no self-registration)

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Request:**

```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "SecurePass123",
  "role": "user",
  "can_write": false
}
```

**Response (201 Created):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "newuser",
  "email": "newuser@example.com",
  "role": "user",
  "can_write": false,
  "created_at": "2024-01-16T15:30:00Z"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role
- `400 Bad Request`: Missing required fields or invalid data
- `409 Conflict`: Username or email already exists

---

#### POST /users:update

**Purpose:** Update user properties or perform admin actions

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `id` (required): User ULID

**Request (update role and permissions):**

```json
{
  "role": "user",
  "can_write": true
}
```

**Request (reset password):**

```json
{
  "action": "reset_password",
  "new_password": "NewSecurePass456"
}
```

**Request (revoke all sessions):**

```json
{
  "action": "revoke_sessions"
}
```

**Response (200 OK):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "user1",
  "email": "user1@example.com",
  "role": "user",
  "can_write": true,
  "updated_at": "2024-01-16T16:00:00Z"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: User does not exist
- `400 Bad Request`: Invalid action or data

**Notes:**

- Password reset invalidates all user's refresh tokens
- Revoking sessions invalidates all user's refresh tokens
- Cannot downgrade the last admin user to regular user (must have at least one admin)

---

#### POST /users:destroy

**Purpose:** Delete user account

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `id` (required): User ULID

**Response (200 OK):**

```json
{
  "message": "User deleted successfully",
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role or attempting to delete last admin
- `404 Not Found`: User does not exist

**Notes:**

- Cannot delete the last admin user (must have at least one admin)
- Deleting user cascades to refresh_tokens (via foreign key)

---

### API Key Management Endpoints (Admin Only)

#### GET /apikeys:list

**Purpose:** List all API keys (metadata only)

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `limit` (optional, default: 50, max: 100)
- `after` (optional): Cursor for pagination (key ULID)

**Response (200 OK):**

```json
{
  "apikeys": [
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "name": "Production Service",
      "description": "Main API integration",
      "role": "user",
      "can_write": true,
      "created_at": "2024-01-10T10:00:00Z",
      "last_used_at": "2024-01-16T14:30:00Z"
    },
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FBX",
      "name": "Analytics Service",
      "description": "Read-only analytics integration",
      "role": "user",
      "can_write": false,
      "created_at": "2024-01-12T11:00:00Z",
      "last_used_at": "2024-01-16T15:00:00Z"
    }
  ],
  "next_cursor": "01ARZ3NDEKTSV4RRFFQ69G5FBX"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role

**Notes:**

- Actual API key value is never returned (only metadata)

---

#### GET /apikeys:get

**Purpose:** Get specific API key metadata by ID

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `id` (required): API key ULID

**Response (200 OK):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "name": "Production Service",
  "description": "Main API integration",
  "role": "user",
  "can_write": true,
  "created_at": "2024-01-10T10:00:00Z",
  "last_used_at": "2024-01-16T14:30:00Z"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: API key does not exist

---

#### POST /apikeys:create

**Purpose:** Create new API key

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Request:**

```json
{
  "name": "New Service",
  "description": "Optional description",
  "role": "user",
  "can_write": false
}
```

**Response (201 Created):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "name": "New Service",
  "description": "Optional description",
  "role": "user",
  "can_write": false,
  "key": "moon_live_abc123...xyz789",
  "created_at": "2024-01-16T16:30:00Z",
  "warning": "Store this key securely. It will not be shown again."
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role
- `400 Bad Request`: Missing required fields or invalid data
- `409 Conflict`: API key name already exists

**Notes:**

- API key value returned only once during creation
- Key format: `moon_live_` prefix + 64 characters (base62)
- Key stored as SHA-256 hash in database

---

#### POST /apikeys:update

**Purpose:** Update API key metadata or rotate key

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `id` (required): API key ULID

**Request (update metadata):**

```json
{
  "name": "Updated Service Name",
  "description": "Updated description",
  "can_write": true
}
```

**Request (rotate key):**

```json
{
  "action": "rotate"
}
```

**Response (200 OK - metadata update):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "name": "Updated Service Name",
  "description": "Updated description",
  "role": "user",
  "can_write": true,
  "updated_at": "2024-01-16T17:00:00Z"
}
```

**Response (200 OK - key rotation):**

```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "name": "Production Service",
  "key": "moon_live_def456...uvw012",
  "created_at": "2024-01-16T17:00:00Z",
  "warning": "Store this key securely. The old key is now invalid."
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: API key does not exist
- `400 Bad Request`: Invalid action or data

**Notes:**

- Key rotation invalidates old key immediately
- New key returned only once after rotation

---

#### POST /apikeys:destroy

**Purpose:** Delete API key

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**

- `id` (required): API key ULID

**Response (200 OK):**

```json
{
  "message": "API key deleted successfully",
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

**Error Responses:**

- `401 Unauthorized`: Invalid or missing access token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: API key does not exist

---

## Error Responses

All authentication endpoints return consistent JSON error format following SPEC.md conventions.

### Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}
  }
}
```

### Common HTTP Status Codes

| Status Code | Name | When Used |
|-------------|------|-----------|
| 200 | OK | Request succeeded |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request data (missing fields, validation errors) |
| 401 | Unauthorized | Missing, invalid, or expired credentials |
| 403 | Forbidden | Valid credentials but insufficient permissions |
| 404 | Not Found | Resource does not exist |
| 409 | Conflict | Resource conflict (duplicate username/email/name) |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server-side error (should be rare) |

### Error Code Reference

**Authentication Errors (401):**

- `MISSING_AUTH_HEADER`: No Authorization header provided
- `INVALID_TOKEN_FORMAT`: Authorization header not in "Bearer <token>" format
- `INVALID_TOKEN`: Token signature invalid or token malformed
- `EXPIRED_TOKEN`: Access token or refresh token has expired
- `REVOKED_TOKEN`: Refresh token has been revoked or already used
- `INVALID_CREDENTIALS`: Username/password combination incorrect
- `INVALID_API_KEY`: API key does not exist or is invalid

**Authorization Errors (403):**

- `INSUFFICIENT_PERMISSIONS`: User/API key lacks required role or permission
- `ADMIN_REQUIRED`: Endpoint requires admin role
- `WRITE_PERMISSION_REQUIRED`: User role requires can_write flag for this action
- `CANNOT_DELETE_LAST_ADMIN`: Cannot delete the last admin user
- `CANNOT_MODIFY_SELF_ROLE`: Admin cannot change their own role

**Validation Errors (400):**

- `MISSING_REQUIRED_FIELD`: Required field missing from request body
- `INVALID_FIELD_VALUE`: Field value does not meet requirements
- `INVALID_EMAIL_FORMAT`: Email address format invalid
- `WEAK_PASSWORD`: Password does not meet security policy
- `INVALID_ROLE`: Role must be "admin" or "user"
- `INVALID_ACTION`: Action parameter not recognized

**Resource Errors (404):**

- `USER_NOT_FOUND`: User with specified ID does not exist
- `APIKEY_NOT_FOUND`: API key with specified ID does not exist

**Conflict Errors (409):**

- `USERNAME_EXISTS`: Username already taken
- `EMAIL_EXISTS`: Email already registered
- `APIKEY_NAME_EXISTS`: API key name already in use

**Rate Limit Errors (429):**

- `RATE_LIMIT_EXCEEDED`: Too many requests from this user/API key
- `LOGIN_ATTEMPTS_EXCEEDED`: Too many failed login attempts

### Example Error Responses

**Invalid Credentials:**

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid username or password"
  }
}
```

**Insufficient Permissions:**

```json
{
  "error": {
    "code": "INSUFFICIENT_PERMISSIONS",
    "message": "This action requires admin role"
  }
}
```

**Validation Error:**

```json
{
  "error": {
    "code": "WEAK_PASSWORD",
    "message": "Password must be at least 8 characters and include uppercase, lowercase, and number",
    "details": {
      "min_length": 8,
      "required": ["uppercase", "lowercase", "number"]
    }
  }
}
```

**Rate Limit Exceeded:**

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again in 45 seconds."
  }
}
```

## Configuration Reference

Authentication configuration is managed via the `moon.conf` YAML file following SPEC.md conventions.

### Configuration Structure

```yaml
# JWT Configuration (Required)
jwt:
  secret: "your-secret-key-min-32-chars"  # REQUIRED - cryptographically secure random string
  expiry: 3600                             # Access token expiry in seconds (default: 3600 = 1 hour)

# API Key Configuration (Optional)
apikey:
  enabled: false                           # Enable API key authentication (default: false)

# Authentication Security Settings (Optional)
auth:
  # Password policy
  password:
    min_length: 8                          # Minimum password length (default: 8)
    require_uppercase: true                # Require uppercase letter (default: true)
    require_lowercase: true                # Require lowercase letter (default: true)
    require_number: true                   # Require number (default: true)
    require_special: false                 # Require special character (default: false)
  
  # Rate limiting
  rate_limit:
    user_rpm: 100                          # Requests per minute for JWT users (default: 100)
    apikey_rpm: 1000                       # Requests per minute for API keys (default: 1000)
    login_attempts: 5                      # Failed login attempts before lockout (default: 5)
    login_window: 900                      # Login attempt window in seconds (default: 900 = 15 min)
  
  # Refresh token settings
  refresh_token:
    expiry: 604800                         # Refresh token expiry in seconds (default: 604800 = 7 days)
    max_per_user: 10                       # Max concurrent sessions per user (default: 10)
  
  # Bootstrap admin account (REMOVE AFTER FIRST STARTUP)
  bootstrap_admin:
    username: "admin"
    email: "admin@example.com"
    password: "change-me-on-first-login"   # Change immediately after first login

# CORS Configuration (Optional)
security:
  cors:
    enabled: true                          # Enable CORS (default: true)
    allowed_origins:                       # List of allowed origins
      - "https://app.example.com"
      - "http://localhost:3000"            # Development only
    allow_credentials: true                # Allow cookies/auth headers (default: true)
    max_age: 3600                          # Preflight cache in seconds (default: 3600)
```

### Configuration Validation

**Required Fields:**

- `jwt.secret`: Must be at least 32 characters for security
  - Generate with: `openssl rand -base64 32` (Linux/Mac)
  - Generate with: `[Convert]::ToBase64String((1..32|%{Get-Random -Max 256}))` (PowerShell)

**Optional Fields with Defaults:**

- All other fields have sensible defaults defined in `config.Defaults` struct
- Only specify fields you want to override

**Security Recommendations:**

- Never commit `jwt.secret` to version control
- Use strong, randomly generated secrets (min 32 chars)
- Remove `bootstrap_admin` section after first startup
- Change bootstrap admin password immediately
- Use HTTPS in production (HTTP for development only)
- Restrict CORS origins in production (no wildcards)

### Configuration Loading Order

1. Load defaults from `config.Defaults` struct
2. Load values from config file (if exists)
3. Validate required fields (jwt.secret)
4. Apply normalization (paths, prefixes)
5. Store in immutable global `AppConfig`

**Note:** Environment variables are NOT supported. All configuration must be in YAML file (SPEC.md requirement).

## Implementation Checklist

### Phase 1: Database Schema & Core Auth

- [ ] Create database migrations for `users`, `refresh_tokens`, `apikeys` tables
- [ ] Implement user model with bcrypt password hashing
- [ ] Implement refresh token model with SHA-256 hashing
- [ ] Implement API key model with SHA-256 hashing
- [ ] Add database indexes for performance
- [ ] Implement bootstrap admin account creation logic

### Phase 2: JWT Authentication

- [ ] Implement JWT token generation (access + refresh)
- [ ] Implement JWT token validation middleware
- [ ] Implement `POST /auth:login` endpoint
- [ ] Implement `POST /auth:logout` endpoint
- [ ] Implement `POST /auth:refresh` endpoint
- [ ] Implement `GET /auth:me` endpoint
- [ ] Implement `POST /auth:me` endpoint
- [ ] Add JWT rate limiting (100 req/min per user)

### Phase 3: API Key Authentication

- [ ] Implement API key generation (cryptographically secure)
- [ ] Implement API key validation middleware
- [ ] Add API key rate limiting (1000 req/min per key)
- [ ] Implement authentication priority (JWT > API key)
- [ ] Add API key last_used_at tracking

### Phase 4: User Management

- [ ] Implement `GET /users:list` endpoint
- [ ] Implement `GET /users:get` endpoint
- [ ] Implement `POST /users:create` endpoint
- [ ] Implement `POST /users:update` endpoint (role, can_write, reset_password, revoke_sessions)
- [ ] Implement `POST /users:destroy` endpoint
- [ ] Add validation: cannot delete last admin
- [ ] Add validation: cannot modify self role

### Phase 5: API Key Management

- [ ] Implement `GET /apikeys:list` endpoint
- [ ] Implement `GET /apikeys:get` endpoint
- [ ] Implement `POST /apikeys:create` endpoint
- [ ] Implement `POST /apikeys:update` endpoint (metadata, rotate)
- [ ] Implement `POST /apikeys:destroy` endpoint
- [ ] Add validation: API key format (64 chars base62)

### Phase 6: Authorization & Middleware

- [ ] Implement role-based authorization middleware
- [ ] Implement `can_write` permission checking for user role
- [ ] Add authorization checks to all collection endpoints
- [ ] Add authorization checks to all data endpoints
- [ ] Protect admin-only endpoints (users, apikeys)

### Phase 7: Security & Rate Limiting

- [ ] Implement rate limiting middleware (in-memory token bucket)
- [ ] Add rate limit headers (X-RateLimit-*)
- [ ] Implement login attempt rate limiting (5 per 15 min)
- [ ] Add CORS middleware configuration
- [ ] Implement audit logging for auth events
- [ ] Add password validation (policy enforcement)

### Phase 8: Testing

- [ ] Unit tests for JWT generation/validation
- [ ] Unit tests for API key hashing/validation
- [ ] Unit tests for password hashing/validation
- [ ] Integration tests for all auth endpoints
- [ ] Integration tests for all user management endpoints
- [ ] Integration tests for all API key management endpoints
- [ ] Authorization tests (role-based access)
- [ ] Rate limiting tests
- [ ] Security tests (invalid tokens, expired tokens, etc.)

### Phase 9: Documentation & Scripts

- [ ] Update INSTALL.md with authentication setup
- [ ] Create auth test scripts in `scripts/`
- [ ] Add curl examples for all auth endpoints
- [ ] Document configuration options
- [ ] Add security best practices guide
- [ ] Create admin user management guide

### Phase 10: Integration

- [ ] Integrate auth middleware with existing server routes
- [ ] Update server.go to include auth routes
- [ ] Update config.go to include auth configuration
- [ ] Add auth endpoints to documentation generator
- [ ] Ensure backward compatibility with existing features
- [ ] Performance testing with auth enabled

## Alignment with SPEC.md

This authentication design fully aligns with SPEC.md principles:

✅ **AIP-136 Custom Actions:** All auth endpoints use colon separator (`:action` pattern)
✅ **Configuration Architecture:** YAML-only config with centralized defaults (no env vars)
✅ **Database Schema:** Uses ULID for external IDs, auto-increment for internal
✅ **API Consistency:** Follows same error response format as data endpoints
✅ **Middleware Order:** Auth/authz integrated into existing middleware pipeline
✅ **Test-Driven Development:** Comprehensive test coverage required before implementation
✅ **Security First:** bcrypt passwords, SHA-256 API keys, rate limiting, audit logging
✅ **Single Responsibility:** Clear separation between authentication, authorization, and business logic

**No Breaking Changes:**

- Existing collection and data endpoints remain unchanged
- Authentication is opt-in (JWT secret required to enable)
- API key authentication is optional (disabled by default)
- All existing features continue to work without authentication
- Middleware can be selectively applied to specific routes

**Database Compatibility:**

- Auth tables use same conventions as existing tables
- SQLite, PostgreSQL, and MySQL all supported
- Uses dialect-agnostic SQL generation
- Indexes follow existing patterns

**Operational Compatibility:**

- Auth config follows existing YAML structure
- Logging follows existing patterns
- Health endpoint remains unchanged
- Recovery and consistency checks unaffected

---
