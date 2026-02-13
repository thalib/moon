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
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSEE5NEpDSlBWSjczQ0cyRVhaMzE5WTEiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSEE5NEpDSlBWSjczQ0cyRVhaMzE5WTEiLCJleHAiOjE3NzA5NDkwMDAsIm5iZiI6MTc3MDk0NTM3MCwiaWF0IjoxNzcwOTQ1NDAwfQ.DLeAZsQBVn54HI4SNex6UWa25HnFYjVvbBuZgkudBg8",
  "refresh_token": "rwXkrIkSakZfDDWhtFvOFSP4qh2LoIGi-3Oq7v-08rA=",
  "expires_at": "2026-02-13T02:16:40.803122118Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
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
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
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
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
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
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
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
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSEE5NEpDSlBWSjczQ0cyRVhaMzE5WTEiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSEE5NEpDSlBWSjczQ0cyRVhaMzE5WTEiLCJleHAiOjE3NzA5NDkwMDMsIm5iZiI6MTc3MDk0NTM3MywiaWF0IjoxNzcwOTQ1NDAzfQ.4Do8c1d-XE6hb_vXvCRjK0v5El78J24VAJohSon5aGE",
  "refresh_token": "puoqS2Bs7SKBrxGPDmARO0AWLKMfyIYbJUgn-sW_ibY=",
  "expires_at": "2026-02-13T02:16:43.496549642Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KHA94JCJPVJ73CG2EXZ319Y1",
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
