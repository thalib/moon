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

**Response (201 Created):**

```json
{
  "message": "API key created successfully",
  "warning": "Store this key securely. It will not be shown again.",
  "apikey": {
    "id": "01KHA950QHFCVB75J1PSM766T4",
    "name": "Integration Service",
    "description": "Key for integration",
    "role": "user",
    "can_write": false,
    "created_at": "2026-02-13T01:16:53Z"
  },
  "key": "moon_live_BsNlkbYXxLLToULo9uJfTk3YVQWF3uF1B1s9W6d3iEIpnnAJGcQD7DLj6IRI3kSg"
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
      "id": "01KHA950QHFCVB75J1PSM766T4",
      "name": "Integration Service",
      "description": "Key for integration",
      "role": "user",
      "can_write": false,
      "created_at": "2026-02-13T01:16:53Z"
    },
    {
      "id": "01KHA9512J5RQ6TN82J2BX6SDY",
      "name": "Another Service",
      "description": "Another key for testing",
      "role": "user",
      "can_write": false,
      "created_at": "2026-02-13T01:16:54Z"
    }
  ],
  "next_cursor": null,
  "limit": 15
}
```

### Get API Key

```bash
curl -s -X GET "http://localhost:6006/apikeys:get?id=01KHA950QHFCVB75J1PSM766T4" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "apikey": {
    "id": "01KHA950QHFCVB75J1PSM766T4",
    "name": "Integration Service",
    "description": "Key for integration",
    "role": "user",
    "can_write": false,
    "created_at": "2026-02-13T01:16:53Z"
  }
}
```

### Update API Key Metadata

***Note:*** Update metadata fields (name, description, can_write) without changing the API key itself. The key remains valid. To generate a new key, use the rotation action.

```bash
curl -s -X POST "http://localhost:6006/apikeys:update?id=01KHA950QHFCVB75J1PSM766T4" \
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
    "id": "01KHA950QHFCVB75J1PSM766T4",
    "name": "Updated Service Name",
    "description": "Updated description",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-13T01:16:53Z"
  }
}
```

### Rotate API Key

Use `rotate` to securely generate a new API key and invalidate the old one in a single step, minimizing overlap between valid keys.

```bash
curl -s -X POST "http://localhost:6006/apikeys:update?id=01KHA950QHFCVB75J1PSM766T4" \
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
    "id": "01KHA950QHFCVB75J1PSM766T4",
    "name": "Updated Service Name",
    "description": "Updated description",
    "role": "user",
    "can_write": true,
    "created_at": "2026-02-13T01:16:53Z"
  },
  "key": "moon_live_ohx73hfTCiVK14GjneUD4orvleo0N4E0IcY9z6c7Ya5VZwdimKrmykBEJm3s74YY"
}
```

### Delete API Key

```bash
curl -s -X POST "http://localhost:6006/apikeys:destroy?id=01KHA950QHFCVB75J1PSM766T4" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "message": "API key deleted successfully"
}
```
