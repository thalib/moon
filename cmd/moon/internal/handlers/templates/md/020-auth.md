### Login

```bash
curl -s -X POST "http://localhost:6006/auth:login" \
    -H "Content-Type: application/json" \
    -d '
      {
        "username": "newuser",
        "password": "UserPass123#"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDM4RTRYNTlTMTM5OUMwQUtaMjdROUsiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDM4RTRYNTlTMTM5OUMwQUtaMjdROUsiLCJleHAiOjE3NzA3MTMzODQsIm5iZiI6MTc3MDcwOTc1NCwiaWF0IjoxNzcwNzA5Nzg0fQ.8eXAnaZg-TSXtloL4u8rRi_zrjRb26DDAetPkuZnZ58",
  "refresh_token": "9y-gvagGTyf0DmBQSTgigor8_oTp_F8shtpv7A48gWg=",
  "expires_at": "2026-02-10T08:49:44.915026874Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "can_write": true
  }
}
```

### Get Current User

```bash
curl -s -X GET "http://localhost:6006/auth:me" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "user": {
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "can_write": true
  }
}
```

### Update Current User (Change email)

```bash
curl -s -X POST "http://localhost:6006/auth:me" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "email": "newemail@example.com"
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
    "email": "newemail@example.com",
    "role": "user",
    "can_write": true
  }
}
```

### Update Current User (Change Password)

```bash
curl -s -X POST "http://localhost:6006/auth:me" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "old_password": "UserPass123#",
        "password": "NewSecurePass456"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "message": "password updated successfully, please login again",
  "user": {
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "newemail@example.com",
    "role": "user",
    "can_write": true
  }
}
```

### Refresh Token

```bash
curl -s -X POST "http://localhost:6006/auth:refresh" \
    -H "Content-Type: application/json" \
    -d '
      {
        "refresh_token": "$REFRESH_TOKEN"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDM4RTRYNTlTMTM5OUMwQUtaMjdROUsiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDM4RTRYNTlTMTM5OUMwQUtaMjdROUsiLCJleHAiOjE3NzA3MTMzODcsIm5iZiI6MTc3MDcwOTc1NywiaWF0IjoxNzcwNzA5Nzg3fQ.1Kseh4GbdWMXdMo2C9AkwgqVCWHmA59z_C0vhouRYg8",
  "refresh_token": "5468_9rObtkXOzHKFH4UqdvGaAKychif_YzZbc0ZQMc=",
  "expires_at": "2026-02-10T08:49:47.563571789Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH38E4X59S1399C0AKZ27Q9K",
    "username": "newuser",
    "email": "newemail@example.com",
    "role": "user",
    "can_write": true
  }
}
```

### Logout

```bash
curl -s -X POST "http://localhost:6006/auth:logout" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "refresh_token": "$REFRESH_TOKEN"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "message": "logged out successfully"
}
```
