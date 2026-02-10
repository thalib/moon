### Rate Limiting

Each response includes these headers to help you track your usage:

- `X-RateLimit-Limit`: Maximum requests allowed per time window
- `X-RateLimit-Remaining`: Requests left in the current window
- `X-RateLimit-Reset`: Unix timestamp when your quota resets

**Example Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1706875200
```

**If You Exceed the Limit (429 Too Many Requests):**

When you go over your limit (100/min/user for JWT, 1000/min/key for API Key), you’ll get:

```json
{
  "error": "Rate limit exceeded. Try again later.",
  "code": 429
}
```

**Response Headers:**
```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 60
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1706875260
```

Wait until the time in `X-RateLimit-Reset` or use the `Retry-After` value (in seconds) before making more requests.

---

### CORS Configuration

Moon supports Cross-Origin Resource Sharing (CORS) for browser clients with flexible, pattern-based configuration.

**Public Endpoints (No Auth)**

- `GET /health` – Health check
- `GET /doc` – API docs (HTML)
- `GET /doc/llms-full.txt` – API docs (Markdown)

**Default CORS Headers**

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 3600
```

- CORS can be set per endpoint or pattern in config.
- Admins can register CORS-enabled endpoints and restrict origins.
- Public endpoints can bypass authentication.
- See `SPEC.md` for full config details.

**OPTIONS Preflight Example:**

```bash
curl -X OPTIONS "{{$ApiURL}}/collections:list" \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET"
```

**Sample Response:**

```json
{
  "allowed_methods": ["GET", "POST", "OPTIONS"],
  "allowed_headers": ["Authorization", "Content-Type"],
  "max_age": 3600
}
```

**Sample Response Headers:**

```
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 3600
```

- For credentials, set a specific allowed origin (not `*`).
