# Moon - Dynamic Headless Engine

This document outlines the design for a high-performance, API-first backend built in **Go**. The system allows for real-time, migration-less database management via REST APIs using a **custom-action pattern** and **in-memory schema caching**.

## 1. System Philosophy

* **Migration-Less Data Modeling:** Database tables and columns are created, modified, and deleted via API calls rather than manual migration files.
* **AIP-136 Custom Actions:** APIs use a colon separator (`:`) to distinguish between the resource and the action, providing a predictable and AI-friendly interface.
* **Zero-Latency Validation:** An **In-Memory Schema Registry** (using `sync.Map`) stores the current database structure, allowing the server to validate requests in nanoseconds before hitting the disk.
* **Resource Efficiency:** Targeted to run with a memory footprint under **50MB**, optimized for cloud-native and edge deployments.


## 2. API Endpoint Specification

The system uses a strict pattern to ensure that AI agents and developers can interact with any collection without new code deployment.

### A. Schema Management (`/collections`)

These endpoints manage the database tables and metadata.

| Endpoint | Method | Purpose |
| --- | --- | --- |
| `GET /collections:list` | `GET` | List all managed collections from the cache. |
| `GET /collections:get` | `GET` | Retrieve the schema (fields/types) for one collection. |
| `POST /collections:create` | `POST` | Create a new table in the database. |
| `POST /collections:update` | `POST` | Modify table columns (add/remove/rename). |
| `POST /collections:destroy` | `POST` | Drop the table and purge it from the cache. |

### B. Data Access (`/{collectionName}`)

These endpoints manage the records within a specific collection.

| Endpoint | Method | Purpose |
| --- | --- | --- |
| `GET /{name}:list` | `GET` | Fetch all records from the specified table. |
| `GET /{name}:get` | `GET` | Fetch a single record by its unique ID. |
| `POST /{name}:create` | `POST` | Insert a new record (validated against the cache). |
| `POST /{name}:update` | `POST` | Update an existing record. |
| `POST /{name}:destroy` | `POST` | Delete a record from the table. |

---

## 3. Architecture: The Dynamic Data Flow

The server acts as a "Smart Bridge" between the user and the database.

1. **Ingress:** The Go router (e.g., Gin) parses the path `/:name:action`.
2. **Validation:** The server checks the **In-Memory Registry**. If the collection exists in RAM, it validates the JSON body fields against the cached column types.
3. **SQL Generation:** A dynamic query builder generates a sanitized, parameterized SQL statement.
4. **Persistence:** The query is executed against the database (PostgreSQL, MySQL, or SQLite).
5. **Reactive Cache:** If the action was a schema change (e.g., `:update` on collections), the server immediately refreshes that specific map entry in the registry.


## Interface & Integration Layer

* **Dynamic OpenAPI:** The Swagger/OpenAPI documentation is generated dynamically from the **In-Memory Cache**, ensuring the UI always reflects the current DB state.
* **Middleware Security:** A high-speed JWT and API Key layer that checks permissions before the request reaches the dynamic handlers.
* **Webhook Listener:** A concurrent worker pool to ingest third-party data and map it to dynamic collections.

---

## 4. Design for AI Maintainability

* **Predictable Interface:** By standardizing the `:action` suffix, AI agents can guess the correct endpoint for any new collection with 100% accuracy.
* **Statically Typed Logic:** Although data is dynamic (`map[string]any`), the internal validation logic is strictly typed, preventing AI-generated bugs from corrupting the database.
* **TDD Foundation:** Every action is covered by integration tests that mock the database. This allows AI agents to refactor the SQL builder while ensuring no breaking changes occur in the API contract.

---

## 5. Persistence Layer & Agnosticism

* **Dialect-Agnostic:** The server uses a driver-based approach. The user provides a connection string, and Moon-Go detects if it needs to use `Postgres`, `MySQL`, or `SQLite` syntax.
* **Single-Tenant Focus:** Optimized as a high-speed core for a single application, ensuring maximum simplicity and maintainability.


