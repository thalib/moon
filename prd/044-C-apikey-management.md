# PRD-044-C: API Key Management Endpoints

## Overview

Implement API key management endpoints for Moon's authentication system. These endpoints allow administrators to create, list, retrieve, update (including rotation), and delete API keys for machine-to-machine authentication.

**Admin-Only Access:** All API key management endpoints require admin role. API keys provide an alternative authentication method to JWT for long-lived service integrations.

### Problem Statement

Services and automated systems need long-lived authentication credentials that don't expire like JWT tokens. Administrators must be able to manage these credentials, rotate them for security, and revoke them when compromised or no longer needed.

### Dependencies

- **PRD-044-A:** Core authentication (apikeys table, API key generation/hashing, auth middleware)
- Existing: `database` package, `ulid` package, `validation` package

---

## Requirements

### FR-1: API Key Management Endpoints

All endpoints require admin authentication (`Authorization: Bearer <admin_token>`).

**FR-1.1: GET /apikeys:list**
```
GET /apikeys:list?limit=50&after=01ARZ...
```

**Request:**
- Query params:
  - `limit` (optional, default: 50, max: 100): Results per page
  - `after` (optional): Cursor for pagination (API key ULID)

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
      "created_at": "2026-01-10T10:00:00Z",
      "last_used_at": "2026-02-01T14:30:00Z"
    },
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FBX",
      "name": "Analytics Service",
      "description": "Read-only analytics integration",
      "role": "user",
      "can_write": false,
      "created_at": "2026-01-12T11:00:00Z",
      "last_used_at": "2026-02-01T15:00:00Z"
    }
  ],
  "next_cursor": "01ARZ3NDEKTSV4RRFFQ69G5FBX"
}
```

**Notes:**
- Actual API key value is **never** returned (only metadata)
- Sorted by `created_at DESC`
- `last_used_at` updated by authentication middleware

**Authorization:**
- Requires admin role
- Returns 403 for non-admin users

---

**FR-1.2: GET /apikeys:get**
```
GET /apikeys:get?id=01ARZ3NDEKTSV4RRFFQ69G5FAV
```

**Request:**
- Query param:
  - `id` (required): API key ULID

**Response (200 OK):**
```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "name": "Production Service",
  "description": "Main API integration",
  "role": "user",
  "can_write": true,
  "created_at": "2026-01-10T10:00:00Z",
  "last_used_at": "2026-02-01T14:30:00Z"
}
```

**Notes:**
- Returns metadata only (not actual key)
- Key value is never retrievable after creation

**Error Responses:**
- `400 Bad Request`: Missing or invalid `id` parameter
- `403 Forbidden`: Not an admin
- `404 Not Found`: API key does not exist

---

**FR-1.3: POST /apikeys:create**
```
POST /apikeys:create
Content-Type: application/json
```

**Request:**
```json
{
  "name": "New Service",
  "description": "Optional description of the service",
  "role": "user",
  "can_write": false
}
```

**Response (201 Created):**
```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "name": "New Service",
  "description": "Optional description of the service",
  "role": "user",
  "can_write": false,
  "key": "moon_live_abc123def456ghi789jkl012mno345pqr678stu901vwx234yz567ABC890DEF",
  "created_at": "2026-02-01T16:30:00Z",
  "warning": "Store this key securely. It will not be shown again."
}
```

**Validation:**
- `name`: Required, 3-100 chars, unique
- `description`: Optional, max 500 chars
- `role`: Required, must be "admin" or "user"
- `can_write`: Optional, boolean, default false

**Business Rules:**
- API key format: `moon_live_` prefix + 64 chars (base62)
- Key generated using `crypto/rand` (cryptographically secure)
- Key stored as SHA-256 hash
- Actual key returned **only once** in creation response
- ULID auto-generated for API key
- Key must be unique (extremely unlikely collision)

**Error Responses:**
- `400 Bad Request`: Validation errors
- `403 Forbidden`: Not an admin
- `409 Conflict`: Name already exists

---

**FR-1.4: POST /apikeys:update**
```
POST /apikeys:update?id=01ARZ3NDEKTSV4RRFFQ69G5FAV
Content-Type: application/json
```

**Request (Update metadata):**
```json
{
  "name": "Updated Service Name",
  "description": "Updated description",
  "can_write": true
}
```

**Request (Rotate key - regenerate API key):**
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
  "updated_at": "2026-02-01T17:00:00Z"
}
```

**Response (200 OK - key rotation):**
```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "name": "Production Service",
  "description": "Main API integration",
  "role": "user",
  "can_write": true,
  "key": "moon_live_xyz789uvw456rst123opq890nml567kji345hgf012edc678baz901yxw",
  "rotated_at": "2026-02-01T17:00:00Z",
  "warning": "Store this key securely. The old key is now invalid."
}
```

**Supported Operations:**
1. **Update metadata:**
   - Fields: `name`, `description`, `can_write`
   - `role` cannot be changed after creation (security design)
   - Name must be unique if changed

2. **Rotate key (action: rotate):**
   - Generates new cryptographically random key
   - Invalidates old key immediately
   - New key returned **only once** in response
   - Old key hash replaced with new key hash

**Business Rules:**
- Cannot change role after creation (create new key instead)
- Key rotation is instant (old key stops working immediately)
- Clients must update to new key before old key expires

**Error Responses:**
- `400 Bad Request`: Invalid action or missing fields
- `403 Forbidden`: Not an admin
- `404 Not Found`: API key does not exist
- `409 Conflict`: Name already in use

---

**FR-1.5: POST /apikeys:destroy**
```
POST /apikeys:destroy?id=01ARZ3NDEKTSV4RRFFQ69G5FAV
```

**Request:**
- Query param:
  - `id` (required): API key ULID to delete

**Response (200 OK):**
```json
{
  "message": "API key deleted successfully",
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

**Business Rules:**
- Deletion is immediate (key stops working instantly)
- No cascade effects (API keys are independent)
- Cannot be undone (create new key if needed)

**Error Responses:**
- `400 Bad Request`: Missing or invalid `id` parameter
- `403 Forbidden`: Not an admin
- `404 Not Found`: API key does not exist

---

### FR-2: API Key Repository Extensions

Extend the `APIKeyRepository` interface from PRD-044-A:

```go
type APIKeyRepository interface {
    // Existing methods from PRD-044-A
    Create(key *APIKey) error
    GetByID(ulid string) (*APIKey, error)
    GetByHash(hash string) (*APIKey, error)
    Update(key *APIKey) error
    Delete(ulid string) error
    UpdateLastUsed(ulid string) error
    
    // New methods for API key management
    List(limit int, cursor string) ([]*APIKey, string, error)
    ExistsByName(name string) (bool, error)
    UpdateMetadata(ulid string, name string, description string, canWrite bool) error
    RotateKey(ulid string, newKeyHash string) error
}
```

**Implementation Notes:**
- `List` returns keys sorted by `created_at DESC`
- Cursor pagination uses ULID (not internal ID)
- `ExistsByName` is case-insensitive
- `RotateKey` atomically updates key_hash and updated_at
- `UpdateLastUsed` called by auth middleware on each successful API key auth

---

### FR-3: API Key Generation

**FR-3.1: Key Format**
```
moon_live_{64_characters}
```

**FR-3.2: Character Set**
- Base62: `a-z`, `A-Z`, `0-9`, `-`, `_`
- Total possible keys: 62^64 (astronomically large space)
- Collision probability: negligible

**FR-3.3: Generation Function**
```go
func GenerateAPIKey() (string, error) {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
    const keyLength = 64
    
    // Use crypto/rand for cryptographic randomness
    bytes := make([]byte, keyLength)
    _, err := rand.Read(bytes)
    if err != nil {
        return "", err
    }
    
    // Map random bytes to charset
    for i := range bytes {
        bytes[i] = charset[int(bytes[i])%len(charset)]
    }
    
    return "moon_live_" + string(bytes), nil
}
```

**FR-3.4: Key Hashing**
```go
func HashAPIKey(key string) string {
    hash := sha256.Sum256([]byte(key))
    return hex.EncodeToString(hash[:])
}
```

**FR-3.5: Key Comparison**
```go
func CompareAPIKey(key string, hash string) bool {
    keyHash := HashAPIKey(key)
    // Use constant-time comparison to prevent timing attacks
    return subtle.ConstantTimeCompare([]byte(keyHash), []byte(hash)) == 1
}
```

---

### FR-4: API Key Validation

**FR-4.1: Format Validation**
```go
func ValidateAPIKeyFormat(key string) error {
    if !strings.HasPrefix(key, "moon_live_") {
        return errors.New("API key must start with 'moon_live_' prefix")
    }
    
    if len(key) != 74 { // prefix (10) + key (64)
        return errors.New("API key must be exactly 74 characters")
    }
    
    // Validate character set
    keyPart := key[10:] // Remove prefix
    validChars := regexp.MustCompile(`^[a-zA-Z0-9_-]{64}$`)
    if !validChars.MatchString(keyPart) {
        return errors.New("API key contains invalid characters")
    }
    
    return nil
}
```

**FR-4.2: Name Validation**
```go
func ValidateAPIKeyName(name string) error {
    if len(name) < 3 {
        return errors.New("name must be at least 3 characters")
    }
    if len(name) > 100 {
        return errors.New("name must not exceed 100 characters")
    }
    return nil
}
```

---

### FR-5: Audit Logging

Log all API key management operations:

**FR-5.1: Key Creation**
```
INFO: ADMIN_ACTION apikey_created by={admin_ulid} key_id={key_ulid} name={name} role={role}
```

**FR-5.2: Key Metadata Update**
```
INFO: ADMIN_ACTION apikey_updated by={admin_ulid} key_id={key_ulid} changes={field1,field2}
```

**FR-5.3: Key Rotation**
```
INFO: ADMIN_ACTION apikey_rotated by={admin_ulid} key_id={key_ulid} name={name}
```

**FR-5.4: Key Deletion**
```
INFO: ADMIN_ACTION apikey_deleted by={admin_ulid} key_id={key_ulid} name={name}
```

**FR-5.5: Key Usage**
```
DEBUG: APIKEY_AUTH key_id={key_ulid} name={name} endpoint={path}
```

**Logging Requirements:**
- Never log actual API key values
- Log key metadata (ULID, name) only
- Include admin ULID performing action
- Include timestamp (automatic)
- Log at INFO level for management operations
- Log at DEBUG level for key usage (authentication)

---

### FR-6: Error Handling

**FR-6.1: Consistent Error Format**
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message"
  }
}
```

**FR-6.2: Error Codes**
- `MISSING_REQUIRED_FIELD` (400): Required field missing
- `INVALID_FIELD_VALUE` (400): Field value invalid
- `INVALID_KEY_NAME` (400): Name validation failed
- `INVALID_ROLE` (400): Role not "admin" or "user"
- `INVALID_ACTION` (400): Action not recognized
- `ADMIN_REQUIRED` (403): Endpoint requires admin role
- `APIKEY_NOT_FOUND` (404): API key does not exist
- `APIKEY_NAME_EXISTS` (409): Name already in use

---

## Acceptance Criteria

### AC-1: List API Keys Endpoint

**Verification:**
- [ ] Returns paginated list of API keys
- [ ] Respects `limit` parameter (default 50, max 100)
- [ ] Cursor pagination works correctly
- [ ] Returns 403 for non-admin users
- [ ] Never returns actual key values (only metadata)
- [ ] Includes `last_used_at` timestamp

**Test:**
```bash
# Login as admin
ADMIN_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"AdminPass123"}' | jq -r '.access_token')

# List all API keys
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/apikeys:list"

# Expected: 200 with apikeys array

# List with pagination
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/apikeys:list?limit=10&after=01ARZ..."

# Try as non-admin
USER_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"user1","password":"UserPass123"}' | jq -r '.access_token')

curl -H "Authorization: Bearer $USER_TOKEN" \
  "http://localhost:6006/apikeys:list"

# Expected: 403 Forbidden

# Verify key value not in response
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/apikeys:list" | grep -q "moon_live_"

# Expected: No match (exit code 1)
```

### AC-2: Get API Key Endpoint

**Verification:**
- [ ] Returns single API key by ULID
- [ ] Returns 404 for non-existent key
- [ ] Returns 403 for non-admin users
- [ ] Never returns actual key value

**Test:**
```bash
# Get specific API key
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/apikeys:get?id=01ARZ3NDEKTSV4RRFFQ69G5FAV"

# Expected: 200 with key metadata

# Non-existent key
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/apikeys:get?id=01INVALID"

# Expected: 404 Not Found
```

### AC-3: Create API Key Endpoint

**Verification:**
- [ ] Creates API key with valid data
- [ ] Generates ULID for new key
- [ ] Generates cryptographically random key
- [ ] Returns actual key value only once
- [ ] Stores SHA-256 hash in database
- [ ] Returns 409 for duplicate name
- [ ] Returns 400 for invalid data
- [ ] Returns 403 for non-admin users

**Test:**
```bash
# Create new API key
RESPONSE=$(curl -s -X POST http://localhost:6006/apikeys:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Service",
    "description": "Testing API key creation",
    "role": "user",
    "can_write": false
  }')

echo $RESPONSE | jq .

# Expected: 201 Created with key field

# Verify key format
KEY=$(echo $RESPONSE | jq -r '.key')
echo $KEY | grep -q "^moon_live_[a-zA-Z0-9_-]\{64\}$"
# Expected: Match (exit code 0)

# Save key for later tests
API_KEY=$KEY
KEY_ID=$(echo $RESPONSE | jq -r '.id')

# Verify key works for authentication
curl -H "X-API-Key: $API_KEY" \
  "http://localhost:6006/collections:list"

# Expected: 200 OK

# Try to get key value again
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/apikeys:get?id=$KEY_ID" | grep -q "moon_live_"

# Expected: No match (key value not returned)

# Duplicate name
curl -X POST http://localhost:6006/apikeys:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Test Service",
    "description": "Different description",
    "role": "user"
  }'

# Expected: 409 Conflict
```

### AC-4: Update API Key Endpoint

**Verification:**
- [ ] Updates API key metadata
- [ ] Rotates API key (generates new key)
- [ ] Old key stops working after rotation
- [ ] New key returned only once during rotation
- [ ] Returns 404 for non-existent key
- [ ] Returns 409 for duplicate name

**Test:**
```bash
# Update metadata
curl -X POST "http://localhost:6006/apikeys:update?id=$KEY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Updated Service Name",
    "description": "Updated description",
    "can_write": true
  }'

# Expected: 200 OK

# Rotate key
OLD_KEY=$API_KEY
ROTATE_RESPONSE=$(curl -s -X POST "http://localhost:6006/apikeys:update?id=$KEY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "action": "rotate"
  }')

echo $ROTATE_RESPONSE | jq .

# Expected: 200 OK with new key field
NEW_KEY=$(echo $ROTATE_RESPONSE | jq -r '.key')

# Verify old key no longer works
curl -H "X-API-Key: $OLD_KEY" \
  "http://localhost:6006/collections:list"

# Expected: 401 Unauthorized

# Verify new key works
curl -H "X-API-Key: $NEW_KEY" \
  "http://localhost:6006/collections:list"

# Expected: 200 OK

# Update for next tests
API_KEY=$NEW_KEY
```

### AC-5: Delete API Key Endpoint

**Verification:**
- [ ] Deletes API key successfully
- [ ] Key stops working immediately
- [ ] Returns 404 for non-existent key

**Test:**
```bash
# Create test key to delete
DELETE_KEY_RESPONSE=$(curl -s -X POST http://localhost:6006/apikeys:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Delete Me",
    "description": "Key to be deleted",
    "role": "user"
  }')

DELETE_KEY=$(echo $DELETE_KEY_RESPONSE | jq -r '.key')
DELETE_KEY_ID=$(echo $DELETE_KEY_RESPONSE | jq -r '.id')

# Verify key works
curl -H "X-API-Key: $DELETE_KEY" \
  "http://localhost:6006/collections:list"

# Expected: 200 OK

# Delete key
curl -X POST "http://localhost:6006/apikeys:destroy?id=$DELETE_KEY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Expected: 200 OK

# Verify key no longer works
curl -H "X-API-Key: $DELETE_KEY" \
  "http://localhost:6006/collections:list"

# Expected: 401 Unauthorized

# Verify key not in list
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/apikeys:get?id=$DELETE_KEY_ID"

# Expected: 404 Not Found
```

### AC-6: API Key Format Validation

**Verification:**
- [ ] Key has correct prefix
- [ ] Key is correct length (74 chars total)
- [ ] Key uses valid character set (base62)
- [ ] Multiple keys are unique

**Test:**
```go
// Test API key generation
keys := make(map[string]bool)

for i := 0; i < 100; i++ {
    key, err := GenerateAPIKey()
    assert.NoError(t, err)
    
    // Verify format
    assert.True(t, strings.HasPrefix(key, "moon_live_"))
    assert.Equal(t, 74, len(key))
    
    // Verify character set
    validChars := regexp.MustCompile(`^moon_live_[a-zA-Z0-9_-]{64}$`)
    assert.True(t, validChars.MatchString(key))
    
    // Verify uniqueness
    assert.False(t, keys[key], "Duplicate key generated")
    keys[key] = true
}

// All 100 keys should be unique
assert.Equal(t, 100, len(keys))
```

### AC-7: API Key Hashing

**Verification:**
- [ ] Keys hashed with SHA-256
- [ ] Same key produces same hash
- [ ] Different keys produce different hashes
- [ ] Comparison uses constant time

**Test:**
```go
key := "moon_live_abc123def456ghi789jkl012mno345pqr678stu901vwx234yz567ABC890DEF"

// Hash key
hash1 := HashAPIKey(key)
assert.NotEmpty(t, hash1)
assert.NotEqual(t, key, hash1)

// Same key produces same hash
hash2 := HashAPIKey(key)
assert.Equal(t, hash1, hash2)

// Different key produces different hash
differentKey := "moon_live_xyz789uvw456rst123opq890nml567kji345hgf012edc678baz901yxw"
hash3 := HashAPIKey(differentKey)
assert.NotEqual(t, hash1, hash3)

// Verify comparison works
assert.True(t, CompareAPIKey(key, hash1))
assert.False(t, CompareAPIKey(differentKey, hash1))
```

### AC-8: Audit Logging

**Verification:**
- [ ] All API key management operations logged
- [ ] Logs include admin ULID and key ULID
- [ ] Actual key values never logged
- [ ] Logs at INFO level for management operations

**Test:**
```bash
# Perform various API key operations
# Check logs contain expected entries

grep "ADMIN_ACTION apikey_created" /var/log/moon/main.log
grep "ADMIN_ACTION apikey_updated" /var/log/moon/main.log
grep "ADMIN_ACTION apikey_rotated" /var/log/moon/main.log
grep "ADMIN_ACTION apikey_deleted" /var/log/moon/main.log

# Verify no actual key values in logs
! grep -E "moon_live_[a-zA-Z0-9_-]{64}" /var/log/moon/main.log
```

### AC-9: Error Handling

**Verification:**
- [ ] All error responses follow standard format
- [ ] Error codes match specification
- [ ] Error messages are clear

**Test:**
```bash
# Missing required field
curl -X POST http://localhost:6006/apikeys:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"description":"Missing name"}'

# Expected: 400 with MISSING_REQUIRED_FIELD

# Invalid role
curl -X POST http://localhost:6006/apikeys:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name":"test",
    "role":"superadmin"
  }'

# Expected: 400 with INVALID_ROLE
```

---

## Implementation Checklist

- [ ] Create `cmd/moon/internal/handlers/apikeys.go`
- [ ] Implement `APIKeysHandler` struct
- [ ] Implement `List` handler with pagination
- [ ] Implement `Get` handler
- [ ] Implement `Create` handler with key generation
- [ ] Implement `Update` handler with rotation support
- [ ] Implement `Destroy` handler
- [ ] Extend `APIKeyRepository` with new methods
- [ ] Implement API key generation function
- [ ] Implement API key hashing functions
- [ ] Implement API key validation functions
- [ ] Add API key management audit logging
- [ ] Add API key routes to server
- [ ] Write unit tests for key generation
- [ ] Write unit tests for key hashing
- [ ] Write unit tests for key validation
- [ ] Write unit tests for apikey handlers
- [ ] Write integration tests for all endpoints
- [ ] Test key rotation thoroughly
- [ ] Test error conditions
- [ ] Test authorization (admin-only access)

---

## Related PRDs

- [PRD-044: Authentication System](044-authentication-system.md) - Parent PRD
- [PRD-044-A: Core Authentication](044-A-core-authentication.md) - API keys table and key generation
- [PRD-044-B: User Management](044-B-user-management.md) - Similar admin-only management pattern

---

## References

- [SPEC_AUTH.md](../SPEC_AUTH.md) - API key management specification
- [SPEC.md](../SPEC.md) - API conventions
- [crypto/rand](https://pkg.go.dev/crypto/rand) - Cryptographic random generation
- [crypto/sha256](https://pkg.go.dev/crypto/sha256) - SHA-256 hashing
