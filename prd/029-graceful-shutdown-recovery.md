# Graceful Shutdown and Recovery Logic for Moon Backend

## 1. Overview

### Problem Statement
Moon's backend can become inconsistent after abrupt shutdowns or container restarts, especially when using SQLite or MySQL. This leads to situations where the physical database tables and the collections registry are out of sync, causing errors such as failed table creation or missing collections.

### Context and Background
Moon supports multiple database backends (SQLite, MySQL). The current implementation does not guarantee that the collections registry and actual database tables remain consistent after ungraceful shutdowns, container stops, or crashes. This is particularly problematic in Dockerized deployments where the database file is persisted across container restarts.

### High-Level Solution Summary
Implement robust shutdown and recovery logic in the Moon backend to:
- Ensure all in-flight operations are completed and database connections are closed cleanly on shutdown.
- Detect and repair inconsistencies between the collections registry and physical tables on startup.
- Provide health checks and logging for consistency status.
- Support both SQLite and MySQL backends.

## 2. Requirements

### Functional Requirements
- The backend must handle SIGTERM and SIGINT signals, performing a graceful shutdown:
  - Complete all in-flight requests.
  - Commit or roll back all open transactions.
  - Close all database connections.
- On startup, the backend must verify consistency between the collections registry and physical tables:
  - For each registered collection, confirm the corresponding table exists.
  - For each physical table, confirm it is registered (excluding system tables).
- If inconsistencies are detected, the backend must:
  - Log all discrepancies.
  - Optionally repair inconsistencies (configurable):
    - Remove registry entries for missing tables.
    - Register orphaned tables if schema can be inferred.
    - Optionally drop orphaned tables (admin-configurable).
- All create/update/destroy collection operations must be transactional:
  - Registry and table changes must succeed or fail as a unit.
- The health endpoint must report registry/table consistency status.
- All shutdown/startup events, inconsistencies, and repairs must be logged.
- The solution must work for both SQLite and MySQL backends.

### Technical Requirements
- Use Go's context and signal handling for graceful shutdown.
- Use database transactions where supported.
- Provide configuration options for auto-repair and orphan handling.
- Ensure compatibility with Dockerized and host deployments.
- Add automated tests for crash recovery and consistency checks.

### API Specifications
- No new external API endpoints required.
- Health endpoint (`/health`) must include a `consistency` field indicating status and details if inconsistent.
- Configuration options (e.g., `auto_repair`, `drop_orphans`) must be documented in the main config file.

### Validation Rules and Constraints
- Consistency checks must not block startup for more than 5 seconds unless repair is in progress.
- Repairs must be idempotent and safe to run multiple times.
- No data loss unless explicitly configured (e.g., dropping orphaned tables).

### Error Handling and Failure Modes
- If consistency cannot be restored automatically, the backend must start in a degraded mode and log a critical error.
- All errors must be logged with sufficient detail for diagnosis.
- If a transaction fails, all changes must be rolled back.

### Filtering, Sorting, Permissions, and Limits
- Not applicable to this feature.

## 3. Acceptance Criteria

### Verification Steps
- Simulate abrupt shutdowns and verify that the backend can detect and repair inconsistencies on restart.
- Create, update, and destroy collections; verify atomicity and rollback on failure.
- Test both SQLite and MySQL backends for all scenarios.
- Verify that the health endpoint reports consistency status accurately.
- Check logs for all shutdown, startup, and repair events.
- Test configuration options for auto-repair and orphan handling.

### Test Scenarios
- Normal shutdown: All data and registry remain consistent.
- Abrupt shutdown: Backend detects and repairs inconsistencies on next startup.
- Orphaned table: Table exists but not in registry; handled per config.
- Orphaned registry: Registry entry exists but table is missing; handled per config.
- Transaction failure: No partial changes in registry or tables.
- Health endpoint: Reports correct status in all cases.

### Expected API Responses
- Health endpoint returns `{ "status": "ok", "consistency": "ok" }` when consistent.
- Health endpoint returns `{ "status": "ok", "consistency": "inconsistent", "details": [...] }` when issues are found.

### Edge Cases and Negative Paths
- Multiple simultaneous inconsistencies.
- Database connection loss during repair.
- Unsupported or unknown table schemas.
- Manual tampering with database files.
