# Moon - Dynamic Headless Engine

This document outlines the design for a high-performance, API-first backend built in **Go**. The system allows for real-time, migration-less database management via REST APIs using a **custom-action pattern** and **in-memory schema caching**.

## 1. System Philosophy

- **Migration-Less Data Modeling:** Database tables and columns are created, modified, and deleted via API calls rather than manual migration files.
- **AIP-136 Custom Actions:** APIs use a colon separator (`:`) to distinguish between the resource and the action, providing a predictable and AI-friendly interface.
- **Zero-Latency Validation:** An **In-Memory Schema Registry** (using `sync.Map`) stores the current database structure, allowing the server to validate requests in nanoseconds before hitting the disk.
- **Resource Efficiency:** Targeted to run with a memory footprint under **50MB**, optimized for cloud-native and edge deployments.
- **Database Default:** SQLite is used as the default database if no other is specified. For most development and testing scenarios, you do not need to configure a database connection string unless you want to use Postgres or MySQL.

## Configuration Architecture

The system uses YAML-only configuration with centralized defaults:

- **YAML Configuration Only:** All configuration is stored in YAML format at `/etc/moon.conf` (default) or custom path via `--config` flag
- **No Environment Variables:** Configuration values must be set in the YAML file - no environment variable overrides
- **Centralized Defaults:** All default values are defined in the `config.Defaults` struct to eliminate hardcoded literals
- **Immutable State:** On startup, the configuration is parsed into a global, read-only `AppConfig` struct to prevent accidental runtime mutations and ensure thread safety

### Configuration Structure

```yaml
server:
  host: "0.0.0.0"      # Default: 0.0.0.0
  port: 6006           # Default: 6006

database:
  connection: "sqlite" # Default: sqlite (options: sqlite, postgres, mysql)
  database: "/opt/moon/sqlite.db"  # Default: /opt/moon/sqlite.db
  user: ""             # Default: "" (empty for SQLite)
  password: ""         # Default: "" (empty for SQLite)
  host: "0.0.0.0"      # Default: 0.0.0.0

logging:
  path: "/var/log/moon" # Default: /var/log/moon

jwt:
  secret: ""           # REQUIRED - must be set in config file
  expiry: 3600         # Default: 3600 seconds (1 hour)

apikey:
  enabled: false       # Default: false
  header: "X-API-KEY"  # Default: X-API-KEY
```

### Running Modes

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

## 2. API Endpoint Specification

The system uses a strict pattern to ensure that AI agents and developers can interact with any collection without new code deployment.

- **RESTful API:** A standardized, versioned JSON API (`/api/v1`) that follows strict predictable patterns, making it easy for AI to generate documentation.

### A. Schema Management (`/collections`)

These endpoints manage the database tables and metadata.

| Endpoint                    | Method | Purpose                                                |
| --------------------------- | ------ | ------------------------------------------------------ |
| `GET /collections:list`     | `GET`  | List all managed collections from the cache.           |
| `GET /collections:get`      | `GET`  | Retrieve the schema (fields/types) for one collection. |
| `POST /collections:create`  | `POST` | Create a new table in the database.                    |
| `POST /collections:update`  | `POST` | Modify table columns (add/remove/rename).              |
| `POST /collections:destroy` | `POST` | Drop the table and purge it from the cache.            |

### B. Data Access (`/{collectionName}`)

These endpoints manage the records within a specific collection.

| Endpoint               | Method | Purpose                                            |
| ---------------------- | ------ | -------------------------------------------------- |
| `GET /{name}:list`     | `GET`  | Fetch all records from the specified table.        |
| `GET /{name}:get`      | `GET`  | Fetch a single record by its unique ID.            |
| `POST /{name}:create`  | `POST` | Insert a new record (validated against the cache). |
| `POST /{name}:update`  | `POST` | Update an existing record.                         |
| `POST /{name}:destroy` | `POST` | Delete a record from the table.                    |

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
- Returns only requested fields (ulid always included)
- Example: `?fields=name,price`
- Reduces payload size for large tables

**Combined Example:**
```
GET /products:list?q=laptop&price[gt]=500&sort=-price&fields=name,price&limit=10
```

### C. Aggregation Operations (`/{collectionName}`)

These endpoints provide server-side aggregation for analytics without fetching full datasets.

| Endpoint                    | Method | Purpose                                     |
| --------------------------- | ------ | ------------------------------------------- |
| `GET /{name}:count`         | `GET`  | Count records in the collection.            |
| `GET /{name}:sum?field=...` | `GET`  | Sum values of a numeric field.              |
| `GET /{name}:avg?field=...` | `GET`  | Calculate average of a numeric field.       |
| `GET /{name}:min?field=...` | `GET`  | Find minimum value of a numeric field.      |
| `GET /{name}:max?field=...` | `GET`  | Find maximum value of a numeric field.      |

**Parameters:**
- `field` (query): Required for `:sum`, `:avg`, `:min`, `:max`. Must be a numeric field (integer or float).
- Filtering: All aggregation endpoints support the same filtering syntax as `:list` (e.g., `?price[gt]=100`)
- Filters are applied at the database level before aggregation

**Response Format:**
```json
{
  "value": <number>
}
```

**Examples:**
```bash
# Count all orders
GET /orders:count
# Response: {"value": 150}

# Sum total sales
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
- Field must be numeric type (integer or float) for `:sum`, `:avg`, `:min`, `:max`
- Invalid field or missing field parameter returns `400 Bad Request`
- Unknown collection returns `404 Not Found`

## 3. Architecture: The Dynamic Data Flow

The server acts as a "Smart Bridge" between the user and the database.

1. **Ingress:** The Go router (e.g., Gin) parses the path `/:name:action`.
2. **Validation:** The server checks the **In-Memory Registry**. If the collection exists in RAM, it validates the JSON body fields against the cached column types.
3. **SQL Generation:** A dynamic query builder generates a sanitized, parameterized SQL statement.
4. **Persistence:** The query is executed against the database (PostgreSQL, MySQL, or SQLite).
5. **Reactive Cache:** If the action was a schema change (e.g., `:update` on collections), the server immediately refreshes that specific map entry in the registry.

## Interface & Integration Layer

- **Dynamic OpenAPI:** The Swagger/OpenAPI documentation is generated dynamically from the **In-Memory Cache**, ensuring the UI always reflects the current DB state.
- **Dynamic OpenAPI:** The Swagger/OpenAPI documentation is generated dynamically from the **In-Memory Cache**, and always includes authentication/authorization requirements and example payloads for each endpoint, in addition to reflecting the current DB schema.
- **Middleware Security:** A high-speed JWT and API Key layer that enforces simple allow/deny permissions per endpoint before the request reaches the dynamic handlers.

## 4. Design for AI Maintainability

- **Predictable Interface:** By standardizing the `:action` suffix, AI agents can guess the correct endpoint for any new collection with 100% accuracy.
- **Statically Typed Logic:** Although data is dynamic (`map[string]any`), the internal validation logic is strictly typed, preventing AI-generated bugs from corrupting the database.
- **Test-Driven Development:** Every module and feature is covered by automated tests (unit, integration, and API tests). Integration tests mock the database to ensure safe refactoring (e.g., of the SQL builder) and to guarantee the API contract is never broken. Tests are the foundation for all new code and refactoring.
- **Test Coverage:** The project aims for maximum possible test coverage, including both unit and integration tests, to ensure reliability and maintainability.
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

**Example:**

```sh
curl -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"field":"value"}' http://localhost:8080/products:create
```
