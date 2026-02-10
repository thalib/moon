### Create New User

```bash
curl -s -X POST "http://localhost:6006/users:create" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "username": "moonuser",
        "email": "moonuser@example.com",
        "password": "UserPass123#",
        "role": "user"
      }
    ' | jq .
```

**Response (201 Created):**

```json
{
  "message": "user created successfully",
  "user": {
    "id": "01KH38EAYK418S90EAY4KWHDE4",
    "username": "moonuser",
    "email": "moonuser@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-10T07:49:49Z",
    "updated_at": "2026-02-10T07:49:49Z"
  }
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
      "id": "01KH37V1PTKQ725F6QFG2BDHC6",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin",
      "can_write": true,
      "created_at": "2026-02-10T07:39:17Z",
      "updated_at": "2026-02-10T07:49:48Z",
      "last_login_at": "2026-02-10T07:49:48Z"
    },
    {
      "id": "01KH38E4X59S1399C0AKZ27Q9K",
      "username": "newuser",
      "email": "newemail@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-10T07:49:43Z",
      "updated_at": "2026-02-10T07:49:47Z",
      "last_login_at": "2026-02-10T07:49:47Z"
    },
    {
      "id": "01KH38EAYK418S90EAY4KWHDE4",
      "username": "moonuser",
      "email": "moonuser@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-10T07:49:49Z",
      "updated_at": "2026-02-10T07:49:49Z"
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
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "newemail@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-10T07:49:43Z",
    "updated_at": "2026-02-10T07:49:47Z",
    "last_login_at": "2026-02-10T07:49:47Z"
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
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-10T07:49:43Z",
    "updated_at": "2026-02-10T07:49:50Z",
    "last_login_at": "2026-02-10T07:49:47Z"
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
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-10T07:49:43Z",
    "updated_at": "2026-02-10T07:49:51Z",
    "last_login_at": "2026-02-10T07:49:47Z"
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
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-10T07:49:43Z",
    "updated_at": "2026-02-10T07:49:51Z",
    "last_login_at": "2026-02-10T07:49:47Z"
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
