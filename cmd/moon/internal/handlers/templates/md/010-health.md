### Check Health

```bash
curl -s -X GET "http://localhost:6006/health" | jq .
```

**Response (200 OK):**

```json
{
  "name": "moon",
  "status": "live",
  "version": "1.99"
}
```
