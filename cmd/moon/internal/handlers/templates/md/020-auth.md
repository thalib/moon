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
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDJUUTU4SjZLNkFSQjFESDhQMVZBMjgiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDJUUTU4SjZLNkFSQjFESDhQMVZBMjgiLCJleHAiOjE3NzA2OTg5OTksIm5iZiI6MTc3MDY5NTM2OSwiaWF0IjoxNzcwNjk1Mzk5fQ.Nfr7OuNI4d4TW4inDPyOhs1fSRH1KqS0WxMrzdvVYdg",
  "refresh_token": "mex8-2is7JORAwcyCrANZ_CjjkKek3fD-A5JHTya1VY=",
  "expires_at": "2026-02-10T04:49:59.33911662Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH2TQ58J6K6ARB1DH8P1VA28",
    "username": "newuser",
    "email": "newuser@example.com",
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
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDJUUTU4SjZLNkFSQjFESDhQMVZBMjgiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDJUUTU4SjZLNkFSQjFESDhQMVZBMjgiLCJleHAiOjE3NzA2OTg5OTksIm5iZiI6MTc3MDY5NTM2OSwiaWF0IjoxNzcwNjk1Mzk5fQ.Nfr7OuNI4d4TW4inDPyOhs1fSRH1KqS0WxMrzdvVYdg",
  "refresh_token": "doS2OjomQEDQ0yC1BLUBVFwScL02RQhrcldoj3PBJFw=",
  "expires_at": "2026-02-10T04:49:59.761509426Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH2TQ58J6K6ARB1DH8P1VA28",
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
    "id": "01KGXRKR6J9NF6NAB1A1HY4PZQ",
    "username": "admin",
    "email": "newemail@example.com",
    "role": "admin",
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
    "id": "01KGXRKR6J9NF6NAB1A1HY4PZQ",
    "username": "admin",
    "email": "newemail@example.com",
    "role": "admin",
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

**Response (401 Unauthorized):**

```json
{
  "code": 401,
  "error": "invalid old password"
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
