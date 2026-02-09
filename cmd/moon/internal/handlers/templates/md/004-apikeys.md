**Create API Key**

```bash
curl -s -X POST "http://localhost:6006/apikeys:create" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "Integration Service",
        "description": "Key for integration",
        "role": "user",
        "can_write": false
      }
    ' | jq .
```

***Response (409 Conflict):***

```json
{
  "code": 409,
  "error": "API key name already exists",
  "error_code": "APIKEY_NAME_EXISTS"
}
```

**List API Keys**

```bash
curl -s -X GET "http://localhost:6006/apikeys:list" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "apikeys": [
    {
      "id": "01KH1BC54RKQV9SJHN7RMC5ATT",
      "name": "Integration Service",
      "description": "Key for integration",
      "role": "user",
      "can_write": false,
      "created_at": "2026-02-09T14:02:35Z"
    },
    {
      "id": "01KH1KAH6N69SF8ADP1WAHM7AZ",
      "name": "Another Service",
      "description": "Another key for testing",
      "role": "user",
      "can_write": false,
      "created_at": "2026-02-09T16:21:30Z"
    }
  ],
  "next_cursor": null,
  "limit": 15
}
```

**Get API Key**

```bash
curl -s -X GET "http://localhost:6006/apikeys:get?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "apikey": {
    "id": "01KH1BC54RKQV9SJHN7RMC5ATT",
    "name": "Integration Service",
    "description": "Key for integration",
    "role": "user",
    "can_write": false,
    "created_at": "2026-02-09T14:02:35Z"
  }
}
```

**Update API Key Metadata**

```bash
curl -s -X POST "http://localhost:6006/apikeys:update?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "Updated Service Name",
        "description": "Updated description",
        "can_write": true
      }
    ' | jq .
```

***Response (200 OK):***

```json
{
  "message": "API key updated successfully",
  "apikey": {
    "id": "01KH1BC54RKQV9SJHN7RMC5ATT",
    "name": "Updated Service Name",
    "description": "Updated description",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-09T14:02:35Z"
  }
}
```

**Rotate API Key**

```bash
curl -s -X POST "http://localhost:6006/apikeys:update?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "action": "rotate"
      }
    ' | jq .
```

***Response (200 OK):***

```json
{
  "message": "API key rotated successfully",
  "warning": "Store this key securely. It will not be shown again.",
  "apikey": {
    "id": "01KH1BC54RKQV9SJHN7RMC5ATT",
    "name": "Updated Service Name",
    "description": "Updated description",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-09T14:02:35Z"
  },
  "key": "moon_live_MLVFC8yEYFlvCtv5yXkXuOVk0gnpWIzwoudMXjLYuk7InNjnoxDseehhyYE3jROY"
}
```

**Delete API Key**

```bash
curl -s -X POST "http://localhost:6006/apikeys:destroy?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "message": "API key deleted successfully"
}
```
