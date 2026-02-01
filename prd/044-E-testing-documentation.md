# PRD-044-E: Testing, Documentation, and Scripts

## Overview

Implement comprehensive testing, documentation updates, and test scripts for Moon's authentication system. This includes unit tests, integration tests, curl-based test scripts, configuration samples, and documentation updates.

**BREAKING CHANGE:** All documentation must clearly state that authentication is mandatory and cannot be disabled.

### Problem Statement

Authentication features require thorough testing to ensure security and correctness. Users need clear documentation and working examples to adopt the authentication system. Test scripts provide quick validation and serve as executable documentation.

### Dependencies

- **PRD-044-A:** Core authentication
- **PRD-044-B:** User management
- **PRD-044-C:** API key management
- **PRD-044-D:** Authorization & rate limiting

---

## Requirements

### FR-1: Test Scripts

Create bash test scripts in `scripts/` directory using curl for all auth endpoints.

**FR-1.1: scripts/auth-jwt.sh**
Tests JWT authentication flow (login, refresh, logout, auth:me).

```bash
#!/bin/bash
# Test JWT Authentication Flow

set -e
BASE_URL="${BASE_URL:-http://localhost:6006}"

echo "=== JWT Authentication Tests ==="
echo

# 1. Login
echo "1. Testing login..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth:login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "AdminPass123"
  }')

ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.refresh_token')

if [ "$ACCESS_TOKEN" = "null" ]; then
  echo "❌ Login failed"
  echo $LOGIN_RESPONSE | jq .
  exit 1
fi
echo "✅ Login successful"
echo

# 2. Access protected endpoint
echo "2. Testing protected endpoint access..."
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$BASE_URL/collections:list" > /dev/null
echo "✅ Protected endpoint accessible with JWT"
echo

# 3. Get user info
echo "3. Testing auth:me..."
ME_RESPONSE=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$BASE_URL/auth:me")
echo $ME_RESPONSE | jq .
echo "✅ Got user info"
echo

# 4. Refresh token
echo "4. Testing token refresh..."
REFRESH_RESPONSE=$(curl -s -X POST "$BASE_URL/auth:refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

NEW_ACCESS_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.access_token')
NEW_REFRESH_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.refresh_token')

if [ "$NEW_ACCESS_TOKEN" = "null" ]; then
  echo "❌ Token refresh failed"
  echo $REFRESH_RESPONSE | jq .
  exit 1
fi
echo "✅ Token refreshed successfully"
echo

# 5. Try to use old refresh token (should fail)
echo "5. Testing old refresh token rejection..."
OLD_REFRESH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth:refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

STATUS_CODE=$(echo "$OLD_REFRESH_RESPONSE" | tail -n 1)
if [ "$STATUS_CODE" = "401" ]; then
  echo "✅ Old refresh token correctly rejected"
else
  echo "❌ Old refresh token should be rejected"
  exit 1
fi
echo

# 6. Logout
echo "6. Testing logout..."
curl -s -X POST "$BASE_URL/auth:logout" \
  -H "Authorization: Bearer $NEW_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$NEW_REFRESH_TOKEN\"}" > /dev/null
echo "✅ Logout successful"
echo

# 7. Try to use logged out refresh token (should fail)
echo "7. Testing logged out refresh token rejection..."
LOGOUT_REFRESH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth:refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$NEW_REFRESH_TOKEN\"}")

STATUS_CODE=$(echo "$LOGOUT_REFRESH_RESPONSE" | tail -n 1)
if [ "$STATUS_CODE" = "401" ]; then
  echo "✅ Logged out refresh token correctly rejected"
else
  echo "❌ Logged out refresh token should be rejected"
  exit 1
fi
echo

echo "=== All JWT tests passed ✅ ==="
```

**FR-1.2: scripts/auth-apikey.sh**
Tests API key authentication and management.

```bash
#!/bin/bash
# Test API Key Authentication and Management

set -e
BASE_URL="${BASE_URL:-http://localhost:6006}"

echo "=== API Key Tests ==="
echo

# 1. Login as admin
echo "1. Logging in as admin..."
ADMIN_TOKEN=$(curl -s -X POST "$BASE_URL/auth:login" \
  -d '{"username":"admin","password":"AdminPass123"}' | jq -r '.access_token')
echo "✅ Admin login successful"
echo

# 2. Create API key
echo "2. Creating API key..."
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/apikeys:create" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Service",
    "description": "Test API key",
    "role": "user",
    "can_write": true
  }')

API_KEY=$(echo $CREATE_RESPONSE | jq -r '.key')
KEY_ID=$(echo $CREATE_RESPONSE | jq -r '.id')

if [ "$API_KEY" = "null" ]; then
  echo "❌ API key creation failed"
  echo $CREATE_RESPONSE | jq .
  exit 1
fi
echo "✅ API key created: ${API_KEY:0:20}..."
echo

# 3. Use API key for authentication
echo "3. Testing API key authentication..."
curl -s -H "X-API-Key: $API_KEY" \
  "$BASE_URL/collections:list" > /dev/null
echo "✅ API key authentication successful"
echo

# 4. List API keys
echo "4. Listing API keys..."
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$BASE_URL/apikeys:list" | jq '.apikeys[] | {id, name, role}'
echo "✅ Listed API keys"
echo

# 5. Get API key details
echo "5. Getting API key details..."
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$BASE_URL/apikeys:get?id=$KEY_ID" | jq .
echo "✅ Got API key details"
echo

# 6. Update API key metadata
echo "6. Updating API key metadata..."
curl -s -X POST "$BASE_URL/apikeys:update?id=$KEY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Test Service",
    "can_write": false
  }' > /dev/null
echo "✅ API key metadata updated"
echo

# 7. Rotate API key
echo "7. Rotating API key..."
ROTATE_RESPONSE=$(curl -s -X POST "$BASE_URL/apikeys:update?id=$KEY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "rotate"}')

NEW_API_KEY=$(echo $ROTATE_RESPONSE | jq -r '.key')
echo "✅ API key rotated: ${NEW_API_KEY:0:20}..."
echo

# 8. Verify old key doesn't work
echo "8. Testing old API key rejection..."
OLD_KEY_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -H "X-API-Key: $API_KEY" \
  "$BASE_URL/collections:list")

STATUS_CODE=$(echo "$OLD_KEY_RESPONSE" | tail -n 1)
if [ "$STATUS_CODE" = "401" ]; then
  echo "✅ Old API key correctly rejected"
else
  echo "❌ Old API key should be rejected"
  exit 1
fi
echo

# 9. Verify new key works
echo "9. Testing new API key..."
curl -s -H "X-API-Key: $NEW_API_KEY" \
  "$BASE_URL/collections:list" > /dev/null
echo "✅ New API key works"
echo

# 10. Delete API key
echo "10. Deleting API key..."
curl -s -X POST "$BASE_URL/apikeys:destroy?id=$KEY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
echo "✅ API key deleted"
echo

# 11. Verify deleted key doesn't work
echo "11. Testing deleted API key rejection..."
DELETED_KEY_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -H "X-API-Key: $NEW_API_KEY" \
  "$BASE_URL/collections:list")

STATUS_CODE=$(echo "$DELETED_KEY_RESPONSE" | tail -n 1)
if [ "$STATUS_CODE" = "401" ]; then
  echo "✅ Deleted API key correctly rejected"
else
  echo "❌ Deleted API key should be rejected"
  exit 1
fi
echo

echo "=== All API key tests passed ✅ ==="
```

**FR-1.3: scripts/auth-rbac.sh**
Tests role-based access control and permissions.

```bash
#!/bin/bash
# Test Role-Based Access Control

set -e
BASE_URL="${BASE_URL:-http://localhost:6006}"

echo "=== RBAC Tests ==="
echo

# 1. Login as admin
echo "1. Logging in as admin..."
ADMIN_TOKEN=$(curl -s -X POST "$BASE_URL/auth:login" \
  -d '{"username":"admin","password":"AdminPass123"}' | jq -r '.access_token')
echo "✅ Admin login successful"
echo

# 2. Create read-only user
echo "2. Creating read-only user..."
curl -s -X POST "$BASE_URL/users:create" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "readonly",
    "email": "readonly@example.com",
    "password": "ReadOnly123",
    "role": "user",
    "can_write": false
  }' > /dev/null
echo "✅ Read-only user created"
echo

# 3. Create read-write user
echo "3. Creating read-write user..."
curl -s -X POST "$BASE_URL/users:create" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "readwrite",
    "email": "readwrite@example.com",
    "password": "ReadWrite123",
    "role": "user",
    "can_write": true
  }' > /dev/null
echo "✅ Read-write user created"
echo

# 4. Login as read-only user
echo "4. Logging in as read-only user..."
READONLY_TOKEN=$(curl -s -X POST "$BASE_URL/auth:login" \
  -d '{"username":"readonly","password":"ReadOnly123"}' | jq -r '.access_token')
echo "✅ Read-only user login successful"
echo

# 5. Login as read-write user
echo "5. Logging in as read-write user..."
READWRITE_TOKEN=$(curl -s -X POST "$BASE_URL/auth:login" \
  -d '{"username":"readwrite","password":"ReadWrite123"}' | jq -r '.access_token')
echo "✅ Read-write user login successful"
echo

# 6. Test admin can access admin endpoint
echo "6. Testing admin access to user management..."
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$BASE_URL/users:list" > /dev/null
echo "✅ Admin can access admin endpoints"
echo

# 7. Test user cannot access admin endpoint
echo "7. Testing user cannot access user management..."
USER_ADMIN_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -H "Authorization: Bearer $READONLY_TOKEN" \
  "$BASE_URL/users:list")

STATUS_CODE=$(echo "$USER_ADMIN_RESPONSE" | tail -n 1)
if [ "$STATUS_CODE" = "403" ]; then
  echo "✅ User correctly denied access to admin endpoints"
else
  echo "❌ User should be denied access to admin endpoints"
  exit 1
fi
echo

# 8. Test read-only user can read
echo "8. Testing read-only user can read..."
curl -s -H "Authorization: Bearer $READONLY_TOKEN" \
  "$BASE_URL/collections:list" > /dev/null
echo "✅ Read-only user can read"
echo

# 9. Test read-only user cannot write
echo "9. Testing read-only user cannot write..."
READONLY_WRITE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Authorization: Bearer $READONLY_TOKEN" \
  -H "Content-Type: application/json" \
  "$BASE_URL/products:create" \
  -d '{"name":"Test Product"}')

STATUS_CODE=$(echo "$READONLY_WRITE_RESPONSE" | tail -n 1)
if [ "$STATUS_CODE" = "403" ]; then
  echo "✅ Read-only user correctly denied write access"
else
  echo "❌ Read-only user should be denied write access"
  exit 1
fi
echo

# 10. Test read-write user can write
echo "10. Testing read-write user can write..."
curl -s -X POST \
  -H "Authorization: Bearer $READWRITE_TOKEN" \
  -H "Content-Type: application/json" \
  "$BASE_URL/products:create" \
  -d '{"name":"Test Product"}' > /dev/null
echo "✅ Read-write user can write data"
echo

# 11. Test read-write user cannot manage collections
echo "11. Testing read-write user cannot manage collections..."
USER_COLLECTION_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Authorization: Bearer $READWRITE_TOKEN" \
  -H "Content-Type: application/json" \
  "$BASE_URL/collections:create" \
  -d '{"name":"test"}')

STATUS_CODE=$(echo "$USER_COLLECTION_RESPONSE" | tail -n 1)
if [ "$STATUS_CODE" = "403" ]; then
  echo "✅ User correctly denied collection management"
else
  echo "❌ User should be denied collection management"
  exit 1
fi
echo

# Cleanup
echo "Cleanup: Deleting test users..."
READONLY_ID=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$BASE_URL/users:list" | jq -r '.users[] | select(.username=="readonly") | .id')
READWRITE_ID=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$BASE_URL/users:list" | jq -r '.users[] | select(.username=="readwrite") | .id')

curl -s -X POST "$BASE_URL/users:destroy?id=$READONLY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
curl -s -X POST "$BASE_URL/users:destroy?id=$READWRITE_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null

echo "✅ Cleanup complete"
echo

echo "=== All RBAC tests passed ✅ ==="
```

**FR-1.4: scripts/auth-ratelimit.sh**
Tests rate limiting functionality.

```bash
#!/bin/bash
# Test Rate Limiting

set -e
BASE_URL="${BASE_URL:-http://localhost:6006}"

echo "=== Rate Limit Tests ==="
echo

# 1. Login as user
echo "1. Logging in as user..."
USER_TOKEN=$(curl -s -X POST "$BASE_URL/auth:login" \
  -d '{"username":"user1","password":"UserPass123"}' | jq -r '.access_token')
echo "✅ User login successful"
echo

# 2. Test user rate limit (100 req/min)
echo "2. Testing user rate limit (100 req/min)..."
echo "   Making 101 requests..."

SUCCESS_COUNT=0
RATE_LIMITED=false

for i in {1..101}; do
  RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $USER_TOKEN" \
    "$BASE_URL/collections:list" 2>&1)
  
  STATUS_CODE=$(echo "$RESPONSE" | tail -n 1)
  
  if [ "$STATUS_CODE" = "200" ]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  elif [ "$STATUS_CODE" = "429" ]; then
    RATE_LIMITED=true
    break
  fi
  
  # Small delay to avoid overwhelming server
  sleep 0.01
done

if [ $SUCCESS_COUNT -le 100 ] && [ "$RATE_LIMITED" = true ]; then
  echo "✅ User rate limited after ~$SUCCESS_COUNT requests"
else
  echo "❌ User should be rate limited after 100 requests"
  echo "   Success count: $SUCCESS_COUNT, Rate limited: $RATE_LIMITED"
  exit 1
fi
echo

# 3. Check rate limit headers
echo "3. Checking rate limit headers..."
HEADERS=$(curl -s -i -H "Authorization: Bearer $USER_TOKEN" \
  "$BASE_URL/collections:list")

if echo "$HEADERS" | grep -q "X-RateLimit-Limit: 100" && \
   echo "$HEADERS" | grep -q "X-RateLimit-Remaining:" && \
   echo "$HEADERS" | grep -q "X-RateLimit-Reset:"; then
  echo "✅ Rate limit headers present"
  echo "$HEADERS" | grep "X-RateLimit"
else
  echo "❌ Rate limit headers missing"
  exit 1
fi
echo

# 4. Test login rate limiting
echo "4. Testing login rate limiting (5 attempts per 15 min)..."
echo "   Making 6 failed login attempts..."

for i in {1..6}; do
  RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    "$BASE_URL/auth:login" \
    -d '{"username":"testuser","password":"wrong"}' 2>&1)
  
  STATUS_CODE=$(echo "$RESPONSE" | tail -n 1)
  
  if [ $i -le 5 ] && [ "$STATUS_CODE" = "401" ]; then
    echo "   Attempt $i: 401 (expected)"
  elif [ $i -eq 6 ] && [ "$STATUS_CODE" = "429" ]; then
    echo "   Attempt $i: 429 (rate limited - expected)"
    echo "✅ Login rate limiting works"
  else
    echo "❌ Unexpected status code on attempt $i: $STATUS_CODE"
    exit 1
  fi
done
echo

echo "=== All rate limit tests passed ✅ ==="
echo
echo "Note: Wait 1 minute for rate limits to reset before running again"
```

**FR-1.5: scripts/auth-all.sh**
Master test runner that executes all auth tests.

```bash
#!/bin/bash
# Run All Authentication Tests

set -e

echo "╔══════════════════════════════════════════════════════╗"
echo "║   Moon Authentication Test Suite                   ║"
echo "╚══════════════════════════════════════════════════════╝"
echo

# Check if server is running
BASE_URL="${BASE_URL:-http://localhost:6006}"
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
  echo "❌ Server not running at $BASE_URL"
  echo "   Start the server with: ./moon --config samples/moon.conf"
  exit 1
fi

echo "✅ Server is running"
echo

# Run JWT tests
echo "╔══════════════════════════════════════════════════════╗"
echo "║   JWT Authentication Tests                         ║"
echo "╚══════════════════════════════════════════════════════╝"
./scripts/auth-jwt.sh
echo

# Run API Key tests
echo "╔══════════════════════════════════════════════════════╗"
echo "║   API Key Tests                                     ║"
echo "╚══════════════════════════════════════════════════════╝"
./scripts/auth-apikey.sh
echo

# Run RBAC tests
echo "╔══════════════════════════════════════════════════════╗"
echo "║   RBAC Tests                                        ║"
echo "╚══════════════════════════════════════════════════════╝"
./scripts/auth-rbac.sh
echo

# Run Rate Limit tests
echo "╔══════════════════════════════════════════════════════╗"
echo "║   Rate Limiting Tests                               ║"
echo "╚══════════════════════════════════════════════════════╝"
./scripts/auth-ratelimit.sh
echo

echo "╔══════════════════════════════════════════════════════╗"
echo "║   All Tests Passed ✅                               ║"
echo "╚══════════════════════════════════════════════════════╝"
```

---

### FR-2: Configuration Sample Updates

**FR-2.1: Update samples/moon.conf**
```yaml
# Moon - Dynamic Headless Engine Configuration
# BREAKING CHANGE: Authentication is mandatory in this version

server:
  host: "0.0.0.0"
  port: 6006
  prefix: ""  # Optional URL prefix (e.g., "/api/v1")

database:
  connection: "sqlite"
  database: "/opt/moon/sqlite.db"

logging:
  path: "/var/log/moon"

# JWT Configuration (REQUIRED)
jwt:
  # Generate a secure secret with: openssl rand -base64 32
  secret: "CHANGE-THIS-SECRET-KEY-MIN-32-CHARS-REQUIRED-FOR-PRODUCTION"
  access_expiry: 3600      # Access token expiry in seconds (1 hour)
  refresh_expiry: 604800   # Refresh token expiry in seconds (7 days)

# API Key Configuration (Optional)
apikey:
  enabled: false           # Enable API key authentication
  header: "X-API-Key"      # HTTP header name for API keys

# Bootstrap Admin Account (REQUIRED on first startup)
# Remove this section after first startup and change the password immediately
auth:
  bootstrap_admin:
    username: "admin"
    email: "admin@example.com"
    password: "ChangeMe123!FirstLogin"
  
  # Rate Limiting Configuration
  rate_limit:
    user_rpm: 100          # Requests per minute for JWT users
    apikey_rpm: 1000       # Requests per minute for API keys
    login_attempts: 5      # Failed login attempts before lockout
    login_window: 900      # Login window in seconds (15 minutes)

# CORS Configuration (Optional)
security:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"   # Development
      - "https://app.example.com" # Production
    allow_credentials: true
    max_age: 3600

# Recovery Configuration
recovery:
  auto_repair: true
  drop_orphans: false
  check_timeout: 5
```

---

### FR-3: Documentation Updates

**FR-3.1: Update SPEC.md - Add Authentication Section**

Add new section after "Configuration Architecture":

```markdown
## Authentication & Authorization

Moon requires authentication for all API operations (except `/health` and `/doc/*` endpoints). Authentication cannot be disabled.

### Authentication Methods

**JWT Authentication (for users):**
- Access tokens: Short-lived (default 1 hour)
- Refresh tokens: Long-lived (default 7 days), single-use
- Header: `Authorization: Bearer <access_token>`

**API Key Authentication (for services):**
- Long-lived, manually rotated credentials
- Format: `moon_live_` prefix + 64 characters
- Header: `X-API-Key: <api_key>`

### Roles

- **admin:** Full system access (user management, API key management, collection management, data access)
- **user:** Data access only (read-only by default, write access via `can_write` flag)

### Protected Endpoints

- **Public:** `/health`, `/doc/*`
- **Auth Endpoints:** `/auth:login`, `/auth:refresh` (no auth required)
- **Authenticated:** All data, collection, and aggregation endpoints
- **Admin Only:** `/users:*`, `/apikeys:*`, `/collections:create`, `/collections:update`, `/collections:destroy`

### Rate Limits

- JWT users: 100 requests/minute
- API keys: 1000 requests/minute
- Login attempts: 5 per 15 minutes per IP/username

See [SPEC_AUTH.md](SPEC_AUTH.md) for complete authentication specification.
```

**FR-3.2: Update INSTALL.md - Add Authentication Setup**

Add new section "Authentication Setup":

```markdown
## Authentication Setup

### BREAKING CHANGE

Authentication is **mandatory** in Moon. All API endpoints (except `/health` and `/doc/*`) require authentication.

### First-Time Setup

1. **Generate JWT Secret:**
   ```bash
   # Linux/Mac
   openssl rand -base64 32
   
   # Windows (PowerShell)
   [Convert]::ToBase64String((1..32|%{Get-Random -Max 256}))
   ```

2. **Configure Bootstrap Admin:**
   Edit `/etc/moon.conf`:
   ```yaml
   jwt:
     secret: "YOUR-GENERATED-SECRET-HERE"
   
   auth:
     bootstrap_admin:
       username: "admin"
       email: "admin@example.com"
       password: "ChangeMe123!FirstLogin"
   ```

3. **Start Moon:**
   ```bash
   sudo systemctl start moon
   ```

4. **Verify Bootstrap:**
   Check logs:
   ```bash
   sudo journalctl -u moon -n 50
   # Look for: "Bootstrap admin created: admin@example.com"
   ```

5. **Login and Change Password:**
   ```bash
   # Login
   TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
     -d '{"username":"admin","password":"ChangeMe123!FirstLogin"}' | jq -r '.access_token')
   
   # Change password
   curl -X POST http://localhost:6006/auth:me \
     -H "Authorization: Bearer $TOKEN" \
     -d '{
       "current_password": "ChangeMe123!FirstLogin",
       "new_password": "MyNewSecurePassword456!"
     }'
   ```

6. **Remove Bootstrap Config:**
   Edit `/etc/moon.conf` and remove the `auth.bootstrap_admin` section.
   Restart Moon: `sudo systemctl restart moon`

### Creating Additional Users

```bash
# Login as admin
ADMIN_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"YourAdminPassword"}' | jq -r '.access_token')

# Create user
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "username": "newuser",
    "email": "user@example.com",
    "password": "SecurePass123",
    "role": "user",
    "can_write": false
  }'
```

### Creating API Keys

```bash
# Create API key for service integration
curl -X POST http://localhost:6006/apikeys:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "My Service",
    "description": "Production API integration",
    "role": "user",
    "can_write": true
  }'

# Save the returned key securely - it won't be shown again!
```

### Testing Authentication

Run the test suite:
```bash
cd /opt/moon
./scripts/auth-all.sh
```
```

**FR-3.3: Update README.md - Add Authentication Notice**

Add prominent notice at top of README:

```markdown
## ⚠️ Breaking Change: Authentication Required

**Version 2.0+** requires authentication for all API operations. There is no backward compatibility mode.

- All endpoints (except `/health` and `/doc/*`) require valid JWT or API key
- Bootstrap admin account must be configured on first startup
- See [INSTALL.md](INSTALL.md) for authentication setup guide
- See [SPEC_AUTH.md](SPEC_AUTH.md) for complete authentication specification

### Quick Start with Authentication

1. Generate JWT secret: `openssl rand -base64 32`
2. Configure bootstrap admin in `samples/moon.conf`
3. Start Moon: `./moon --config samples/moon.conf`
4. Login: `curl -X POST http://localhost:6006/auth:login -d '{"username":"admin","password":"..."}'`
5. Use returned token: `curl -H "Authorization: Bearer <token>" http://localhost:6006/collections:list`
```

---

### FR-4: Unit and Integration Tests

**FR-4.1: Unit Test Coverage Requirements**
- Password hashing: 100%
- JWT token generation/validation: 100%
- API key generation/hashing: 100%
- Token bucket algorithm: 100%
- Authorization logic: 100%

**FR-4.2: Integration Test Requirements**
- All auth endpoints: 100%
- All user management endpoints: 100%
- All API key management endpoints: 100%
- Authorization enforcement: 100%
- Rate limiting: 100%

**FR-4.3: Test Organization**
```
cmd/moon/internal/
├── auth/
│   ├── auth.go
│   ├── auth_test.go
│   ├── jwt.go
│   ├── jwt_test.go
│   ├── password.go
│   └── password_test.go
├── handlers/
│   ├── auth_handler.go
│   ├── auth_handler_test.go
│   ├── users_handler.go
│   ├── users_handler_test.go
│   ├── apikeys_handler.go
│   └── apikeys_handler_test.go
└── middleware/
    ├── authentication.go
    ├── authentication_test.go
    ├── authorization.go
    ├── authorization_test.go
    ├── ratelimit.go
    └── ratelimit_test.go
```

---

## Acceptance Criteria

### AC-1: Test Scripts Work

**Verification:**
- [ ] All test scripts execute without errors
- [ ] Scripts detect server unavailability
- [ ] Scripts clean up test data
- [ ] Scripts provide clear output

**Test:**
```bash
# Start server
./moon --config samples/moon.conf &
sleep 2

# Run all tests
./scripts/auth-all.sh

# Expected: All tests pass with ✅ symbols
```

### AC-2: Configuration Sample Valid

**Verification:**
- [ ] Sample config includes all auth fields
- [ ] Comments explain each field
- [ ] Breaking change warning present
- [ ] Bootstrap admin example provided

**Test:**
```bash
# Validate config syntax
./moon --config samples/moon.conf --validate

# Start with sample config
./moon --config samples/moon.conf

# Check logs for bootstrap
grep "Bootstrap admin created" /var/log/moon/main.log
```

### AC-3: Documentation Complete

**Verification:**
- [ ] SPEC.md includes authentication section
- [ ] INSTALL.md includes setup guide
- [ ] README.md includes breaking change notice
- [ ] SPEC_AUTH.md referenced in all docs

**Test:**
Manual review of documentation files.

### AC-4: Test Coverage Adequate

**Verification:**
- [ ] Unit test coverage ≥ 90%
- [ ] Integration tests cover all endpoints
- [ ] Tests pass on all database backends

**Test:**
```bash
# Run tests with coverage
go test ./cmd/moon/internal/... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
# Expected: total (statements) ≥ 90%
```

---

## Implementation Checklist

- [ ] Create `scripts/auth-jwt.sh` test script
- [ ] Create `scripts/auth-apikey.sh` test script
- [ ] Create `scripts/auth-rbac.sh` test script
- [ ] Create `scripts/auth-ratelimit.sh` test script
- [ ] Create `scripts/auth-all.sh` master runner
- [ ] Make all scripts executable (`chmod +x scripts/auth-*.sh`)
- [ ] Update `samples/moon.conf` with auth configuration
- [ ] Add authentication section to SPEC.md
- [ ] Add authentication setup to INSTALL.md
- [ ] Add breaking change notice to README.md
- [ ] Write unit tests for all auth functions
- [ ] Write integration tests for all auth endpoints
- [ ] Write integration tests for authorization
- [ ] Write integration tests for rate limiting
- [ ] Achieve ≥90% test coverage
- [ ] Test on SQLite, PostgreSQL, MySQL
- [ ] Document test running procedure
- [ ] Create BREAKING_CHANGES.md document

---

## Related PRDs

- [PRD-044: Authentication System](044-authentication-system.md) - Parent PRD
- [PRD-044-A: Core Authentication](044-A-core-authentication.md)
- [PRD-044-B: User Management](044-B-user-management.md)
- [PRD-044-C: API Key Management](044-C-apikey-management.md)
- [PRD-044-D: Authorization & Rate Limiting](044-D-authorization-ratelimit.md)

---

## References

- [SPEC_AUTH.md](../SPEC_AUTH.md) - Complete authentication specification
- [SPEC.md](../SPEC.md) - System architecture
- [INSTALL.md](../INSTALL.md) - Installation guide
