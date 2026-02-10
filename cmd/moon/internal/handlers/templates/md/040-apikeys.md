### Create API Key

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

**Response (409 Conflict):**

```json
{
  "code": 409,
  "error": "API key name already exists",
  "error_code": "APIKEY_NAME_EXISTS"
}
```

### List API Keys

```bash
curl -s -X GET "http://localhost:6006/apikeys:list" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "apikeys": [
    {
      "id": "01KH2TF1H135KPW0E4K0GK74PJ",
      "name": "Integration Service",
      "description": "Key for integration",
      "role": "user",
      "can_write": false,
      "created_at": "2026-02-10T03:45:32Z"
    },
    {
      "id": "01KH2TQDK3FETD959WMH4BVT8A",
      "name": "Another Service",
      "description": "Another key for testing",
      "role": "user",
      "can_write": false,
      "created_at": "2026-02-10T03:50:07Z"
    }
  ],
  "next_cursor": null,
  "limit": 15
}
```

### Get API Key

```bash
curl -s -X GET "http://localhost:6006/apikeys:get?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "apikey": {
    "id": "01KH2TF1H135KPW0E4K0GK74PJ",
    "name": "Integration Service",
    "description": "Key for integration",
    "role": "user",
    "can_write": false,
    "created_at": "2026-02-10T03:45:32Z"
  }
}
```

### Update API Key Metadata

***Note:*** Update metadata fields (name, description, can_write) without changing the API key itself. The key remains valid. To generate a new key, use the rotation action.

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

**Response (200 OK):**

```json
{
  "message": "API key updated successfully",
  "apikey": {
    "id": "01KH2TF1H135KPW0E4K0GK74PJ",
    "name": "Updated Service Name",
    "description": "Updated description",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-10T03:45:32Z"
  }
}
```

### Rotate API Key

Use `rotate` to securely generate a new API key and invalidate the old one in a single step, minimizing overlap between valid keys.

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

**Response (200 OK):**

```json
{
  "message": "API key rotated successfully",
  "warning": "Store this key securely. It will not be shown again.",
  "apikey": {
    "id": "01KH2TF1H135KPW0E4K0GK74PJ",
    "name": "Updated Service Name",
    "description": "Updated description",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-10T03:45:32Z"
  },
  "key": "moon_live_vyupo3NArXJSsPm42E1gWcpW6Ncqxbkhdio6v47uVIjnrkdk5de3Qqaxr0c4mQKn"
}
```

### Delete API Key

```bash
curl -s -X POST "http://localhost:6006/apikeys:destroy?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "message": "API key deleted successfully"
}
```
