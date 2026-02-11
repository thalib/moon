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
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDU4VjJQVjZaQU0zMTlGQlBUMUJGVjYiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDU4VjJQVjZaQU0zMTlGQlBUMUJGVjYiLCJleHAiOjE3NzA3ODA5MTgsIm5iZiI6MTc3MDc3NzI4OCwiaWF0IjoxNzcwNzc3MzE4fQ.92VaCZHog_u0GIae-5uJ7CUTcpozrwoz5DcvufH_WFo",
  "refresh_token": "4DxBuYxZxC79zZnbuF2yHY74RuFtnHaUW6v2ozSgwUs=",
  "expires_at": "2026-02-11T03:35:18.155242475Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
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
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
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
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
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
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
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
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDU4VjJQVjZaQU0zMTlGQlBUMUJGVjYiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDU4VjJQVjZaQU0zMTlGQlBUMUJGVjYiLCJleHAiOjE3NzA3ODA5MjEsIm5iZiI6MTc3MDc3NzI5MSwiaWF0IjoxNzcwNzc3MzIxfQ.BcrvgqQy7bCFjBbvV7t0F7DblQGYkq12xRP2fvpdis8",
  "refresh_token": "W8ktmOw6jFywWBh_rE1FlfONk-Xi0sOy7eVJwRQwy8s=",
  "expires_at": "2026-02-11T03:35:21.186642708Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH58V2PV6ZAM319FBPT1BFV6",
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
