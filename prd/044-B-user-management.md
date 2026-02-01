# PRD-044-B: User Management Endpoints

## Overview

Implement comprehensive user management endpoints for Moon's authentication system. These endpoints allow administrators to create, read, update, and delete user accounts, as well as perform administrative actions such as password resets and session revocation.

**Admin-Only Access:** All user management endpoints require admin role. Regular users cannot manage other users or modify their own role/permissions beyond what `/auth:me` allows.

### Problem Statement

Administrators need API endpoints to manage user accounts, assign roles, control permissions, reset passwords, and revoke sessions. This enables proper user lifecycle management and security administration.

### Dependencies

- **PRD-044-A:** Core authentication (users table, auth middleware, JWT)
- Existing: `database` package, `ulid` package, `validation` package

---

## Requirements

### FR-1: User Management Endpoints

All endpoints require admin authentication (`Authorization: Bearer <admin_token>`).

**FR-1.1: GET /users:list**
```
GET /users:list?limit=50&after=01ARZ...&role=user
```

**Request:**
- Query params:
  - `limit` (optional, default: 50, max: 100): Results per page
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
      "created_at": "2026-01-10T08:00:00Z",
      "last_login_at": "2026-02-01T09:00:00Z"
    },
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FBX",
      "username": "user1",
      "email": "user1@example.com",
      "role": "user",
      "can_write": false,
      "created_at": "2026-01-12T10:30:00Z",
      "last_login_at": "2026-01-15T14:20:00Z"
    }
  ],
  "next_cursor": "01ARZ3NDEKTSV4RRFFQ69G5FBX"
}
```

**Validation:**
- `limit` must be between 1 and 100
- `role` must be "admin" or "user" if provided
- `after` must be valid ULID if provided

**Authorization:**
- Requires admin role
- Returns 403 for non-admin users

---

**FR-1.2: GET /users:get**
```
GET /users:get?id=01ARZ3NDEKTSV4RRFFQ69G5FAV
```

**Request:**
- Query param:
  - `id` (required): User ULID

**Response (200 OK):**
```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "user1",
  "email": "user1@example.com",
  "role": "user",
  "can_write": false,
  "created_at": "2026-01-12T10:30:00Z",
  "updated_at": "2026-01-15T11:00:00Z",
  "last_login_at": "2026-01-15T14:20:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: Missing or invalid `id` parameter
- `403 Forbidden`: Not an admin
- `404 Not Found`: User does not exist

---

**FR-1.3: POST /users:create**
```
POST /users:create
Content-Type: application/json
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
  "created_at": "2026-02-01T15:30:00Z"
}
```

**Validation:**
- `username`: Required, 3-50 chars, alphanumeric + underscore + hyphen
- `email`: Required, valid email format
- `password`: Required, min 8 chars, includes uppercase, lowercase, number
- `role`: Required, must be "admin" or "user"
- `can_write`: Optional, boolean, default false

**Business Rules:**
- Username must be unique (case-insensitive)
- Email must be unique (case-insensitive)
- Password must meet security policy
- New user ULID auto-generated
- Password hashed with bcrypt (cost 12)
- `created_at` and `updated_at` set to current time

**Error Responses:**
- `400 Bad Request`: Validation errors
- `403 Forbidden`: Not an admin
- `409 Conflict`: Username or email already exists

---

**FR-1.4: POST /users:update**
```
POST /users:update?id=01ARZ3NDEKTSV4RRFFQ69G5FAV
Content-Type: application/json
```

**Request (Update role/permissions):**
```json
{
  "role": "user",
  "can_write": true,
  "email": "updated@example.com"
}
```

**Request (Reset password - admin action):**
```json
{
  "action": "reset_password",
  "new_password": "NewSecurePass456"
}
```

**Request (Revoke all sessions - admin action):**
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
  "email": "updated@example.com",
  "role": "user",
  "can_write": true,
  "updated_at": "2026-02-01T16:00:00Z"
}
```

**Supported Operations:**
1. **Update role and permissions:**
   - Fields: `role`, `can_write`, `email`
   - Email must be unique if changed
   - Updates `updated_at` timestamp

2. **Reset password (action: reset_password):**
   - Requires `new_password` field
   - Validates password against policy
   - Hashes and updates password
   - Invalidates all user's refresh tokens
   - User must re-login with new password

3. **Revoke sessions (action: revoke_sessions):**
   - Invalidates all user's refresh tokens
   - User must re-login on all devices
   - Does not change password

**Business Rules:**
- Cannot modify own role (admin cannot demote themselves)
- Cannot downgrade last admin to user role
- Password reset requires new_password field
- Email uniqueness checked if email updated
- `updated_at` timestamp always updated

**Error Responses:**
- `400 Bad Request`: Invalid action or missing fields
- `403 Forbidden`: Not an admin, or attempting to modify own role, or trying to delete last admin
- `404 Not Found`: User does not exist
- `409 Conflict`: Email already in use

---

**FR-1.5: POST /users:destroy**
```
POST /users:destroy?id=01ARZ3NDEKTSV4RRFFQ69G5FAV
```

**Request:**
- Query param:
  - `id` (required): User ULID to delete

**Response (200 OK):**
```json
{
  "message": "User deleted successfully",
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

**Business Rules:**
- Cannot delete self (admin cannot delete their own account)
- Cannot delete last admin user (must always have at least one admin)
- Deletion cascades to refresh_tokens table (foreign key constraint)
- User's active sessions immediately invalidated

**Error Responses:**
- `400 Bad Request`: Missing or invalid `id` parameter
- `403 Forbidden`: Not an admin, or attempting to delete self, or attempting to delete last admin
- `404 Not Found`: User does not exist

---

### FR-2: User Repository Extensions

Extend the `UserRepository` interface from PRD-044-A:

```go
type UserRepository interface {
    // Existing methods from PRD-044-A
    Create(user *User) error
    GetByID(ulid string) (*User, error)
    GetByUsername(username string) (*User, error)
    GetByEmail(email string) (*User, error)
    Update(user *User) error
    Delete(ulid string) error
    UpdateLastLogin(ulid string) error
    CountAdmins() (int, error)
    
    // New methods for user management
    List(limit int, cursor string, role string) ([]*User, string, error)
    UpdateRole(ulid string, role string, canWrite bool) error
    UpdateEmail(ulid string, email string) error
    UpdatePassword(ulid string, passwordHash string) error
    ExistsByUsername(username string) (bool, error)
    ExistsByEmail(email string) (bool, error)
    GetInternalIDByULID(ulid string) (int, error)
}
```

**Implementation Notes:**
- `List` returns users sorted by `created_at DESC`
- Cursor pagination uses ULID (not internal ID)
- `ExistsByUsername` and `ExistsByEmail` are case-insensitive
- `UpdatePassword` updates `password_hash` and `updated_at`
- `GetInternalIDByULID` needed for refresh token operations

---

### FR-3: Password Validation

**FR-3.1: Password Policy**
```go
type PasswordPolicy struct {
    MinLength        int
    RequireUppercase bool
    RequireLowercase bool
    RequireNumber    bool
    RequireSpecial   bool
}

var DefaultPasswordPolicy = PasswordPolicy{
    MinLength:        8,
    RequireUppercase: true,
    RequireLowercase: true,
    RequireNumber:    true,
    RequireSpecial:   false,
}
```

**FR-3.2: Validation Function**
```go
func ValidatePassword(password string, policy PasswordPolicy) error
```

**Validation Rules:**
- Check minimum length
- Check for uppercase letter (A-Z)
- Check for lowercase letter (a-z)
- Check for digit (0-9)
- Check for special character if required (!@#$%^&*()_+-=[]{}|;:,.<>?)

**Error Messages:**
- "Password must be at least 8 characters long"
- "Password must include an uppercase letter"
- "Password must include a lowercase letter"
- "Password must include a number"
- "Password must include a special character"

---

### FR-4: Admin Validation

**FR-4.1: Cannot Modify Self**
When admin performs `/users:update` or `/users:destroy`:
- Extract admin ULID from JWT token
- Compare with target user ULID
- If same, return 403: "Cannot modify own account via user management endpoints"

**FR-4.2: Cannot Delete Last Admin**
Before deleting admin user:
- Count total admins
- If count == 1 and target user is admin, return 403: "Cannot delete the last admin user"

Before downgrading admin to user:
- Count total admins
- If count == 1 and target user is admin, return 403: "Cannot downgrade the last admin user"

---

### FR-5: Audit Logging

Log all user management operations:

**FR-5.1: User Creation**
```
INFO: ADMIN_ACTION user_created by={admin_ulid} target={user_ulid} username={username} role={role}
```

**FR-5.2: User Update**
```
INFO: ADMIN_ACTION user_updated by={admin_ulid} target={user_ulid} changes={field1,field2}
```

**FR-5.3: Password Reset**
```
INFO: ADMIN_ACTION password_reset by={admin_ulid} target={user_ulid}
```

**FR-5.4: Session Revocation**
```
INFO: ADMIN_ACTION sessions_revoked by={admin_ulid} target={user_ulid}
```

**FR-5.5: User Deletion**
```
INFO: ADMIN_ACTION user_deleted by={admin_ulid} target={user_ulid} username={username}
```

**Logging Requirements:**
- Never log passwords or tokens
- Include admin user ULID performing action
- Include target user ULID
- Include timestamp (automatic)
- Include IP address if available
- Log at INFO level for success, WARN for authorization failures

---

### FR-6: Error Handling

**FR-6.1: Consistent Error Format**
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "details": {}
  }
}
```

**FR-6.2: Error Codes**
- `MISSING_REQUIRED_FIELD` (400): Required field missing
- `INVALID_FIELD_VALUE` (400): Field value invalid
- `WEAK_PASSWORD` (400): Password doesn't meet policy
- `INVALID_EMAIL_FORMAT` (400): Email format invalid
- `INVALID_ROLE` (400): Role not "admin" or "user"
- `INVALID_ACTION` (400): Action not recognized
- `ADMIN_REQUIRED` (403): Endpoint requires admin role
- `CANNOT_MODIFY_SELF` (403): Admin cannot modify own account
- `CANNOT_DELETE_LAST_ADMIN` (403): Must keep at least one admin
- `USER_NOT_FOUND` (404): User does not exist
- `USERNAME_EXISTS` (409): Username already taken
- `EMAIL_EXISTS` (409): Email already registered

---

## Acceptance Criteria

### AC-1: List Users Endpoint

**Verification:**
- [ ] Returns paginated list of users
- [ ] Respects `limit` parameter (default 50, max 100)
- [ ] Cursor pagination works correctly
- [ ] Role filter works (admin/user)
- [ ] Returns 403 for non-admin users
- [ ] Excludes `password_hash` from response

**Test:**
```bash
# Login as admin
ADMIN_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"admin","password":"AdminPass123"}' | jq -r '.access_token')

# List all users
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/users:list"

# Expected: 200 with user array

# List with pagination
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/users:list?limit=10&after=01ARZ..."

# Filter by role
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/users:list?role=user"

# Try as non-admin
USER_TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -d '{"username":"user1","password":"UserPass123"}' | jq -r '.access_token')

curl -H "Authorization: Bearer $USER_TOKEN" \
  "http://localhost:6006/users:list"

# Expected: 403 Forbidden
```

### AC-2: Get User Endpoint

**Verification:**
- [ ] Returns single user by ULID
- [ ] Returns 404 for non-existent user
- [ ] Returns 403 for non-admin users
- [ ] Excludes `password_hash` from response

**Test:**
```bash
# Get specific user
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/users:get?id=01ARZ3NDEKTSV4RRFFQ69G5FAV"

# Expected: 200 with user details

# Non-existent user
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/users:get?id=01INVALID"

# Expected: 404 Not Found
```

### AC-3: Create User Endpoint

**Verification:**
- [ ] Creates user with valid data
- [ ] Generates ULID for new user
- [ ] Hashes password with bcrypt
- [ ] Returns 409 for duplicate username/email
- [ ] Returns 400 for invalid data
- [ ] Returns 403 for non-admin users

**Test:**
```bash
# Create new user
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "email": "newuser@example.com",
    "password": "SecurePass123",
    "role": "user",
    "can_write": false
  }'

# Expected: 201 Created

# Duplicate username
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "username": "newuser",
    "email": "different@example.com",
    "password": "SecurePass123",
    "role": "user"
  }'

# Expected: 409 Conflict

# Weak password
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "weak",
    "role": "user"
  }'

# Expected: 400 Bad Request with password policy error
```

### AC-4: Update User Endpoint

**Verification:**
- [ ] Updates user role and permissions
- [ ] Resets password and invalidates sessions
- [ ] Revokes all sessions
- [ ] Returns 403 when admin tries to modify self
- [ ] Returns 403 when trying to downgrade last admin
- [ ] Updates `updated_at` timestamp

**Test:**
```bash
# Update user permissions
curl -X POST "http://localhost:6006/users:update?id=01ARZ..." \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "role": "user",
    "can_write": true
  }'

# Expected: 200 OK

# Reset password
curl -X POST "http://localhost:6006/users:update?id=01ARZ..." \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "action": "reset_password",
    "new_password": "NewSecure456"
  }'

# Expected: 200 OK
# Verify: User must re-login

# Revoke sessions
curl -X POST "http://localhost:6006/users:update?id=01ARZ..." \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "action": "revoke_sessions"
  }'

# Expected: 200 OK

# Try to modify self
ADMIN_ID=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:6006/auth:me | jq -r '.id')

curl -X POST "http://localhost:6006/users:update?id=$ADMIN_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "role": "user"
  }'

# Expected: 403 Forbidden
```

### AC-5: Delete User Endpoint

**Verification:**
- [ ] Deletes user successfully
- [ ] Cascades to refresh_tokens
- [ ] Returns 403 when trying to delete self
- [ ] Returns 403 when trying to delete last admin
- [ ] Returns 404 for non-existent user

**Test:**
```bash
# Create test user to delete
TEST_USER_ID=$(curl -s -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "username": "deleteme",
    "email": "delete@example.com",
    "password": "Delete123",
    "role": "user"
  }' | jq -r '.id')

# Delete user
curl -X POST "http://localhost:6006/users:destroy?id=$TEST_USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Expected: 200 OK

# Verify user deleted
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:6006/users:get?id=$TEST_USER_ID"

# Expected: 404 Not Found

# Try to delete self
curl -X POST "http://localhost:6006/users:destroy?id=$ADMIN_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Expected: 403 Forbidden
```

### AC-6: Password Validation

**Verification:**
- [ ] Enforces minimum length
- [ ] Requires uppercase letter
- [ ] Requires lowercase letter
- [ ] Requires number
- [ ] Returns clear error messages

**Test:**
```go
// Test password validation
tests := []struct {
    password string
    valid    bool
    error    string
}{
    {"Short1", false, "must be at least 8 characters"},
    {"nouppercase123", false, "must include an uppercase letter"},
    {"NOLOWERCASE123", false, "must include a lowercase letter"},
    {"NoNumbers", false, "must include a number"},
    {"ValidPass123", true, ""},
}

for _, tt := range tests {
    err := ValidatePassword(tt.password, DefaultPasswordPolicy)
    if tt.valid {
        assert.NoError(t, err)
    } else {
        assert.Error(t, err)
        assert.Contains(t, err.Error(), tt.error)
    }
}
```

### AC-7: Audit Logging

**Verification:**
- [ ] All user management operations logged
- [ ] Logs include admin ULID and target user ULID
- [ ] Passwords never logged
- [ ] Logs at INFO level for success

**Test:**
```bash
# Perform various user management operations
# Check logs contain expected entries

grep "ADMIN_ACTION user_created" /var/log/moon/main.log
grep "ADMIN_ACTION user_updated" /var/log/moon/main.log
grep "ADMIN_ACTION password_reset" /var/log/moon/main.log
grep "ADMIN_ACTION sessions_revoked" /var/log/moon/main.log
grep "ADMIN_ACTION user_deleted" /var/log/moon/main.log

# Verify no passwords in logs
! grep -i "password.*:" /var/log/moon/main.log | grep -v "password_hash"
```

### AC-8: Error Handling

**Verification:**
- [ ] All error responses follow standard format
- [ ] Error codes match specification
- [ ] Error messages are clear
- [ ] No internal details exposed

**Test:**
```bash
# Test various error conditions
# Missing required field
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"username":"test"}'

# Expected: 400 with MISSING_REQUIRED_FIELD

# Invalid role
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "username":"test",
    "email":"test@example.com",
    "password":"Test1234",
    "role":"superadmin"
  }'

# Expected: 400 with INVALID_ROLE
```

---

## Implementation Checklist

- [ ] Create `cmd/moon/internal/handlers/users.go`
- [ ] Implement `UsersHandler` struct
- [ ] Implement `List` handler with pagination
- [ ] Implement `Get` handler
- [ ] Implement `Create` handler with validation
- [ ] Implement `Update` handler with actions support
- [ ] Implement `Destroy` handler
- [ ] Extend `UserRepository` with new methods
- [ ] Implement password validation function
- [ ] Implement admin validation helpers
- [ ] Add admin action audit logging
- [ ] Add user management routes to server
- [ ] Write unit tests for password validation
- [ ] Write unit tests for admin validation
- [ ] Write unit tests for user handlers
- [ ] Write integration tests for all endpoints
- [ ] Test error conditions thoroughly
- [ ] Test authorization (admin-only access)
- [ ] Test last admin protection
- [ ] Test cannot modify self protection

---

## Related PRDs

- [PRD-044: Authentication System](044-authentication-system.md) - Parent PRD
- [PRD-044-A: Core Authentication](044-A-core-authentication.md) - Users table and auth middleware
- [PRD-044-C: API Key Management](044-C-apikey-management.md) - Similar admin-only management

---

## References

- [SPEC_AUTH.md](../SPEC_AUTH.md) - User management specification
- [SPEC.md](../SPEC.md) - API conventions and error handling
