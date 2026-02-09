**Create New User**

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

***Response (409 Conflict):***

```json
{
  "code": 409,
  "error": "username already exists",
  "error_code": "USERNAME_EXISTS"
}
```

**List All Users**

```bash
curl -s -X GET "http://localhost:6006/users:list" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

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
      "updated_at": "2026-02-09T16:21:02Z",
      "last_login_at": "2026-02-09T16:21:02Z"
    },
    {
      "id": "01KH1K9G4KPG10BEV0PNTMYNWJ",
      "username": "newuser",
      "email": "newuser@example.com",
      "role": "user",
      "can_write": true,
      "created_at": "2026-02-09T16:20:56Z",
      "updated_at": "2026-02-09T16:20:57Z",
      "last_login_at": "2026-02-09T16:20:57Z"
    }
  ],
  "next_cursor": null,
  "limit": 15
}
```

**Get Specific User by ID**

```bash
curl -s -X GET "http://localhost:6006/users:get?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "user": {
    "id": "01KH1K9G4KPG10BEV0PNTMYNWJ",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-09T16:20:56Z",
    "updated_at": "2026-02-09T16:20:57Z",
    "last_login_at": "2026-02-09T16:20:57Z"
  }
}
```

**Update User**

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

***Response (200 OK):***

```json
{
  "message": "user updated successfully",
  "user": {
    "id": "01KH1K9G4KPG10BEV0PNTMYNWJ",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-09T16:20:56Z",
    "updated_at": "2026-02-09T16:21:04Z",
    "last_login_at": "2026-02-09T16:20:57Z"
  }
}
```

**Reset User Password**

```bash
curl -s -X POST "http://localhost:6006/users:update?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "action": "reset_password",
        "password": "NewSecurePassword123#"
      }
    ' | jq .
```

***Response (400 Bad Request):***

```json
{
  "code": 400,
  "error": "new_password is required for password reset",
  "error_code": "MISSING_REQUIRED_FIELD"
}
```

**Revoke All User Sessions**

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

***Response (200 OK):***

```json
{
  "message": "all sessions revoked successfully",
  "user": {
    "id": "01KH1K9G4KPG10BEV0PNTMYNWJ",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "can_write": true,
    "created_at": "2026-02-09T16:20:56Z",
    "updated_at": "2026-02-09T16:21:04Z",
    "last_login_at": "2026-02-09T16:20:57Z"
  }
}
```

**Delete User Account**

```bash
curl -s -X POST "http://localhost:6006/users:destroy?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "message": "user deleted successfully"
}
```
