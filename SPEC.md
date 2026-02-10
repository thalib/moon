# Moon - Dynamic Headless Engine

This document outlines the design for a high-performance, API-first backend built in **Go**. The system allows for real-time, migration-less database management via REST APIs using a **custom-action pattern** and **in-memory schema caching**.

## 1. System Philosophy

- **Migration-Less Data Modeling:** Database tables and columns are created, modified, and deleted via API calls rather than manual migration files.
- **AIP-136 Custom Actions:** APIs use a colon separator (`:`) to distinguish between the resource and the action, providing a predictable and AI-friendly interface.
- **Zero-Latency Validation:** An **In-Memory Schema Registry** (using `sync.Map`) stores the current database structure, allowing the server to validate requests in nanoseconds before hitting the disk.
- **Resource Efficiency:** Targeted to run with a memory footprint under **50MB**, optimized for cloud-native and edge deployments.
- **Database Default:** SQLite is used as the default database if no other is specified. For most development and testing scenarios, you do not need to configure a database connection string unless you want to use Postgres or MySQL.

## Data Types

Moon supports a simplified, portable type system that maps consistently across all supported databases (SQLite, PostgreSQL, MySQL). This design prioritizes simplicity and predictability over fine-grained type control.

### Supported Data Types

| API Type   | Description                             | SQLite   | PostgreSQL   | MySQL        |
| ---------- | --------------------------------------- | -------- | ------------ | ------------ |
| `string`   | Text values of any length               | TEXT     | TEXT         | TEXT         |
| `integer`  | 64-bit integer values                   | INTEGER  | BIGINT       | BIGINT       |
| `decimal`  | Exact numeric values (e.g., price)      | NUMERIC  | NUMERIC(19,2)| DECIMAL(19,2)|
| `boolean`  | True/false values                       | INTEGER  | BOOLEAN      | BOOLEAN      |
| `datetime` | Date and time (RFC3339/ISO 8601 format) | TEXT     | TIMESTAMP    | TIMESTAMP    |
| `json`     | Arbitrary JSON objects or arrays        | TEXT     | JSON         | JSON         |

### Decimal Type

The `decimal` type provides **exact, deterministic numeric handling** for precision-critical values such as price, amount, weight, tax, and quantity. This addresses the inherent precision errors in floating-point arithmetic.

**API Representation:**
- Input and output are **strings** (e.g., `"199.99"`, `"-42.75"`, `"0.01"`)
- Preserves precision across serialization and deserialization
- Supports SQL aggregation functions (`SUM`, `AVG`, `MIN`, `MAX`)

**Validation:**
- Default scale: 2 decimal places
- Maximum scale: 10 decimal places
- No scientific notation allowed
- No locale-specific separators (e.g., no comma thousands separator)

**Valid formats:**
- `"10"`, `"10.50"`, `"1299.99"`, `"-42.75"`, `"0.01"`

**Invalid formats:**
- `"abc"` (non-numeric)
- `"1e10"` (scientific notation)
- `"10.999"` (exceeds default scale of 2)
- `"10."` (trailing decimal point)
- `".50"` (leading decimal point)

**Example usage:**
```json
{
  "name": "products",
  "columns": [
    {"name": "price", "type": "decimal", "nullable": false},
    {"name": "tax", "type": "decimal", "nullable": true}
  ]
}
```

### Design Rationale

- **No `float` type:** Floating-point numbers are discouraged due to precision issues. Use `integer` for whole numbers or `decimal` for exact precision values like currency and measurements.
- **No `text` vs `string` distinction:** All string data maps to `TEXT` for simplicity. There is no VARCHAR length limit enforcement at the database level.
- **JSON storage:** JSON data is stored as TEXT in SQLite and native JSON in PostgreSQL/MySQL.
- **Boolean storage:** SQLite uses INTEGER (0/1) for boolean values; PostgreSQL and MySQL use native BOOLEAN.
- **Decimal storage:** Uses native NUMERIC/DECIMAL types for exact arithmetic. API exposes values as strings to preserve precision in JSON serialization.

### Migration from Previous Versions

If upgrading from a previous version that supported `text` or `float` types:

- **`text`** columns should be changed to `string` - behavior is identical
- **`float`** columns should be changed to `decimal` for exact precision or `integer` for whole numbers

## Validation Constraints

Moon enforces strict validation rules to ensure data integrity and prevent naming conflicts.

### Collection Name Constraints

| Constraint | Value | Notes |
|------------|-------|-------|
| Minimum length | 2 characters | Single-character names are not allowed |
| Maximum length | 63 characters | Matches PostgreSQL identifier limit |
| Pattern | `^[a-zA-Z][a-zA-Z0-9_]*$` | Must start with letter, alphanumeric + underscores |
| Case normalization | Lowercase | Names are automatically converted to lowercase |
| Reserved endpoints | `collections`, `auth`, `users`, `apikeys`, `doc`, `health` | Case-insensitive |
| System prefix | `moon_*`, `moon` | Reserved for internal system tables |
| SQL keywords | 100+ keywords | `select`, `insert`, `update`, `delete`, `table`, etc. |

### Column Name Constraints

| Constraint | Value | Notes |
|------------|-------|-------|
| Minimum length | 3 characters | Short names like `id`, `at` are not allowed |
| Maximum length | 63 characters | Matches PostgreSQL identifier limit |
| Pattern | `^[a-z][a-z0-9_]*$` | Lowercase only, must start with letter |
| Reserved names | `id`, `ulid` | System columns, automatically created |
| SQL keywords | 100+ keywords | Same list as collection names |

**Important:** Unlike collection names, column names are NOT auto-normalized to lowercase. Uppercase characters will be rejected with an error.

### System Limits

| Limit | Default | Configurable | Notes |
|-------|---------|--------------|-------|
| Max collections | 1000 | Yes (`limits.max_collections`) | Per server |
| Max columns | 100 | Yes (`limits.max_columns_per_collection`) | Per collection (includes system columns) |
| Max filters | 20 | Yes (`limits.max_filters_per_request`) | Per request |
| Max sort fields | 5 | Yes (`limits.max_sort_fields_per_request`) | Per request |

### Pagination Limits

| Limit | Default | Configurable | Notes |
|-------|---------|--------------|-------|
| Min page size | 1 | No | Hardcoded minimum |
| Default page size | 15 | Yes (`pagination.default_page_size`) | When no limit specified |
| Max page size | 200 | Yes (`pagination.max_page_size`) | Maximum allowed |

### Deprecated Types

The following types are deprecated and will return an error:

| Deprecated | Use Instead | Rationale |
|------------|-------------|-----------|
| `text` | `string` | Redundant - both map to TEXT |
| `float` | `decimal` or `integer` | Precision issues with floating-point |

## API Standards

Moon implements industry-standard API patterns for consistent client experience.

### HTTP Methods

**Moon only supports GET and POST HTTP methods.** All other HTTP methods (PUT, DELETE, PATCH, etc.) are not supported and will return a `405 Method Not Allowed` error.

- **GET** is used for read operations (list, get, count, aggregation queries)
- **POST** is used for write operations (create, update, destroy) and authentication operations
- **OPTIONS** is supported for CORS preflight requests only

This design choice:
- Simplifies routing and middleware logic
- Works universally with all HTTP clients and proxies
- Follows the AIP-136 custom actions pattern where the action is in the URL (`:create`, `:update`, `:destroy`)
- Ensures compatibility with restrictive network environments that may filter uncommon HTTP methods

### Rate Limiting Headers

When rate limiting is enabled, all responses include rate limit headers:

| Header | Description | Example |
|--------|-------------|---------|
| `X-RateLimit-Limit` | Maximum requests per window | `100` |
| `X-RateLimit-Remaining` | Remaining requests in window | `87` |
| `X-RateLimit-Reset` | Unix timestamp when window resets | `1704067200` |
| `Retry-After` | Seconds until retry (429 responses only) | `60` |

### Error Response Format

All error responses follow a consistent JSON structure:

```json
{
  "error": "human-readable error message",
  "code": "ERROR_CODE"
}
```

With optional details:

```json
{
  "error": "validation failed",
  "code": "VALIDATION_ERROR",
  "details": {
    "field": "email",
    "expected": "valid email format"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Input validation failed |
| `INVALID_JSON` | 400 | Malformed JSON |
| `INVALID_ULID` | 400 | Invalid ULID format |
| `PAGE_SIZE_EXCEEDED` | 400 | Page size exceeds maximum |
| `COLLECTION_NOT_FOUND` | 404 | Collection does not exist |
| `RECORD_NOT_FOUND` | 404 | Record not found |
| `DUPLICATE_COLLECTION` | 409 | Collection name already exists |
| `MAX_COLLECTIONS_REACHED` | 409 | Maximum collections limit reached |
| `MAX_COLUMNS_REACHED` | 409 | Maximum columns limit reached |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

### CORS Support

Cross-Origin Resource Sharing (CORS) can be enabled via configuration:

```yaml
cors:
  enabled: true
  allowed_origins:
    - "https://app.example.com"
  allowed_methods:
    - GET
    - POST
    - OPTIONS
  allowed_headers:
    - Content-Type
    - Authorization
  allow_credentials: true
  max_age: 3600
  
  # Endpoint-specific CORS registration (PRD-058)
  endpoints:
    - path: "/health"
      pattern_type: "exact"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
    
    - path: "/doc/"
      pattern_type: "prefix"
      allowed_origins: ["*"]
      allowed_methods: ["GET", "OPTIONS"]
      allowed_headers: ["Content-Type"]
      allow_credentials: false
      bypass_auth: true
```

**Note:** Only GET, POST, and OPTIONS methods are supported by the Moon server. Including other methods (PUT, DELETE, PATCH) in the `allowed_methods` configuration will not enable them on the server.

**CORS Endpoint Registration (PRD-058):**

Moon supports dynamic CORS endpoint registration with pattern matching:

- **Pattern Types:**
  - `exact`: Matches exact path only (e.g., `/health` matches `/health` but not `/health/status`)
  - `prefix`: Matches path prefix (e.g., `/doc/` matches `/doc/api`, `/doc/llms-full.txt`. Note: `/doc/` does not match `/doc` without trailing slash)
  - `suffix`: Matches path suffix (e.g., `*.json` matches `/data/users.json`)
  - `contains`: Matches if path contains substring (e.g., `/public/` matches any path with `/public/`)

- **Priority:** When multiple patterns match, the most specific match is used:
  1. Exact matches (highest priority)
  2. Longest prefix matches
  3. Longest suffix matches
  4. Longest contains matches
  5. Global CORS configuration (fallback)

- **Authentication Bypass:** Set `bypass_auth: true` to skip authentication for public endpoints (health, docs, status).

- **Default Endpoints:** If `cors.endpoints` is not specified, these defaults are applied:
  - `/health` (exact, `*`, no auth)
  - `/doc/` (prefix, `*`, no auth - matches all paths starting with `/doc/` including `/doc/` and `/doc/llms-full.txt`)

CORS headers exposed to browsers:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`
- `X-Request-ID`

### Sensitive Data Redaction

Moon automatically redacts sensitive fields in logs to prevent credential leakage:

**Default Sensitive Fields:**
- `password`, `token`, `secret`, `api_key`, `apikey`
- `authorization`, `jwt`, `refresh_token`, `access_token`
- `client_secret`, `private_key`, `credential`, `auth`

**Configuration:**
```yaml
logging:
  redact_sensitive: true  # Default: true
  additional_sensitive_fields:
    - "ssn"
    - "credit_card"
```

## Configuration Architecture

The system uses YAML-only configuration with centralized defaults:

- **YAML Configuration Only:** All configuration is stored in YAML format at `/etc/moon.conf` (default) or custom path via `--config` flag
- **No Environment Variables:** Configuration values must be set in the YAML file - no environment variable overrides
- **Centralized Defaults:** All default values are defined in the `config.Defaults` struct to eliminate hardcoded literals
- **Immutable State:** On startup, the configuration is parsed into a global, read-only `AppConfig` struct to prevent accidental runtime mutations and ensure thread safety

### Configuration Structure

```yaml
server:
  host: "0.0.0.0" # Default: 0.0.0.0
  port: 6006 # Default: 6006
  prefix: "" # Default: "" (empty - no prefix)

database:
  connection: "sqlite" # Default: sqlite (options: sqlite, postgres, mysql)
  database: "/opt/moon/sqlite.db" # Default: /opt/moon/sqlite.db
  user: "" # Default: "" (empty for SQLite)
  password: "" # Default: "" (empty for SQLite)
  host: "0.0.0.0" # Default: 0.0.0.0

logging:
  path: "/var/log/moon" # Default: /var/log/moon

jwt:
  secret: "" # REQUIRED - must be set in config file
  expiry: 3600 # Default: 3600 seconds (1 hour)

apikey:
  enabled: false # Default: false
  header: "X-API-KEY" # Default: X-API-KEY

recovery:
  auto_repair: true # Default: true - automatically repair consistency issues
  drop_orphans: false # Default: false - drop orphaned tables (if false, register them)
  check_timeout: 5 # Default: 5 seconds - timeout for consistency checks

pagination:
  default_page_size: 15 # Default: 15 - returned when no limit specified
  max_page_size: 200 # Default: 200 - maximum allowed page size

limits:
  max_collections: 1000 # Default: 1000 - maximum collections per server
  max_columns_per_collection: 100 # Default: 100 - including system columns
  max_filters_per_request: 20 # Default: 20 - filter parameters per request
  max_sort_fields_per_request: 5 # Default: 5 - sort fields per request
```

### Recovery and Consistency Checking

Moon includes robust consistency checking and recovery logic that ensures the in-memory schema registry remains synchronized with the physical database tables across restarts and failures.

**On Startup:**

- Moon performs an automatic consistency check comparing the registry with physical database tables
- If inconsistencies are detected, they are logged with detailed information
- With `auto_repair: true` (default), Moon automatically repairs inconsistencies:
  - **Orphaned registry entries** (registered but table doesn't exist): Removed from registry
  - **Orphaned tables** (table exists but not registered):
    - If `drop_orphans: false` (default): Table schema is inferred and registered
    - If `drop_orphans: true`: Table is dropped from database

**Consistency Check:**

- Runs within the configured timeout (default 5 seconds)
- Non-blocking with configurable timeout to prevent indefinite startup delays
- Results are logged and displayed during startup
- Startup fails if critical issues cannot be repaired

**Health Endpoint:**

- The `/health` endpoint provides health check information for liveness and readiness checks
- Returns a JSON response with four fields:
  - `status`: Service health status (`ok` or `degraded`)
  - `database`: Database connectivity status (`ok` or `error`)
  - `version`: Service version string (e.g., `1.0.0`)
  - `timestamp`: ISO 8601 timestamp of the health check
- Always returns HTTP 200, even when the service is degraded
- Clients must check the `status` field to determine service health
- Does not expose internal details like database type, collection count, or consistency status

**Example health response:**

```json
{
  "status": "ok",
  "database": "ok",
  "version": "1.0.0",
  "timestamp": "2026-02-03T13:58:53Z"
}
```

```json
{
  "status": "live",
  "name": "moon",
  "version": "1.99"
}
```

### Running Modes

#### Preflight Checks

Before the server starts, Moon performs filesystem preflight checks:

- Ensures the logging directory exists (and creates it if missing)
- For SQLite, ensures the database parent directory exists (and creates it if missing)
- In daemon mode, truncates the log file to start fresh

#### Console Mode (Default)

```bash
moon --config /etc/moon.conf
```

- Runs in foreground attached to terminal
- Logs output to both stdout/stderr AND log file (`/var/log/moon/main.log` or path specified in config)
- Stdout logs use console format (colorized, human-readable)
- File logs use simple format (`[LEVEL](TIMESTAMP): {MESSAGE}`)
- Process terminates when terminal closes or Ctrl+C is pressed

#### Daemon Mode

```bash
moon --daemon --config /etc/moon.conf
# or shorthand
moon -d --config /etc/moon.conf
```

- Runs as background daemon process
- Logs written only to `/var/log/moon/main.log` (or path specified in config)
- PID file written to `/var/run/moon.pid`
- Process continues after terminal closes
- Supports graceful shutdown via SIGTERM/SIGINT

## 2. API Endpoint Specification

The system uses a strict pattern to ensure that AI agents and developers can interact with any collection without new code deployment.

- **RESTful API:** A standardized API following strict predictable patterns, making it easy for AI to generate documentation.
- **Configurable Prefix:** All API endpoints are mounted under a configurable URL prefix (default: empty string).
  - Default (no prefix): `/health`, `/collections:list`, `/{collection}:list`
  - With `/api/v1` prefix: `/api/v1/health`, `/api/v1/collections:list`, `/api/v1/{collection}:list`
  - With custom prefix: `/{prefix}/health`, `/{prefix}/collections:list`, `/{prefix}/{collection}:list`

### A. Schema Management (`/collections`)

These endpoints manage the database tables and metadata.

**Note:** All endpoints below are shown without a prefix. If a prefix is configured (e.g., `/api/v1`), prepend it to all paths.

| Endpoint                    | Method | Purpose                                                |
| --------------------------- | ------ | ------------------------------------------------------ |
| `GET /collections:list`     | `GET`  | List all managed collections from the cache.           |
| `GET /collections:get`      | `GET`  | Retrieve the schema (fields/types) for one collection. |
| `POST /collections:create`  | `POST` | Create a new table in the database.                    |
| `POST /collections:update`  | `POST` | Modify table columns (add/remove/rename).              |
| `POST /collections:destroy` | `POST` | Drop the table and purge it from the cache.            |

#### Collections List Response Format (PRD-065)

The `GET /collections:list` endpoint returns detailed information about each collection, including record counts.

**Response Format:**
```json
{
  "collections": [
    { "name": "customers", "records": 150 },
    { "name": "products", "records": 42 },
    { "name": "orders", "records": 328 }
  ],
  "count": 3
}
```

**Fields:**
- `collections` (array): Array of collection objects, each containing:
  - `name` (string): The collection name
  - `records` (integer): Total number of records in the collection. Returns `-1` if count cannot be retrieved (e.g., database error). A value of 0 indicates an empty collection, while -1 specifically indicates an error condition.
- `count` (integer): Total number of collections returned

**Note:** This is a breaking change from the previous format which returned collection names as a simple string array. Clients must be updated to consume the new object-based format.

### B. Data Access (`/{collectionName}`)

These endpoints manage the records within a specific collection.

**Note:** All endpoints below are shown without a prefix. If a prefix is configured, prepend it to all paths.

| Endpoint               | Method | Purpose                                            |
| ---------------------- | ------ | -------------------------------------------------- |
| `GET /{name}:list`     | `GET`  | Fetch all records from the specified table.        |
| `GET /{name}:get`      | `GET`  | Fetch a single record by its unique ID.            |
| `GET /{name}:schema`   | `GET`  | Retrieve the schema for a specific collection.     |
| `POST /{name}:create`  | `POST` | Insert a new record (validated against the cache). |
| `POST /{name}:update`  | `POST` | Update an existing record.                         |
| `POST /{name}:destroy` | `POST` | Delete a record from the table.                    |

#### Batch Operations (PRD-064)

The `:create`, `:update`, and `:destroy` endpoints support both **single-object** and **batch** modes, allowing you to process multiple records in a single request. This feature reduces network overhead and improves throughput for bulk operations.

**Overview:**

- **Automatic Detection:** The server detects batch mode by inspecting the request body. If the body is a JSON array, batch processing is triggered. If it's a single JSON object, single-object mode is used.
- **Atomic Mode (Default):** By default, batch operations are atomic. All operations succeed or all fail. If any record fails validation or processing, the entire batch is rejected with a `400 Bad Request` response.
- **Best-Effort Mode:** Set `?atomic=false` to enable best-effort processing. Each record is processed independently. The server returns `HTTP 207 Multi-Status` with individual success/error details for each record.
- **Size Limits:** Batches are subject to configurable limits to prevent resource exhaustion:
  - **Max Batch Size:** Default 500 records per request (configurable via `api.batch.max_size`)
  - **Max Payload Size:** Default 2MB (configurable via `api.batch.max_payload_bytes`)
- **Backward Compatibility:** Single-object requests continue to work exactly as before. Batch mode is an additive feature.

**Request Format:**

Single-object mode (original behavior):
```json
{
  "name": "Alice",
  "email": "alice@example.com"
}
```

Batch mode (array of objects):
```json
[
  {"name": "Alice", "email": "alice@example.com"},
  {"name": "Bob", "email": "bob@example.com"},
  {"name": "Charlie", "email": "charlie@example.com"}
]
```

**Query Parameters:**

- `atomic` (boolean, default: `true`)
  - `true`: All operations succeed or all fail (returns `200 OK` or `400 Bad Request`)
  - `false`: Best-effort processing (returns `207 Multi-Status` with per-record results)

**Response Codes:**

- `200 OK`: Atomic mode - all records processed successfully
- `207 Multi-Status`: Best-effort mode - partial success (some records succeeded, some failed)
- `400 Bad Request`: Atomic mode - at least one record failed validation or processing
- `413 Payload Too Large`: Batch size or payload size limit exceeded
- `422 Unprocessable Entity`: Request body is not valid JSON or empty array

**Examples:**

**1. Batch Create (Atomic Mode - All Succeed):**

```bash
curl -X POST https://api.example.com/users:create \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[
    {"name": "Alice", "email": "alice@example.com"},
    {"name": "Bob", "email": "bob@example.com"}
  ]'
```

Response (HTTP 201 Created):
```json
{
  "data": [
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAA",
      "name": "Alice",
      "email": "alice@example.com",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FBB",
      "name": "Bob",
      "email": "bob@example.com",
      "created_at": "2024-01-15T10:30:01Z",
      "updated_at": "2024-01-15T10:30:01Z"
    }
  ],
  "message": "2 records created successfully"
}
```

**2. Batch Create (Best-Effort Mode - Partial Success):**

```bash
curl -X POST https://api.example.com/users:create?atomic=false \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[
    {"name": "Alice", "email": "alice@example.com"},
    {"name": "X", "email": "invalid"},
    {"name": "Bob", "email": "bob@example.com"}
  ]'
```

Response (HTTP 207 Multi-Status):
```json
{
  "results": [
    {
      "index": 0,
      "status": "created",
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAA",
      "data": {
        "id": "01ARZ3NDEKTSV4RRFFQ69G5FAA",
        "name": "Alice",
        "email": "alice@example.com",
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    },
    {
      "index": 1,
      "status": "failed",
      "error_code": "validation_error",
      "error_message": "validation failed: name must be at least 3 characters, email format invalid"
    },
    {
      "index": 2,
      "status": "created",
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FBB",
      "data": {
        "id": "01ARZ3NDEKTSV4RRFFQ69G5FBB",
        "name": "Bob",
        "email": "bob@example.com",
        "created_at": "2024-01-15T10:30:01Z",
        "updated_at": "2024-01-15T10:30:01Z"
      }
    }
  ],
  "summary": {
    "total": 3,
    "succeeded": 2,
    "failed": 1
  }
}
```

**3. Batch Update (Atomic Mode):**

```bash
curl -X POST https://api.example.com/users:update \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[
    {"id": "01ARZ3NDEKTSV4RRFFQ69G5FAA", "name": "Alice Updated"},
    {"id": "01ARZ3NDEKTSV4RRFFQ69G5FBB", "name": "Bob Updated"}
  ]'
```

Response (HTTP 200 OK):
```json
{
  "data": [
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAA",
      "name": "Alice Updated",
      "email": "alice@example.com",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T11:45:00Z"
    },
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FBB",
      "name": "Bob Updated",
      "email": "bob@example.com",
      "created_at": "2024-01-15T10:30:01Z",
      "updated_at": "2024-01-15T11:45:01Z"
    }
  ],
  "message": "2 records updated successfully"
}
```

**4. Batch Destroy (Atomic Mode):**

```bash
curl -X POST https://api.example.com/users:destroy \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"data": ["01ARZ3NDEKTSV4RRFFQ69G5FAA", "01ARZ3NDEKTSV4RRFFQ69G5FBB"]}'
```

Response (HTTP 200 OK):
```json
{
  "message": "2 records deleted successfully"
}
```

**5. Batch Size Exceeded Error:**

```bash
curl -X POST https://api.example.com/users:create \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[... 501 records ...]'
```

Response (HTTP 413 Payload Too Large):
```json
{
  "error": "batch size exceeds maximum allowed (500)",
  "details": "received 501 records, maximum is 500"
}
```

**Backward Compatibility:**

- **No Breaking Changes:** Existing single-object requests continue to work exactly as before.
- **Array Detection:** The server automatically detects batch mode by checking if the request body is a JSON array.
- **Single-Object Response:** Single-object requests return the original response format (a single object, not an array).
- **Opt-In Batch Mode:** Clients must explicitly send an array to trigger batch processing.

**Configuration Options:**

Configure batch operation limits in your configuration file:

```yaml
api:
  batch:
    max_size: 500              # Maximum records per batch request
    max_payload_bytes: 2097152 # Maximum payload size (2MB)
```

Or via environment variables:

```bash
MOON_API_BATCH_MAX_SIZE=500
MOON_API_BATCH_MAX_PAYLOAD_BYTES=2097152
```

**Performance Considerations:**

- Batch operations are processed in a single database transaction (atomic mode) or as individual transactions (best-effort mode).
- For large batches, consider using pagination to stay within size and payload limits.
- Monitor memory usage when processing large batches with complex objects.
- Best-effort mode (`atomic=false`) may be slower due to per-record transaction overhead.

#### Identifiers

- Records use a ULID as the external identifier.
- The database stores an auto-increment `id` column (internal use only) and a `ulid` column (ULID string).
- API responses expose the `ulid` column as `id` for simplicity.
- The internal auto-increment `id` is never exposed via the API.

#### Advanced Query Parameters for `/{name}:list`

The list endpoint supports powerful query parameters for filtering, sorting, searching, and field selection:

**Filtering:**

- Syntax: `?column[operator]=value`
- Operators:
  - Comparison: `eq` (equal), `ne` (not equal), `gt` (greater than), `lt` (less than), `gte` (greater/equal), `lte` (less/equal)
  - Pattern matching: `like` (SQL LIKE pattern with %), `contains` (substring, case-sensitive), `icontains` (substring, case-insensitive), `startswith`, `endswith`
  - List: `in` (comma-separated values, e.g., `?status[in]=active,pending`)
  - Null checks: `null` (is NULL), `notnull` (is NOT NULL)
- Example: `?price[gt]=100&category[eq]=electronics&title[contains]=widget`
- Multiple filters are combined with AND logic
- Maximum 20 filters per request

**Sorting:**

- Syntax: `?sort=field` (ascending) or `?sort=-field` (descending)
- Multiple fields: `?sort=-created_at,name` (comma-separated, max 5 fields)
- Example: `?sort=-price,name`

**Full-Text Search:**

- Syntax: `?q=searchterm`
- Searches across all text/string columns with OR logic
- Example: `?q=laptop`
- Can be combined with filters and sorting

**Field Selection:**

- Syntax: `?fields=field1,field2`
- Returns only requested fields (id always included)
- Example: `?fields=name,price`
- Reduces payload size for large tables

**Cursor Pagination:**

- Syntax: `?after=<ulid>`
- Returns `next_cursor` in the response when more results are available
- Example: `?after=01ARZ3NDEKTSV4RRFFQ69G5FBX`

**List Response Format:**

The list endpoint returns a JSON object with the following fields:

```json
{
  "data": [...],
  "total": 42,
  "next_cursor": "01ARZ3NDEKTSV4RRFFQ69G5FBX",
  "limit": 15
}
```

- `data`: Array of records matching the query
- `total`: Total count of records matching all filters (independent of limit/cursor)
- `next_cursor`: ULID cursor for next page, or null if no more data
- `limit`: Current page size

**Combined Example:**

```
GET /products:list?q=laptop&price[gt]=500&title[contains]=pro&sort=-price&fields=name,price&limit=10
```

#### Schema Retrieval

To retrieve the schema (field names, types, and constraints) for a specific collection, use the dedicated schema endpoint:

**Endpoint:** `GET /{collection}:schema`

**Response Format:**
```json
{
  "collection": "products",
  "fields": [
    { "name": "id", "type": "string", "nullable": false },
    { "name": "title", "type": "string", "nullable": false },
    { "name": "price", "type": "integer", "nullable": false },
    { "name": "description", "type": "string", "nullable": true }
  ],
  "total": 42
}
```

The `total` field contains the total number of records currently in the collection. It is always included in the schema response.

**Authentication:** Required (Bearer token or API key)

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication
- `404 Not Found`: Collection does not exist
- `500 Internal Server Error`: Unexpected errors

**Example:**
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" https://api.example.com/products:schema
```

### C. Aggregation Operations (`/{collectionName}`)

These endpoints provide server-side aggregation for analytics without fetching full datasets.

| Endpoint                    | Method | Purpose                                |
| --------------------------- | ------ | -------------------------------------- |
| `GET /{name}:count`         | `GET`  | Count records in the collection.       |
| `GET /{name}:sum?field=...` | `GET`  | Sum values of a numeric field.         |
| `GET /{name}:avg?field=...` | `GET`  | Calculate average of a numeric field.  |
| `GET /{name}:min?field=...` | `GET`  | Find minimum value of a numeric field. |
| `GET /{name}:max?field=...` | `GET`  | Find maximum value of a numeric field. |

**Parameters:**

- `field` (query): Required for `:sum`, `:avg`, `:min`, `:max`. Must be a numeric field (`integer` or `decimal`).
- Filtering: All aggregation endpoints support the same filtering syntax as `:list` (e.g., `?price[gt]=100`)
- Filters are applied at the database level before aggregation

**Response Format:**

```json
{
  "value": <number>
}
```

**Note:** Both `integer` and `decimal` type fields support aggregation. Results preserve precision for decimal calculations.

**Examples:**

```bash
# Count all orders
GET /orders:count
# Response: {"value": 150}

# Sum total sales (decimal field)
GET /orders:sum?field=total
# Response: {"value": 15750.50}

# Average order value for completed orders
GET /orders:avg?field=total&status[eq]=completed
# Response: {"value": 125.75}

# Find highest order amount
GET /orders:max?field=total
# Response: {"value": 999.99}
```

**Validation:**

- Collection must exist
- Field must exist in the collection schema
- Field must be numeric type (integer) for `:sum`, `:avg`, `:min`, `:max`
- Invalid field or missing field parameter returns `400 Bad Request`
- Unknown collection returns `404 Not Found`

### D. Documentation Endpoints

Moon provides human- and AI-readable documentation endpoints that automatically reflect the current API state.

**Note:** All endpoints below are shown without a prefix. If a prefix is configured, prepend it to all paths.

| Endpoint                     | Method | Purpose                                                              |
| ---------------------------- | ------ | -------------------------------------------------------------------- |
| `GET /doc/`                  | `GET`  | Retrieve HTML documentation (single-page, styled)                    |
| `GET /doc/llms-full.txt`     | `GET`  | Retrieve Markdown documentation (for AI agents and markdown readers) |
| `POST /doc:refresh`          | `POST` | Clear cached documentation and force regeneration                    |

**Features:**

- Generic documentation with `{collection}` placeholders
- Lists currently available collections
- Includes all endpoint categories (schema, data, aggregation)
- Quickstart guide with 5-step workflow
- Comprehensive examples with curl commands (with and without jq)
- Filtering, sorting, pagination, and field selection documentation
- Error response format and common status codes
- Table of contents for easy navigation

**Caching:**

- Documentation is generated once and cached in memory
- Responses include `Cache-Control`, `ETag`, and `Last-Modified` headers
- Supports conditional caching with `If-None-Match` (returns 304 Not Modified)
- Cache can be cleared using `POST /doc:refresh`

**Response Headers:**

- HTML: `Content-Type: text/html; charset=utf-8`
- Markdown: `Content-Type: text/markdown; charset=utf-8`
- Both: `Cache-Control: public, max-age=3600`

### E. Collection Column Operations

The `POST /collections:update` endpoint supports comprehensive column lifecycle management through four operation types that can be combined in a single request.

**Operation Order:**
Operations are executed in the following order: rename → modify → add → remove

**Request Body Structure:**

```json
{
  "name": "collection_name",
  "add_columns": [...],      // Optional: Add new columns
  "remove_columns": [...],   // Optional: Remove existing columns
  "rename_columns": [...],   // Optional: Rename existing columns
  "modify_columns": [...]    // Optional: Modify column types/constraints
}
```

**Add Columns:**

```json
{
  "name": "products",
  "add_columns": [
    {
      "name": "category",
      "type": "string",
      "nullable": true,
      "unique": false,
      "default_value": null
    }
  ]
}
```

**Remove Columns:**

```json
{
  "name": "products",
  "remove_columns": ["old_field", "deprecated_column"]
}
```

- Cannot remove system columns (`id`, `ulid`)
- Column must exist in collection

**Rename Columns:**

```json
{
  "name": "products",
  "rename_columns": [
    {
      "old_name": "description",
      "new_name": "details"
    }
  ]
}
```

- Cannot rename system columns (`id`, `ulid`)
- New name must not conflict with existing columns
- Old column must exist

**Modify Columns:**

```json
{
  "name": "products",
  "modify_columns": [
    {
      "name": "price",
      "type": "integer",
      "nullable": false,
      "unique": false,
      "default_value": "0"
    }
  ]
}
```

- Cannot modify system columns (`id`, `ulid`)
- Column must exist
- Type changes should be compatible with existing data

**Combined Operations Example:**

```json
{
  "name": "products",
  "rename_columns": [{ "old_name": "stock", "new_name": "quantity" }],
  "modify_columns": [{ "name": "price", "type": "integer", "nullable": false }],
  "add_columns": [{ "name": "brand", "type": "string", "nullable": true }],
  "remove_columns": ["deprecated_field"]
}
```

**Validation & Error Handling:**

- All operations are validated before execution
- Registry is atomically updated only after successful DDL execution
- On failure, registry is rolled back to previous state
- System columns (`id`, `ulid`) are protected from modification
- Descriptive errors returned for invalid operations

**Database Dialect Support:**

- SQLite: DROP COLUMN (3.35.0+), RENAME COLUMN (3.25.0+)
- PostgreSQL: Full support for all operations
- MySQL: Full support for all operations
- Note: SQLite MODIFY COLUMN has limited support and may require table recreation

## 3. Architecture: The Dynamic Data Flow

The server acts as a "Smart Bridge" between the user and the database.

1. **Ingress:** The Go router (e.g., Gin) parses the path `/:name:action`.
2. **Validation:** The server checks the **In-Memory Registry**. If the collection exists in RAM, it validates the JSON body fields against the cached column types.
3. **SQL Generation:** A dynamic query builder generates a sanitized, parameterized SQL statement.
4. **Persistence:** The query is executed against the database (PostgreSQL, MySQL, or SQLite).
5. **Reactive Cache:** If the action was a schema change (e.g., `:update` on collections), the server immediately refreshes that specific map entry in the registry.

## Interface & Integration Layer

- **Documentation Endpoints:** Human- and AI-readable documentation is available via `/doc/` (HTML) and `/doc/llms-full.txt` (Markdown) endpoints (see Section 2.D for details).
- **Middleware Security:** A high-speed JWT and API Key layer that enforces simple allow/deny permissions per endpoint before the request reaches the dynamic handlers.
- **Advanced Auth Controls:**
  - JWT role-based authorization per path
  - API key permissions per endpoint (read/write/delete/admin)
  - Protected/unprotected path lists and protect-by-default mode

## Authentication & Authorization

Moon requires authentication for all API endpoints except `/health`. Two authentication methods are supported:

### Authentication Methods

| Method | Header | Use Case | Rate Limit |
|--------|--------|----------|------------|
| **JWT** | `Authorization: Bearer <token>` | Interactive users (web/mobile) | 100 req/min |
| **API Key** | `Authorization: Bearer moon_live_*` | Machine-to-machine integrations | 1000 req/min |

### Roles and Permissions

Moon supports three roles with configurable write permissions:

| Role | Manage Users/Keys | Manage Collections | Read Data | Write Data |
|------|-------------------|-------------------|-----------|------------|
| **admin** | ✓ | ✓ | ✓ | ✓ |
| **user** | ✗ | ✗ | ✓ | if `can_write: true` |
| **readonly** | ✗ | ✗ | ✓ | ✗ |

- **admin**: Full system access including user management, collection schema changes, and all data operations. The `can_write` flag is ignored for admin role (always has write access).
- **user**: Read access to all collections; write access controlled by `can_write` flag (default: true for user role)
- **readonly**: Read-only access to all collections; cannot write data even if `can_write` flag is set to true

### Protected Endpoints

| Category | Endpoints | Admin | User (read-only) | User (can_write) |
|----------|-----------|-------|------------------|------------------|
| Health | `/health` | ✓ (no auth) | ✓ (no auth) | ✓ (no auth) |
| Auth | `/auth:*` | ✓ | ✓ | ✓ |
| Collections | `/collections:list`, `/collections:get` | ✓ | ✓ | ✓ |
| Collections | `/collections:create`, `/collections:update`, `/collections:destroy` | ✓ | ✗ | ✗ |
| Data Read | `/{name}:list`, `/{name}:get`, `/{name}:count/sum/avg/min/max` | ✓ | ✓ | ✓ |
| Data Write | `/{name}:create`, `/{name}:update`, `/{name}:destroy` | ✓ | ✗ | ✓ |
| Users | `/users:*` | ✓ | ✗ | ✗ |
| API Keys | `/apikeys:*` | ✓ | ✗ | ✗ |

### Rate Limits

- **JWT Users**: 100 requests/minute
- **API Keys**: 1000 requests/minute  
- **Login Attempts**: 5 per 15 minutes per IP/username

Rate limit headers included in responses:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining in window
- `X-RateLimit-Reset`: Unix timestamp when limit resets

> **📖 For detailed authentication specification**, see [SPEC_AUTH.md](SPEC_AUTH.md)

## 4. Design for AI Maintainability

- **Predictable Interface:** By standardizing the `:action` suffix, AI agents can guess the correct endpoint for any new collection with 100% accuracy.
- **Statically Typed Logic:** Although data is dynamic (`map[string]any`), the internal validation logic is strictly typed, preventing AI-generated bugs from corrupting the database.
- **Test-Driven Development:** Every module and feature is covered by automated tests (unit, integration, and API tests). Integration tests mock the database to ensure safe refactoring (e.g., of the SQL builder) and to guarantee the API contract is never broken. Tests are the foundation for all new code and refactoring. The project aims for maximum possible test coverage to ensure reliability and maintainability.
- **Strict Conventions:** By adhering to standard Go patterns, the codebase remains "recognizably structured." Both AI agents and human developers can navigate the project with 99% accuracy because files are exactly where they are expected to be.

---

## 5. Persistence Layer & Agnosticism

- **Dialect-Agnostic:** The server uses a driver-based approach. The user provides a connection string, and Moon-Go detects if it needs to use `Postgres`, `MySQL`, or `SQLite` syntax.
- **Database Type Fixed at Startup:** The database type is selected at server startup and cannot be changed at runtime.
- **Single-Tenant Focus:** Optimized as a high-speed core for a single application, ensuring maximum simplicity and maintainability.

## 6. End-User Testing (with curl)

For most API endpoints, simple curl-based testing is sufficient and recommended for end users:

- Authenticate (if required) by passing JWT/API keys in headers.
- Send requests to any endpoint (e.g., create, update, list, delete).
- Provide JSON payloads with `-d` and set `Content-Type: application/json`.
- Inspect HTTP status codes and response bodies for validation.
- Adjust URLs based on configured prefix (default: no prefix).

**Examples:**

```sh
# Without prefix (default)
curl -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"field":"value"}' http://localhost:6006/products:create

# With /api/v1 prefix
curl -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"field":"value"}' http://localhost:6006/api/v1/products:create

# With custom prefix
curl -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"field":"value"}' http://localhost:6006/moon/api/products:create
```
