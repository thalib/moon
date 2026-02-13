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
    "id": "01KHA94RX000DPWXM9YENT6PH1",
    "username": "moonuser",
    "email": "moonuser@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-13T01:16:45Z",
    "updated_at": "2026-02-13T01:16:45Z"
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
      "id": "01KHA923Q12MQZTTZNWKNAPWYC",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin",
      "can_write": true,
      "created_at": "2026-02-13T01:15:18Z",
      "updated_at": "2026-02-13T01:16:44Z",
      "last_login_at": "2026-02-13T01:16:44Z"
    },
    {
      "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
      "username": "newuser",
      "email": "newemail@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-13T01:16:39Z",
      "updated_at": "2026-02-13T01:16:43Z",
      "last_login_at": "2026-02-13T01:16:43Z"
    },
    {
      "id": "01KHA94RX000DPWXM9YENT6PH1",
      "username": "moonuser",
      "email": "moonuser@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-13T01:16:45Z",
      "updated_at": "2026-02-13T01:16:45Z"
    }
  ],
  "next_cursor": null,
  "limit": 15
}
```

### Get Specific User by ID

```bash
curl -s -X GET "http://localhost:6006/users:get?id=01KHA94JCJPVJ73CG2EXZ319Y1" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "user": {
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
    "username": "newuser",
    "email": "newemail@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-13T01:16:39Z",
    "updated_at": "2026-02-13T01:16:43Z",
    "last_login_at": "2026-02-13T01:16:43Z"
  }
}
```

### Update User

```bash
curl -s -X POST "http://localhost:6006/users:update?id=01KHA94JCJPVJ73CG2EXZ319Y1" \
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
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-13T01:16:39Z",
    "updated_at": "2026-02-13T01:16:51Z",
    "last_login_at": "2026-02-13T01:16:43Z"
  }
}
```

### Reset User Password

```bash
curl -s -X POST "http://localhost:6006/users:update?id=01KHA94JCJPVJ73CG2EXZ319Y1" \
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
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-13T01:16:39Z",
    "updated_at": "2026-02-13T01:16:51Z",
    "last_login_at": "2026-02-13T01:16:43Z"
  }
}
```

### Revoke All User Sessions

```bash
curl -s -X POST "http://localhost:6006/users:update?id=01KHA94JCJPVJ73CG2EXZ319Y1" \
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
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-13T01:16:39Z",
    "updated_at": "2026-02-13T01:16:51Z",
    "last_login_at": "2026-02-13T01:16:43Z"
  }
}
```

### Delete User Account

```bash
curl -s -X POST "http://localhost:6006/users:destroy?id=01KHA94JCJPVJ73CG2EXZ319Y1" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "message": "user deleted successfully"
}
```
