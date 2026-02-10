## Overview

- Add first-class batch operations for Create, Update, and Destroy on dynamic collections to enable clients to perform multiple writes in a single request while preserving backward compatibility with existing single-object requests.
- Batch operations reduce network overhead, improve throughput, and enable atomic or partial-success semantics configurable per request.
- Endpoints reuse existing custom action URIs: `POST /{collection}:create`, `POST /{collection}:update`, `POST /{collection}:destroy`.
- Payload format is auto-detected: single object (existing behavior) or array of items (new batch mode).
- Default behavior is atomic (all-or-nothing); optional `atomic=false` query parameter enables partial-success mode with per-item results.

**In-Scope:**

- Batch Create, Batch Update, Batch Destroy on a single collection per request.
- Validation, transaction semantics, response contracts, configurable limits, metrics, documentation, and comprehensive testing.
- Support for atomic (all-or-nothing) and best-effort (partial-success) modes.
- Idempotency support via optional `Idempotency-Key` header or per-item `client_id`.

**Out-of-Scope:**

- Cross-collection transactions.
- Advanced conflict-resolution strategies beyond unique constraint handling.
- Graph/relationship cascading across collections.
- Distributed multi-service two-phase commit.

## Requirements

### API Endpoints and Request Formats

**Endpoints (reuse existing):**

- `POST /{collection}:create` — create one or many records.
- `POST /{collection}:update` — update one or many records.
- `POST /{collection}:destroy` — delete one or many records.

**Request Payload Detection:**

- Single-object mode (existing): `request.data` is an object.
  - Example: `{"data": {"name": "Alice", "email": "alice@example.com"}}`
- Batch mode (new): `request.data` is an array of objects.
  - Create: `{"data": [{"name": "Alice"}, {"name": "Bob"}]}`
  - Update: `{"data": [{"id": "01J...", "name": "Alice"}, {"id": "01J...", "email": "bob@example.com"}]}`
  - Destroy: `{"data": ["01J...", "01J...", "01J..."]}` (array of IDs)

**Query Parameter:**

- `atomic` (boolean, default: `true`)
  - `atomic=true`: All operations succeed or all fail; rollback on any error.
  - `atomic=false`: Best-effort; successful operations are committed, failures are reported per-item.

### Validation Rules

**Pre-write Validation (`atomic=true`):**

- Validate every item in the batch against the collection schema before performing any writes.
- If any item fails validation, reject the entire request with HTTP 400.
- Return structured per-item validation errors with index, field, and error message.

**Best-effort Validation (`atomic=false`):**

- Validate each item individually.
- Proceed with writing valid items; collect failures.
- Return per-item results indicating success or failure with error details.

**Update-Specific Validation:**

- Each update item must include an `id` field.
- Only provided fields are updated (partial update).
- Validate field names and types against collection schema.
- Reject unknown fields or type mismatches.

**Destroy-Specific Validation:**

- Accept an array of IDs.
- Validate ID format (ULID).
- Non-existent IDs are treated as `not_found` per-item; atomic flag controls whether this fails the entire batch.

### Transaction and Database Strategy

**Atomic Mode (`atomic=true`):**

- Wrap the entire batch operation in a database transaction.
- Rollback on any validation error, constraint violation, or database failure.
- Use multi-row/parameterized insert/update/delete statements when supported by the database dialect (Postgres, SQLite).
- If multi-row statements are not supported, execute per-row operations inside the transaction.

**Best-effort Mode (`atomic=false`):**

- Execute operations per-item or in batches.
- Commit successful operations; report failures per-item.
- Use per-row execution with error collection, or execute in transaction and commit partial results.
- Document the chosen strategy clearly in implementation.

**Supported Dialects:**

- Postgres: Use multi-row `INSERT`, `UPDATE`, `DELETE` with RETURNING clause.
- SQLite: Use multi-row statements where supported; fall back to per-row in transaction.
- MySQL: Use multi-row statements where supported.

### Performance and Limits

**Configurable Limits:**

- `max_batch_size` (default: 500 items): Maximum number of items per batch request.
- `max_payload_size` (default: 2 MB): Maximum total payload size.

**Enforcement:**

- Reject requests exceeding `max_batch_size` with HTTP 413 Payload Too Large and message: `"Batch size exceeds limit of {max_batch_size}"`.
- Reject requests exceeding `max_payload_size` with HTTP 413 and message: `"Payload size exceeds limit of {max_payload_size} bytes"`.

**Performance Targets:**

- Bulk create of 100 records: complete within 500ms on CI SQLite database.
- Bulk create of 250 records: complete within 1000ms on CI SQLite database.
- Provide benchmarks for Postgres and MySQL in integration tests.

### Response Formats and HTTP Status Codes

**Single-Object Requests (existing, unchanged):**

- Create: HTTP 201 with created record in `data` field.
- Update: HTTP 200 with updated record in `data` field.
- Destroy: HTTP 200 with success message.

**Batch Requests (`atomic=true`, all succeed):**

- Create: HTTP 201 with array of created records in `data` field and summary message.
  - Example: `{"data": [{...}, {...}], "message": "2 records created successfully"}`
- Update: HTTP 200 with array of updated records in `data` field and summary message.
  - Example: `{"data": [{...}, {...}], "message": "2 records updated successfully"}`
- Destroy: HTTP 200 with summary message.
  - Example: `{"message": "2 records deleted successfully"}`

**Batch Requests (`atomic=false`, partial success):**

- HTTP 207 Multi-Status with per-item results in `results` array.
- Each result includes:
  - `index` (integer): Position in the request array.
  - `id` (string, optional): Record ID if applicable.
  - `status` (string): One of `created`, `updated`, `deleted`, `failed`, `not_found`.
  - `data` (object, optional): The created/updated record if successful.
  - `error_code` (string, optional): Error code if failed.
  - `error_message` (string, optional): Human-readable error message if failed.
- Example:

  ```json
  {
    "results": [
      {"index": 0, "id": "01J...", "status": "created", "data": {...}},
      {"index": 1, "status": "failed", "error_code": "validation_error", "error_message": "Field 'email' is required"}
    ],
    "summary": {
      "total": 2,
      "succeeded": 1,
      "failed": 1
    }
  }
  ```

**Error Responses:**

- HTTP 400 Bad Request: Malformed request or validation error in `atomic=true` mode.
  - Include per-item validation errors with index, field, and message.
- HTTP 401 Unauthorized: Missing or invalid authentication.
- HTTP 403 Forbidden: Insufficient permissions for the operation.
- HTTP 409 Conflict: Unique constraint violation in `atomic=true` mode; rollback entire batch.
  - Include details of the conflicting item(s).
- HTTP 413 Payload Too Large: Batch size or payload size exceeded.
- HTTP 500 Internal Server Error: Database failure or unexpected error in `atomic=true` mode.

### Idempotency and Deduplication

**Idempotency-Key Header (Recommended for Create):**

- Accept optional `Idempotency-Key` header for create operations.
- Scope: per-endpoint (per-collection).
- Behavior: If a request with the same `Idempotency-Key` is received within a configurable window (e.g., 24 hours), return the original response (cached or retrieved from database).
- Implementation: Store key, response, and timestamp; deduplicate on match.

**Client-Supplied ID (Alternative):**

- Allow clients to supply a `client_id` per item in create requests.
- Treat `client_id` as a unique constraint; reject duplicates with HTTP 409 if `atomic=true`.
- In `atomic=false` mode, report duplicate per-item as `failed` with `error_code: "duplicate"`.

### Security and Permissions

**Authentication and Authorization:**

- Reuse existing auth model (JWT and API Key).
- Check collection write permission once at the start of the request.
- Return HTTP 403 if caller lacks required permission; do not proceed with any operations.

**Per-Item Authorization:**

- Ensure per-item operations do not elevate privileges.
- If fine-grained per-record authorization is required, validate per item before write.

**Data Privacy:**

- Do not log sensitive fields (e.g., passwords, tokens) in request or response logs.
- Redact or mask sensitive data in error messages and observability logs.

### Observability and Metrics

**Metrics (per operation type):**

- `batch_create.count`: Total number of batch create requests.
- `batch_create.items.count`: Total number of items in batch create requests.
- `batch_create.success.count`: Total number of successful batch create requests.
- `batch_create.failure.count`: Total number of failed batch create requests.
- `batch_create.items.success.count`: Total number of successfully created items.
- `batch_create.items.failure.count`: Total number of failed items.
- `batch_create.latency`: Histogram of batch create request latency.
- Repeat for `batch_update` and `batch_destroy`.

**Logging:**

- Log summary for each batch request at INFO level:
  - Client ID or IP, collection name, operation type, batch size, atomic flag, success/failure counts, duration.
- Log per-item failures at ERROR level:
  - Index, item ID (if available), error code, error message.
- Include request ID in all logs for correlation.

**Structured Logging Format:**

```json
{
  "level": "info",
  "timestamp": "2026-02-10T10:05:00Z",
  "request_id": "req-123",
  "operation": "batch_create",
  "collection": "users",
  "batch_size": 10,
  "atomic": true,
  "succeeded": 10,
  "failed": 0,
  "duration_ms": 45
}
```

### Testing Requirements

**Unit Tests:**

- Backward compatibility: Verify single-object create/update/destroy requests remain unchanged.
- Batch create success (`atomic=true`): All items created, HTTP 201 returned.
- Batch update success (`atomic=true`): All items updated, HTTP 200 returned.
- Batch destroy success (`atomic=true`): All items deleted, HTTP 200 returned.
- Validation failure in `atomic=true`: No database writes occur; HTTP 400 with per-item errors.
- Partial success in `atomic=false`: Successful items committed, failures reported; HTTP 207 returned.
- Batch size limit enforcement: Reject requests exceeding `max_batch_size` with HTTP 413.
- Payload size limit enforcement: Reject requests exceeding `max_payload_size` with HTTP 413.
- Unique constraint violation in `atomic=true`: Rollback entire batch, HTTP 409 returned.
- Unique constraint violation in `atomic=false`: Report per-item conflict, commit successes.
- Non-existent IDs in destroy: Treated as `not_found` per item; atomic flag controls outcome.

**Integration Tests (per dialect: Postgres, SQLite, MySQL):**

- Validate SQL generation for multi-row insert, update, delete.
- Transaction rollback on database error in `atomic=true` mode.
- Transaction commit of partial results in `atomic=false` mode.
- Verify RETURNING clause behavior (Postgres) or equivalent.

**Performance Tests:**

- Benchmark bulk create with 100, 250, and 500 items.
- Measure latency, memory usage, and database connection usage.
- Run in CI or staging environment with representative database load.

**Test Vectors (canonical JSON examples):**

- Single-object create/update/destroy (existing).
- Batch create success (atomic=true).
- Batch update success (atomic=true).
- Batch destroy success (atomic=true).
- Batch partial success (atomic=false).
- Validation failure (atomic=true).
- Batch size limit exceeded.

### Documentation Requirements

**API Documentation Updates:**

- Describe batch mode for create, update, and destroy endpoints.
- Provide request and response examples for:
  - Single-object requests (existing).
  - Batch requests with `atomic=true` (all succeed).
  - Batch requests with `atomic=false` (partial success, HTTP 207).
  - Validation failure (HTTP 400 with per-item errors).
  - Batch size exceeded (HTTP 413).
- Document `atomic` query parameter (default: true).
- Document `Idempotency-Key` header semantics.
- Document configurable limits: `max_batch_size`, `max_payload_size`.
- Include CURL examples for all scenarios.

**Migration Notes:**

- Emphasize backward compatibility: existing single-object clients are unaffected.
- Recommend gradual migration to batch endpoints for bulk operations.
- Provide guidance on choosing `atomic=true` vs `atomic=false`.

**Configuration Documentation:**

- Document configuration keys for batch limits:
  - `api.batch.max_size` (default: 500)
  - `api.batch.max_payload_bytes` (default: 2097152)
- Document idempotency configuration (if implemented).

### Migration and Rollout Plan

**Phase 1: Implementation**

- Implement batch mode for create, update, and destroy handlers.
- Add validation, transaction management, and response formatting.
- Ensure backward compatibility with single-object requests.

**Phase 2: Testing**

- Execute unit, integration, and performance tests.
- Validate against all supported database dialects.
- Ensure 100% pass rate.

**Phase 3: Staging Rollout**

- Deploy to staging environment.
- Enable batch operations with feature flag or configuration toggle (optional).
- Execute smoke tests with realistic batch payloads.
- Monitor metrics and logs for errors or performance issues.

**Phase 4: Production Rollout**

- Deploy to production with batch operations enabled.
- Monitor error rates, latency, and database resource usage.
- Gradually increase batch size limits if performance allows.

**Rollback Plan:**

- If critical issues occur, disable batch mode via configuration toggle.
- Revert to single-request-only mode by rejecting array payloads with HTTP 400.
- Document rollback steps in operations runbook.

## Acceptance

### Functional Acceptance Criteria

- [ ] Existing single-object create, update, and destroy requests continue to work unchanged; all existing tests pass.
- [ ] Batch create endpoint accepts an array of objects and creates all records when `atomic=true`; returns HTTP 201 with array of created records.
- [ ] Batch update endpoint accepts an array of objects with IDs and updates all records when `atomic=true`; returns HTTP 200 with array of updated records.
- [ ] Batch destroy endpoint accepts an array of IDs and deletes all records when `atomic=true`; returns HTTP 200 with success message.
- [ ] When `atomic=true` and any item fails validation, no database writes occur; HTTP 400 is returned with per-item validation errors.
- [ ] When `atomic=true` and any database constraint is violated, the entire batch is rolled back; HTTP 409 or 500 is returned with error details.
- [ ] When `atomic=false` and some items fail, successful items are committed and HTTP 207 is returned with per-item results indicating success/failure.
- [ ] Requests exceeding `max_batch_size` are rejected with HTTP 413 and appropriate error message.
- [ ] Requests exceeding `max_payload_size` are rejected with HTTP 413 and appropriate error message.
- [ ] Update operations require an `id` per item; missing IDs result in validation error.
- [ ] Destroy operations accept an array of IDs; non-existent IDs are reported as `not_found` per item; atomic flag controls whether this fails the batch.
- [ ] Unique constraint violations are detected and handled per atomic mode (rollback in `atomic=true`, per-item failure in `atomic=false`).

### Transaction and Database Acceptance Criteria

- [ ] Batch operations in `atomic=true` mode are wrapped in a database transaction; rollback occurs on any error.
- [ ] Multi-row insert, update, and delete statements are used when supported by the database dialect (Postgres, SQLite, MySQL).
- [ ] Transaction rollback is verified via integration tests simulating database errors.
- [ ] Partial results in `atomic=false` mode are committed successfully; integration tests validate commit behavior.

### Response Format Acceptance Criteria

- [ ] Batch create success (`atomic=true`) returns HTTP 201 with array of created records and summary message.
- [ ] Batch update success (`atomic=true`) returns HTTP 200 with array of updated records and summary message.
- [ ] Batch destroy success (`atomic=true`) returns HTTP 200 with summary message.
- [ ] Batch partial success (`atomic=false`) returns HTTP 207 with per-item results array including index, status, data (if success), and error details (if failure).
- [ ] Validation errors in `atomic=true` return HTTP 400 with structured per-item errors.

### Performance Acceptance Criteria

- [ ] Bulk create of 100 records completes within 500ms on CI SQLite database.
- [ ] Bulk create of 250 records completes within 1000ms on CI SQLite database.
- [ ] Performance benchmarks for 100, 250, and 500 items are documented and reproducible.

### Security and Permissions Acceptance Criteria

- [ ] Collection write permission is checked at the start of each batch request; HTTP 403 is returned if permission is missing.
- [ ] Sensitive fields are not logged in request or response logs.
- [ ] Error messages do not expose sensitive data.

### Observability Acceptance Criteria

- [ ] Metrics are emitted for each batch operation: count, success count, failure count, item counts, and latency.
- [ ] Summary logs are emitted at INFO level for each batch request with key details (collection, batch size, counts, duration).
- [ ] Per-item failures are logged at ERROR level with index and error details.
- [ ] All logs include request ID for correlation.

### Testing Acceptance Criteria

- [ ] Unit tests cover backward compatibility, batch success, validation failures, partial success, limits enforcement, and conflict handling.
- [ ] Integration tests validate SQL generation and transaction behavior for Postgres, SQLite, and MySQL.
- [ ] Performance tests benchmark bulk create operations and measure latency/resource usage.
- [ ] Test vectors (canonical JSON examples) are provided for all major scenarios.

### Documentation Acceptance Criteria

- [ ] API documentation includes batch mode descriptions, request/response examples, and CURL examples for all scenarios.
- [ ] Migration notes emphasize backward compatibility and provide guidance on adopting batch endpoints.
- [ ] Configuration documentation describes batch limit keys and defaults.

### Rollout Acceptance Criteria

- [ ] Batch operations are deployed to staging and smoke-tested successfully.
- [ ] Batch operations are deployed to production with monitoring enabled.
- [ ] Metrics and logs show <0.1% unexpected failures in production.
- [ ] Rollback plan is documented and tested.

### Post-Launch Success Metrics

- [ ] Correctness: <0.1% unexpected failures for batch requests in production.
- [ ] Performance: Median latency for 100-item batch is within 500ms (or defined SLA).
- [ ] Error visibility: Every batch failure produces actionable logs and metrics.
- [ ] Client adoption: Bulk operations migrate to batch APIs at expected rate (define target with product team).

### Implementation Checklist

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Update API doc template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect these changes.
- [ ] Run all tests and ensure 100% pass rate.
- [ ] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.
- [ ] Ensure all unit tests and integration tests in `tests/*` are passing successfully.
