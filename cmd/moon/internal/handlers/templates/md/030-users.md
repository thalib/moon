### Create New User

```bash
curl -s -X POST "http://localhost:6006/users:create" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "username": "newuser",
        "email": "newuser@example.com",
        "password": "UserPass123#",
        "role": "user"
      }
    ' | jq .
```

**Response (409 Conflict):**

```json
{
  "code": 409,
  "error": "username already exists",
  "error_code": "USERNAME_EXISTS"
}
```

### List All Users

```bash
curl -s -X GET "http://localhost:6006/users:list" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "users": [
    {
      "id": "01KGXRKR6J9NF6NAB1A1HY4PZQ",
      "username": "admin",
      "email": "newemail@example.com",
      "role": "admin",
      "can_write": true,
      "created_at": "2026-02-08T04:36:57Z",
      "updated_at": "2026-02-10T03:50:02Z",
      "last_login_at": "2026-02-10T03:50:02Z"
    },
    {
      "id": "01KH2TQ58J6K6ARB1DH8P1VA28",
      "username": "newuser",
      "email": "newuser@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-10T03:49:58Z",
      "updated_at": "2026-02-10T03:49:59Z",
      "last_login_at": "2026-02-10T03:49:59Z"
    }
  ],
  "next_cursor": null,
  "limit": 15
}
```

### Get Specific User by ID

```bash
curl -s -X GET "http://localhost:6006/users:get?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "user": {
    "id": "01KH2TQ58J6K6ARB1DH8P1VA28",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-10T03:49:58Z",
    "updated_at": "2026-02-10T03:49:59Z",
    "last_login_at": "2026-02-10T03:49:59Z"
  }
}
```

### Update User

```bash
curl -s -X POST "http://localhost:6006/users:update?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "email": "updateduser@example.com",
        "role": "admin"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "message": "user updated successfully",
  "user": {
    "id": "01KH2TQ58J6K6ARB1DH8P1VA28",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-10T03:49:58Z",
    "updated_at": "2026-02-10T03:50:04Z",
    "last_login_at": "2026-02-10T03:49:59Z"
  }
}
```

### Reset User Password

```bash
curl -s -X POST "http://localhost:6006/users:update?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "action": "reset_password",
        "new_password": "NewSecurePassword123#"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "message": "password reset successfully",
  "user": {
    "id": "01KH2TQ58J6K6ARB1DH8P1VA28",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-10T03:49:58Z",
    "updated_at": "2026-02-10T03:50:04Z",
    "last_login_at": "2026-02-10T03:49:59Z"
  }
}
```

### Revoke All User Sessions

```bash
curl -s -X POST "http://localhost:6006/users:update?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "action": "revoke_sessions"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "message": "all sessions revoked successfully",
  "user": {
    "id": "01KH2TQ58J6K6ARB1DH8P1VA28",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-10T03:49:58Z",
    "updated_at": "2026-02-10T03:50:04Z",
    "last_login_at": "2026-02-10T03:49:59Z"
  }
}
```

### Delete User Account

```bash
curl -s -X POST "http://localhost:6006/users:destroy?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "message": "user deleted successfully"
}
```
