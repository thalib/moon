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
| `boolean`  | True/false values                       | INTEGER  | BOOLEAN      | BOOLEAN      |
| `datetime` | Date and time (RFC3339/ISO 8601 format) | TEXT     | TIMESTAMP    | TIMESTAMP    |
| `json`     | Arbitrary JSON objects or arrays        | TEXT     | JSON         | JSON         |
| `decimal`  | Exact numeric values (e.g., price)      | NUMERIC  | NUMERIC(19,2)| DECIMAL(19,2)|

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

- The `/health` endpoint provides minimal, non-sensitive information for liveness checks
- Returns a simple JSON response with three fields:
  - `status`: Service liveness (`live` or `down`)
  - `name`: Service name (always `moon`)
  - `version`: Service version in format `{major}.{minor}` (e.g., `1.99`)
- Always returns HTTP 200, even when the service is down
- Clients must check the `status` field to determine service health
- Does not expose internal details like database type, collection count, or consistency status

**Example health response:**

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

### Systemd Integration

A systemd service file is provided at `samples/moon.service` for production deployment:

```bash
# Install service
sudo cp samples/moon.service /etc/systemd/system/
sudo systemctl daemon-reload

# Start service
sudo systemctl start moon

# Enable on boot
sudo systemctl enable moon

# Check status
sudo systemctl status moon
```

### Test Scripts

Test scripts are located in the `scripts/` directory and provide examples of API operations:

- All test scripts support the `PREFIX` environment variable for testing with custom URL prefixes
- Example: `PREFIX=/api/v1 ./scripts/collection.sh` or `PREFIX="" ./scripts/health.sh`

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

### B. Data Access (`/{collectionName}`)

These endpoints manage the records within a specific collection.

**Note:** All endpoints below are shown without a prefix. If a prefix is configured, prepend it to all paths.

| Endpoint               | Method | Purpose                                            |
| ---------------------- | ------ | -------------------------------------------------- |
| `GET /{name}:list`     | `GET`  | Fetch all records from the specified table.        |
| `GET /{name}:get`      | `GET`  | Fetch a single record by its unique ID.            |
| `POST /{name}:create`  | `POST` | Insert a new record (validated against the cache). |
| `POST /{name}:update`  | `POST` | Update an existing record.                         |
| `POST /{name}:destroy` | `POST` | Delete a record from the table.                    |

#### Identifiers

- Records use a ULID as the external identifier.
- The database stores an auto-increment `id` column (internal use only) and a `ulid` column (ULID string).
- API responses expose the `ulid` column as `id` for simplicity.
- The internal auto-increment `id` is never exposed via the API.

#### Advanced Query Parameters for `/{name}:list`

The list endpoint supports powerful query parameters for filtering, sorting, searching, and field selection:

**Filtering:**

- Syntax: `?column[operator]=value`
- Operators: `eq` (equal), `ne` (not equal), `gt` (greater than), `lt` (less than), `gte` (greater/equal), `lte` (less/equal), `like` (pattern match), `in` (in list)
- Example: `?price[gt]=100&category[eq]=electronics`
- Multiple filters are combined with AND logic

**Sorting:**

- Syntax: `?sort=field` (ascending) or `?sort=-field` (descending)
- Multiple fields: `?sort=-created_at,name` (comma-separated)
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

**Combined Example:**

```
GET /products:list?q=laptop&price[gt]=500&sort=-price&fields=name,price&limit=10
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

| Endpoint            | Method | Purpose                                                              |
| ------------------- | ------ | -------------------------------------------------------------------- |
| `GET /doc/`         | `GET`  | Retrieve HTML documentation (single-page, styled)                    |
| `GET /doc/md`       | `GET`  | Retrieve Markdown documentation (for AI agents and markdown readers) |
| `POST /doc:refresh` | `POST` | Clear cached documentation and force regeneration                    |

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

- **Documentation Endpoints:** Human- and AI-readable documentation is available via `/doc/` (HTML) and `/doc/md` (Markdown) endpoints (see Section 2.D for details).
- **Middleware Security:** A high-speed JWT and API Key layer that enforces simple allow/deny permissions per endpoint before the request reaches the dynamic handlers.
- **Advanced Auth Controls:**
  - JWT role-based authorization per path
  - API key permissions per endpoint (read/write/delete/admin)
  - Protected/unprotected path lists and protect-by-default mode

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
