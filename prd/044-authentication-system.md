# PRD-044: Authentication System

## Overview

Implement a comprehensive authentication and authorization system for Moon that provides JWT-based user authentication and API key authentication for machine-to-machine integrations. The system supports role-based access control (RBAC) with two roles (`admin` and `user`) and configurable write permissions.

This PRD provides a high-level overview. Implementation is split into five self-contained tasks:
- **PRD-044-A:** Core Authentication (JWT + API Key, Database Schema, Bootstrap Admin)
- **PRD-044-B:** User Management Endpoints (Admin Operations)
- **PRD-044-C:** API Key Management Endpoints (Admin Operations)
- **PRD-044-D:** Authorization Middleware & Rate Limiting
- **PRD-044-E:** Testing, Documentation, and Scripts

### Problem Statement

Moon currently has no authentication or authorization mechanism. All API endpoints are publicly accessible, making it unsuitable for production deployments where:
- User identity and access control are required
- Different users need different permission levels
- Service integrations need stable, long-lived credentials
- Rate limiting per user/service is necessary
- Audit trails for security-sensitive operations are needed

### Solution

Implement a two-tiered authentication system:
1. **JWT Authentication:** For interactive users via web/mobile clients
2. **API Key Authentication:** For machine-to-machine integrations

Both methods support RBAC with:
- **admin role:** Full system access
- **user role:** Read access by default, optional write access via `can_write` flag

**BREAKING CHANGE:** Authentication is **mandatory** and **always enabled**. All endpoints require authentication (except `/health` and `/doc/*`). There is no backward compatibility mode.

### Scope

**In Scope:**
- JWT-based user authentication with access and refresh tokens
- API key authentication for services
- Role-based access control (admin/user)
- Per-user write permission control (`can_write` flag)
- Bootstrap admin account creation
- User management endpoints (admin only)
- API key management endpoints (admin only)
- Authentication and authorization middleware
- Rate limiting per user/API key
- Audit logging for auth events
- Test scripts for all auth endpoints
- Documentation updates
- **BREAKING CHANGE:** Mandatory authentication on all endpoints (no opt-out)

**Out of Scope:**
- Multi-factor authentication (MFA)
- OAuth2/OIDC integration
- Email verification
- Self-service password reset
- Social login
- Session management UI
- Advanced permission models (ACLs, ABAC)
- Backward compatibility with unauthenticated access

---

## Requirements

### FR-1: Configuration

**FR-1.1: JWT Configuration (Required)**
- Configuration fields:
  - `jwt.secret` (string, required): Signing secret (min 32 chars)
  - `jwt.access_expiry` (int, default: 3600): Access token TTL in seconds
  - `jwt.refresh_expiry` (int, default: 604800): Refresh token TTL in seconds
- Server will not start without valid JWT configuration

**FR-1.2: API Key Configuration**
- Optional authentication method (disabled by default)
- Configuration fields:
  - `apikey.enabled` (bool, default: false): Enable API key auth
  - `apikey.header` (string, default: "X-API-Key"): HTTP header name

**FR-1.3: Rate Limiting Configuration**
- Configuration fields:
  - `auth.rate_limit.user_rpm` (int, default: 100): Requests/min for JWT users
  - `auth.rate_limit.apikey_rpm` (int, default: 1000): Requests/min for API keys
  - `auth.rate_limit.login_attempts` (int, default: 5): Failed login attempts
  - `auth.rate_limit.login_window` (int, default: 900): Login window in seconds

**FR-1.5: Bootstrap Admin**
- Configuration fields (optional):
  ```yaml
  auth:
    bootstrap_admin:
      username: "admin"
      email: "admin@example.com"
      password: "change-me"
  ```
- If no admin exists and bootstrap config present, create admin on startup
- If admin already exists, skip bootstrap
- If no admin and no bootstrap config, log warning

### FR-2: Database Schema

**FR-2.1: Users Table**
- Fields: `id` (int), `ulid` (text), `username` (text), `email` (text), `password_hash` (text), `role` (text), `can_write` (boolean), `created_at`, `updated_at`, `last_login_at`
- Indexes: `ulid`, `username`, `email`
- Password hashing: bcrypt (cost factor 12)

**FR-2.2: Refresh Tokens Table**
- Fields: `id` (int), `user_id` (int FK), `token_hash` (text), `expires_at`, `created_at`, `last_used_at`
- Indexes: `user_id`, `token_hash`, `expires_at`
- Foreign key: `user_id` references `users(id)` with CASCADE delete
- Token storage: SHA-256 hash

**FR-2.3: API Keys Table**
- Fields: `id` (int), `ulid` (text), `name` (text), `description` (text), `key_hash` (text), `role` (text), `can_write` (boolean), `created_at`, `last_used_at`
- Indexes: `ulid`, `key_hash`
- Key storage: SHA-256 hash
- Key format: `moon_live_` prefix + 64 chars base62

### FR-3: Authentication Endpoints

All endpoints follow AIP-136 custom actions pattern.

**FR-3.1: POST /auth:login**
- Input: `{"username": "...", "password": "..."}`
- Output: `{"access_token": "...", "refresh_token": "...", "expires_in": 3600, "token_type": "Bearer", "user": {...}}`
- Updates `last_login_at` timestamp
- Creates refresh token in database
- Rate limited: 5 attempts per 15 minutes

**FR-3.2: POST /auth:logout**
- Requires: `Authorization: Bearer <token>` header
- Input: `{"refresh_token": "..."}`
- Output: `{"message": "Logged out successfully"}`
- Invalidates refresh token

**FR-3.3: POST /auth:refresh**
- Input: `{"refresh_token": "..."}`
- Output: `{"access_token": "...", "refresh_token": "...", "expires_in": 3600, "token_type": "Bearer"}`
- Validates and invalidates old refresh token
- Issues new access + refresh token pair
- Updates `last_used_at` timestamp

**FR-3.4: GET /auth:me**
- Requires: `Authorization: Bearer <token>` header
- Output: Current user info (excluding password_hash)

**FR-3.5: POST /auth:me**
- Requires: `Authorization: Bearer <token>` header
- Input: Update email OR change password
- Email update: `{"email": "new@example.com"}`
- Password change: `{"current_password": "...", "new_password": "..."}`
- Password change invalidates all refresh tokens (force re-login)

### FR-4: User Management Endpoints (Admin Only)

**FR-4.1: GET /users:list**
- Query params: `limit`, `after` (cursor), `role` (filter)
- Returns: List of users with pagination

**FR-4.2: GET /users:get**
- Query param: `id` (user ULID)
- Returns: Single user details

**FR-4.3: POST /users:create**
- Input: `{"username": "...", "email": "...", "password": "...", "role": "user|admin", "can_write": false}`
- Returns: Created user (excluding password)
- No self-registration (admin only)

**FR-4.4: POST /users:update**
- Query param: `id` (user ULID)
- Input: Update role/permissions OR admin actions
- Update: `{"role": "...", "can_write": true}`
- Reset password: `{"action": "reset_password", "new_password": "..."}`
- Revoke sessions: `{"action": "revoke_sessions"}`
- Validates: Cannot delete last admin

**FR-4.5: POST /users:destroy**
- Query param: `id` (user ULID)
- Deletes user and cascades to refresh tokens
- Validates: Cannot delete last admin

### FR-5: API Key Management Endpoints (Admin Only)

**FR-5.1: GET /apikeys:list**
- Query params: `limit`, `after` (cursor)
- Returns: List of API key metadata (NOT actual keys)

**FR-5.2: GET /apikeys:get**
- Query param: `id` (key ULID)
- Returns: API key metadata (NOT actual key)

**FR-5.3: POST /apikeys:create**
- Input: `{"name": "...", "description": "...", "role": "user|admin", "can_write": false}`
- Returns: Created key metadata + actual key (only time key is shown)
- Generates cryptographically secure 64-char key
- Stores SHA-256 hash

**FR-5.4: POST /apikeys:update**
- Query param: `id` (key ULID)
- Update metadata: `{"name": "...", "description": "...", "can_write": true}`
- Rotate key: `{"action": "rotate"}` (returns new key once)

**FR-5.5: POST /apikeys:destroy**
- Query param: `id` (key ULID)
- Immediately invalidates key

### FR-6: Authorization & Middleware

**FR-6.1: Authentication Middleware**
- Check for JWT (`Authorization: Bearer <token>`) OR API key (`X-API-Key: <key>`)
- JWT takes precedence if both present
- Extract and validate credentials
- Add user/key info to request context
- Return 401 on invalid/missing credentials

**FR-6.2: Authorization Middleware**
- Read user/key info from context
- Check required role for endpoint
- Check `can_write` permission for data modification endpoints
- Return 403 on insufficient permissions

**FR-6.3: Rate Limiting Middleware**
- Track requests per user ULID or API key ULID
- Enforce per-entity rate limits
- Add rate limit headers to responses:
  - `X-RateLimit-Limit`
  - `X-RateLimit-Remaining`
  - `X-RateLimit-Reset`
- Return 429 when limit exceeded

**FR-6.4: Protected Endpoints**
- **Public (No Auth):** `/health`, `/doc/*` only
- **Public (Auth Required):** `/auth:login`, `/auth:refresh`
- **Authenticated (Any Role):** `/auth:logout`, `/auth:me`, all data/collection read endpoints
- **Admin Only:** `/users:*`, `/apikeys:*`, `/collections:create`, `/collections:update`, `/collections:destroy`
- **User (can_write: false):** Read-only access to collections/data
- **User (can_write: true):** Read + write access to data (cannot manage collections)

### FR-7: Audit Logging

**FR-7.1: Authentication Events**
- Log all login attempts (success/failure)
- Log all token refresh attempts
- Log all logout events
- Include: timestamp, username/key_id, IP address, user agent, outcome

**FR-7.2: Admin Actions**
- Log all user management operations
- Log all API key management operations
- Include: timestamp, admin user, action, target resource, outcome

**FR-7.3: Rate Limit Violations**
- Log all rate limit exceeded events
- Include: timestamp, entity ID, endpoint, attempt count

**FR-7.4: Log Format**
- Use structured logging (key-value pairs)
- Never log passwords or tokens
- Log at appropriate levels (INFO for success, WARN for rate limits, ERROR for auth failures)

### FR-8: Error Handling

**FR-8.1: Error Response Format**
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message"
  }
}
```

**FR-8.2: Error Codes**
- `MISSING_AUTH_HEADER` (401)
- `INVALID_TOKEN` (401)
- `EXPIRED_TOKEN` (401)
- `INVALID_CREDENTIALS` (401)
- `INSUFFICIENT_PERMISSIONS` (403)
- `ADMIN_REQUIRED` (403)
- `WRITE_PERMISSION_REQUIRED` (403)
- `RATE_LIMIT_EXCEEDED` (429)
- `WEAK_PASSWORD` (400)
- `USERNAME_EXISTS` (409)
- `EMAIL_EXISTS` (409)

---

## Acceptance Criteria

### AC-1: Mandatory Authentication

**Verification:**
- [ ] Server requires `jwt.secret` to start
- [ ] Server requires bootstrap admin config on first run
- [ ] Protected endpoints return 401 without auth
- [ ] Public endpoints accessible without auth: `/health`, `/doc/*`

**Test:**
```bash
# Start without JWT secret
./moon --config moon-no-secret.conf
# Expected: Error "jwt.secret is required"

# Start with valid config
./moon --config moon.conf
# Expected: Success

# Try to access protected endpoint without auth
curl http://localhost:6006/collections:list
# Expected: 401 Unauthorized

# Access public endpoint
curl http://localhost:6006/health
# Expected: 200 OK
```

### AC-2: Bootstrap Admin Creation

**Verification:**
- [ ] Bootstrap admin created on first startup if config present
- [ ] Bootstrap skipped if admin already exists
- [ ] Warning logged if no admin and no bootstrap config

**Test:**
```bash
# First startup with bootstrap config
./moon --config moon-bootstrap.conf
# Log: "Bootstrap admin created: admin@example.com"

# Second startup
./moon --config moon-bootstrap.conf
# Log: "Admin user already exists, skipping bootstrap"
```

### AC-3: JWT Authentication Flow

**Verification:**
- [ ] Login with valid credentials returns tokens
- [ ] Access token grants access to protected endpoints
- [ ] Refresh token can be exchanged for new token pair
- [ ] Logout invalidates refresh token
- [ ] Password change invalidates all refresh tokens

**Test:**
See `scripts/auth-jwt.sh` test script

### AC-4: API Key Authentication

**Verification:**
- [ ] Admin can create API keys
- [ ] API key grants access based on role
- [ ] API key can be rotated
- [ ] API key can be destroyed
- [ ] Key value shown only once

**Test:**
See `scripts/auth-apikey.sh` test script

### AC-5: Role-Based Access Control

**Verification:**
- [ ] Admin can access all endpoints
- [ ] User (can_write: false) can read but not write
- [ ] User (can_write: true) can read and write data
- [ ] User cannot manage collections
- [ ] User cannot manage other users/keys

**Test:**
See `scripts/auth-rbac.sh` test script

### AC-6: Rate Limiting

**Verification:**
- [ ] JWT users limited to 100 req/min
- [ ] API keys limited to 1000 req/min
- [ ] Login attempts limited to 5 per 15 min
- [ ] 429 returned when limit exceeded
- [ ] Rate limit headers present in responses

**Test:**
See `scripts/auth-ratelimit.sh` test script

### AC-7: Audit Logging

**Verification:**
- [ ] All login attempts logged
- [ ] All admin actions logged
- [ ] All rate limit violations logged
- [ ] No passwords or tokens in logs
- [ ] Structured log format

**Test:**
```bash
# Trigger various auth events
# Check logs contain expected entries
grep "AUTH_LOGIN" /var/log/moon/main.log
grep "ADMIN_ACTION" /var/log/moon/main.log
grep "RATE_LIMIT" /var/log/moon/main.log
```

### AC-8: Error Handling

**Verification:**
- [ ] All error responses follow standard format
- [ ] Error codes match specification
- [ ] Error messages are clear and actionable
- [ ] No stack traces or internal details exposed

**Test:**
```bash
# Test various error conditions
curl -X POST http://localhost:6006/auth:login \
  -d '{"username":"wrong","password":"wrong"}'
# Returns: {"error":{"code":"INVALID_CREDENTIALS","message":"..."}}
```

### AC-9: Documentation

**Verification:**
- [ ] SPEC.md updated with auth configuration
- [ ] INSTALL.md includes auth setup instructions
- [ ] README.md references auth documentation
- [ ] samples/moon.conf includes auth config with comments
- [ ] API doc endpoint includes auth information

### AC-10: Breaking Changes Acknowledged

**Verification:**
- [ ] All endpoints (except `/health`, `/doc/*`) require authentication
- [ ] Bootstrap admin must be created on first startup
- [ ] API clients must be updated to include authentication
- [ ] Documentation clearly states breaking change

**Test:**
```bash
# Verify all protected endpoints return 401
curl http://localhost:6006/collections:list
# Expected: 401

curl http://localhost:6006/products:list
# Expected: 401

# Verify public endpoints work
curl http://localhost:6006/health
# Expected: 200

curl http://localhost:6006/doc/
# Expected: 200
```

---

## Implementation Checklist

This PRD is implemented across five sub-tasks:

### PRD-044-A: Core Authentication
- [ ] Database schema (users, refresh_tokens, apikeys)
- [ ] JWT token generation and validation
- [ ] API key generation and validation
- [ ] Bootstrap admin creation
- [ ] Authentication middleware
- [ ] Configuration loading and validation
- [ ] Unit tests for auth logic

### PRD-044-B: User Management
- [ ] POST /users:create endpoint
- [ ] GET /users:list endpoint
- [ ] GET /users:get endpoint
- [ ] POST /users:update endpoint (with admin actions)
- [ ] POST /users:destroy endpoint
- [ ] User management unit and integration tests

### PRD-044-C: API Key Management
- [ ] POST /apikeys:create endpoint
- [ ] GET /apikeys:list endpoint
- [ ] GET /apikeys:get endpoint
- [ ] POST /apikeys:update endpoint (with key rotation)
- [ ] POST /apikeys:destroy endpoint
- [ ] API key management unit and integration tests

### PRD-044-D: Authorization & Rate Limiting
- [ ] Authorization middleware
- [ ] Role checking logic
- [ ] can_write permission enforcement
- [ ] Rate limiting middleware (in-memory)
- [ ] Rate limit headers
- [ ] Audit logging
- [ ] Authorization and rate limit tests

### PRD-044-E: Testing & Documentation
- [ ] Test script: scripts/auth-jwt.sh
- [ ] Test script: scripts/auth-apikey.sh
- [ ] Test script: scripts/auth-rbac.sh
- [ ] Test script: scripts/auth-ratelimit.sh
- [ ] Update SPEC.md
- [ ] Update INSTALL.md
- [ ] Update samples/moon.conf
- [ ] Update API documentation template
- [ ] Integration testing across all features

---

## Related PRDs

- [PRD-002: Configuration Loader](002-configuration-loader.md) - Configuration architecture
- [PRD-009: JWT Authentication](009-jwt-authentication.md) - (Superseded by this PRD)
- [PRD-010: API Key Authentication](010-api-key-authentication.md) - (Superseded by this PRD)
- [PRD-017: Configuration YAML Only](017-configuration-yaml-only.md) - YAML-only config approach

---

## References

- [SPEC_AUTH.md](../SPEC_AUTH.md) - Complete authentication specification
- [SPEC.md](../SPEC.md) - System architecture and conventions
- [AGENTS.md](../AGENTS.md) - Development principles and rules
