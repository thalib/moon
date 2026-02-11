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
    "id": "01KH58YJCJSD8NFKCTP2D4VM9C",
    "username": "moonuser",
    "email": "moonuser@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-11T02:37:10Z",
    "updated_at": "2026-02-11T02:37:10Z"
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
      "updated_at": "2026-02-11T02:37:09Z",
      "last_login_at": "2026-02-11T02:37:09Z"
    },
    {
      "id": "01KH58V2PV6ZAM319FBPT1BFV6",
      "username": "newuser",
      "email": "newemail@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-11T02:35:16Z",
      "updated_at": "2026-02-11T02:35:20Z",
      "last_login_at": "2026-02-11T02:35:20Z"
    },
    {
      "id": "01KH58YJCJSD8NFKCTP2D4VM9C",
      "username": "moonuser",
      "email": "moonuser@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-11T02:37:10Z",
      "updated_at": "2026-02-11T02:37:10Z"
    }
  ],
  "next_cursor": null,
  "limit": 15
}
```

### Get Specific User by ID

```bash
curl -s -X GET "http://localhost:6006/users:get?id=01KH58V2PV6ZAM319FBPT1BFV6" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "user": {
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
    "username": "newuser",
    "email": "newemail@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-11T02:35:16Z",
    "updated_at": "2026-02-11T02:35:20Z",
    "last_login_at": "2026-02-11T02:35:20Z"
  }
}
```

### Update User

```bash
curl -s -X POST "http://localhost:6006/users:update?id=01KH58V2PV6ZAM319FBPT1BFV6" \
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
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-11T02:35:16Z",
    "updated_at": "2026-02-11T02:37:11Z",
    "last_login_at": "2026-02-11T02:35:20Z"
  }
}
```

### Reset User Password

```bash
curl -s -X POST "http://localhost:6006/users:update?id=01KH58V2PV6ZAM319FBPT1BFV6" \
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
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-11T02:35:16Z",
    "updated_at": "2026-02-11T02:37:12Z",
    "last_login_at": "2026-02-11T02:35:20Z"
  }
}
```

### Revoke All User Sessions

```bash
curl -s -X POST "http://localhost:6006/users:update?id=01KH58V2PV6ZAM319FBPT1BFV6" \
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
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-11T02:35:16Z",
    "updated_at": "2026-02-11T02:37:12Z",
    "last_login_at": "2026-02-11T02:35:20Z"
  }
}
```

### Delete User Account

```bash
curl -s -X POST "http://localhost:6006/users:destroy?id=01KH58V2PV6ZAM319FBPT1BFV6" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "message": "user deleted successfully"
}
```
