**Login**

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

***Response (200 OK):***

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDFLOUc0S1BHMTBCRVYwUE5UTVlOV0oiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDFLOUc0S1BHMTBCRVYwUE5UTVlOV0oiLCJleHAiOjE3NzA2NTc2NTcsIm5iZiI6MTc3MDY1NDAyNywiaWF0IjoxNzcwNjU0MDU3fQ.tnjRv9SunQgUmv1naz2Bon_Ic1WZCVKcPemVN4vqVRY",
  "refresh_token": "ii2Sr4zZP6yEZmNSQ3ZUTNkHx--Y5ZDQ5ht49Hfpk0A=",
  "expires_at": "2026-02-09T17:20:57.698028118Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH1K9G4KPG10BEV0PNTMYNWJ",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "can_write": true
  }
}
```

**Refresh Token**

```bash
curl -s -X POST "http://localhost:6006/auth:refresh" \
    -H "Content-Type: application/json" \
    -d '
      {
        "refresh_token": "$REFRESH_TOKEN"
      }
    ' | jq .
```

***Response (200 OK):***

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFLSDFLOUc0S1BHMTBCRVYwUE5UTVlOV0oiLCJ1c2VybmFtZSI6Im5ld3VzZXIiLCJyb2xlIjoidXNlciIsImNhbl93cml0ZSI6dHJ1ZSwic3ViIjoiMDFLSDFLOUc0S1BHMTBCRVYwUE5UTVlOV0oiLCJleHAiOjE3NzA2NTc2NTgsIm5iZiI6MTc3MDY1NDAyOCwiaWF0IjoxNzcwNjU0MDU4fQ.TVQz0mJgDQl1W5DB8Rir75caBj1PLn54_kEi_aLtduI",
  "refresh_token": "1gXYUcT5FH3uzVpLmyGHAlOBtM_ROW33N3CvQPATgU8=",
  "expires_at": "2026-02-09T17:20:58.33201166Z",
  "token_type": "Bearer",
  "user": {
    "id": "01KH1K9G4KPG10BEV0PNTMYNWJ",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "can_write": true
  }
}
```

**Get Current User**

```bash
curl -s -X GET "http://localhost:6006/auth:me" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

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

**Update Current User (Change email)**

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

***Response (200 OK):***

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

**Update Current User (Change Password)**

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

***Response (401 Unauthorized):***

```json
{
  "code": 401,
  "error": "invalid old password"
}
```

**Logout**

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

***Response (200 OK):***

```json
{
  "message": "logged out successfully"
}
```
