Moon enables rapid, migrationless management of database tables and records through a dynamic RESTful API.

### Moon Terminology

Moon stores data in the format: Collection, Field, and Record.

In database terminology, these are Tables, Columns, and Rows, respectively.

For example, consider the table `products` below.

```md
| id        | name           | price   | in_stock |
|-----------|----------------|---------|----------|
| 01H1...   | Wireless Mouse | 29.99   | true     |
| 01H2...   | USB Keyboard   | 19.99   | false    |
```

- **Collection:** `products` (the table)
- **Field:** `id`, `name`, `price`, `in_stock` (the columns)
- **Record:** Each row, e.g., `{ "id": "01H1...", "name": "Wireless Mouse", "price": "29.99", "in_stock": true }`

### What Moon Does NOT Do

Moon is intentionally minimal. It does **not** support:

- **Transactions:** No multi-statement atomicity; each request is independent.
- **Joins:** No SQL joins; model relationships in your application.
- **Triggers/Hooks:** No database-level triggers or event hooks.
- **Background Jobs:** No built-in job queue or async processing.
- **Foreign Keys:** Relations must be managed manually.
- **Migrations/Schema Versioning:** No built-in migration or schema versioning system.
- **Backup/Restore:** No built-in backup or restore; handle externally.
- **Fine-grained ACLs:** Only `user` and `admin` roles; no per-collection/field ACL.
- **WebSocket/Realtime:** No realtime or WebSocket support.
- **File/Binary Storage:** No file uploads or binary/blob storage.
- **Scheduled Tasks:** No cron or scheduled task support.
- **Encryption at Rest:** No built-in data encryption at rest.
- **Admin UI:** API-only; no built-in web UI or dashboard.
- **HTTP Methods:** Only supports `GET`, `POST`, and `OPTIONS` (no `PUT`, `PATCH`, or `DELETE`).
- **Public endpoints:** `/health`, `/doc`, `/doc/llms.md`, `/doc/llms.txt`, `/doc/llms.json`.

### Design Constraints

- Collection names: lowercase, snake_case.
- Field names: unique per collection.
- No joins; handle relations at the application layer.

### Rules: Do's

- Moon is schema-on-demand.
- Always check or create collections before inserting data.
- Use API keys for server-side apps; JWT for user-facing auth.
- Endpoints follow AIP-136 custom actions (colon separator).
- Rate limits: JWT 100/min/user, API Key 1,000/min/key.
- User roles: `user` (limited), `admin` (full).
- Input validation: types and constraints enforced; always sanitize input.
- **Always use HTTPS in production.** (All curl examples assume HTTPS for production use.)
- Set explicit allowed origins for CORS.

### Rules: Don'ts

- Don’t use joins, transactions, triggers, or background jobs—handle these in your app.
- Don’t assume foreign keys; manage relations manually.

### Data Types

Supported column data types:

| Type        | Description |
|-------------|-------------|
| `string`    | Text values of any length (maps to TEXT in SQL) |
| `integer`   | 64-bit whole numbers |
| `decimal`   | For decimal values. API input/output uses strings (e.g., `"199.99"`), default 2 decimal places |
| `boolean`   | true/false values |
| `datetime`  | Date/time in RFC3339 or ISO 8601 format (e.g., 2023-01-31T13:45:00Z) |
| `json`      | Arbitrary JSON object or array |

***Note:*** Aggregation functions (sum, avg, min, max) are supported on both `integer` and `decimal` field types.

**Default Values by Type** 

- Default values are applied during collection creation if not explicitly provided.
- Defaults are assigned only to nullable fields.
- Non-nullable fields do not receive defaults and must always be included in API requests.

| Type | Default Value | Notes |
|------|--------------|-------|
| `string` | `""` (empty string) | Applied only if field is nullable |
| `integer` | `0` | Applied only if field is nullable |
| `decimal` | `"0.00"` | Applied only if field is nullable |
| `boolean` | `false` | Applied only if field is nullable |
| `datetime` | `null` | Applied for nullable fields |
| `json` | `"{}"` (empty object) | Applied only if field is nullable |

### Authentication

Except for documentation and health endpoints, all other endpoints require authentication. To access protected endpoints, include the ```Authorization: Bearer <TOKEN>``` header in your requests

Supported authentication types:

- **JWT tokens** (`eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`) – for interactive users
- **API keys** (`moon_live_<64_chars>`) – for service integrations
- Both use the same `Authorization: Bearer` header format

**JWT Token Example :** JWT tokens are used for interactive users and are obtained from `POST /auth:login`:

```bash
# Login and save tokens to environment variables
response=$(curl -s -X POST "http://localhost:6006/auth:login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"moonadmin12#"}')
ACCESS_TOKEN=$(echo $response | jq -r '.access_token')
REFRESH_TOKEN=$(echo $response | jq -r '.refresh_token')

# Use in subsequent requests
curl -s "http://localhost:6006/collections:list" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**API Key Example :** API keys are used for service integrations and are obtained from `POST /apikeys:create`:

```bash
# Create an API key (requires admin role)
curl -X POST "http://localhost:6006/apikeys:create" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Service Key","role":"user"}' | jq .

# Use the API key in requests (same Authorization header format)
curl -s "http://localhost:6006/collections:list" \
  -H "Authorization: Bearer moon_live_abc123..." | jq .
```

---

## Response Format

All successful responses return JSON with relevant data.

| Status Code | Meaning |
|-------------|---------|
| `200 OK`      | Successful GET requests                                        |
| `201 Created` | Successful POST requests creating resources                    |
| `400 Bad Request` | Invalid input, missing required field, invalid filter operator |
| `404 Not Found`   | Collection/record not found                             |
| `409 Conflict`    | Collection/record already exists                                  |
| `500 Internal Server Error` | Server error                                     |

### Success Responses (200 or 201)

```json
{
  "message": "Collection created successfully",
  "collection": {
    "name": "products",
    "columns": [...]
  }
}
```

### Error Responses

All errors follow a consistent JSON structure:

```json
{
  "error": "Error message describing what went wrong",
  "code": {HTTP status error code}
}
```
