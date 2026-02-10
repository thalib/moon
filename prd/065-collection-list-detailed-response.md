## Overview

- Enhance the `GET /collections:list` endpoint to return detailed metadata for each collection, including the number of records and fields, to provide clients with a comprehensive overview without requiring additional API calls.
- Currently, the collections list endpoint returns minimal information (likely just collection names), requiring clients to make separate requests to retrieve metadata for each collection.
- The enhanced response enables dashboards, admin interfaces, and monitoring tools to display collection statistics efficiently in a single request.

**Problem Statement:**
- Clients need to display collection statistics (record counts, field counts) but must make multiple API calls to gather this information.
- The current endpoint does not provide sufficient metadata for building informative user interfaces or monitoring dashboards.

**Solution:**
- Extend the response structure to include `records_count` and `field_count` for each collection.
- Add a top-level `count` field indicating the total number of collections.
- Maintain backward compatibility by enriching the existing response structure rather than changing its fundamental shape.

## Requirements

### API Endpoint Specification

**Endpoint:**
- `GET /collections:list`

**Authentication:**
- Requires valid authentication (JWT or API Key).
- Requires read permission on collections metadata.

**Request:**
- No request body required.
- No query parameters required (maintain existing behavior).

**Response Structure:**

**Success Response (HTTP 200):**
```json
{
  "collections": [
    {
      "name": "users",
      "records_count": 150,
      "field_count": 8
    },
    {
      "name": "products",
      "records_count": 2340,
      "field_count": 12
    },
    {
      "name": "orders",
      "records_count": 5672,
      "field_count": 15
    }
  ],
  "count": 3
}
```

**Response Fields:**
- `collections` (array of objects, required): List of collection metadata objects.
  - `name` (string, required): Collection name.
  - `records_count` (integer, required): Total number of records in the collection. Must be >= 0.
  - `field_count` (integer, required): Total number of fields defined in the collection schema. Must be > 0 (every collection has at least one field).
- `count` (integer, required): Total number of collections. Must match the length of the `collections` array.

### Functional Requirements

**Data Retrieval:**
- Retrieve all collection names from the schema registry.
- For each collection, query the database to count the total number of records.
- For each collection, retrieve the schema and count the number of defined fields.
- Return the aggregated data in a single response.

**Record Count Calculation:**
- Execute a `SELECT COUNT(*) FROM {collection}` query for each collection.
- Include soft-deleted records if the collection supports soft deletes (count all records regardless of `deleted_at` status).
- **Assumption:** Record count includes all records, not just active records. If soft-delete filtering is required, mark as **Needs Clarification**.

**Field Count Calculation:**
- Retrieve the collection schema from the in-memory schema registry.
- Count the number of fields in the schema definition.
- Include all field types (string, integer, boolean, etc.).
- Do not include system-generated fields (e.g., `id`, `created_at`, `updated_at`) unless explicitly defined in the schema.
- **Assumption:** Field count reflects user-defined fields in the schema. If system fields should be included, mark as **Needs Clarification**.

**Sorting:**
- Collections should be returned in alphabetical order by name (ascending).
- **Assumption:** Alphabetical sorting by name. If a different sort order is required, mark as **Needs Clarification**.

**Empty Collections:**
- Collections with zero records should return `records_count: 0`.
- All collections must have at least one field; if a collection has no fields, it is invalid and should not be listed.

**Performance Considerations:**
- The endpoint may become slow with many collections or large datasets.
- **Assumption:** This endpoint is intended for admin/monitoring use cases, not high-frequency production queries.
- If the number of collections is large (>100) or record counts are expensive to compute, consider implementing caching or async refresh.
- **Recommendation:** Log a warning if the request takes longer than 2 seconds; document performance characteristics in API docs.

### Validation Rules

**Authentication and Authorization:**
- Return HTTP 401 if the request lacks valid authentication.
- Return HTTP 403 if the authenticated user lacks permission to list collections.

**Input Validation:**
- No input parameters to validate.

**Schema Validation:**
- If the schema registry is empty (no collections defined), return HTTP 200 with `{"collections": [], "count": 0}`.

### Error Handling

**Error Responses:**

**HTTP 401 Unauthorized:**
```json
{
  "error": "Authentication required"
}
```

**HTTP 403 Forbidden:**
```json
{
  "error": "Insufficient permissions to list collections"
}
```

**HTTP 500 Internal Server Error:**
- If database query fails for any collection, return HTTP 500.
- Include details about the failure in logs but not in the response body.
```json
{
  "error": "Failed to retrieve collection metadata"
}
```

**Partial Failure Handling:**
- If the record count query fails for a specific collection, decide on one of the following strategies:
  - **Option A (Recommended):** Return HTTP 500 and log the error.
  - **Option B:** Return the collection with `records_count: -1` or `records_count: null` to indicate unavailable data.
- **Needs Clarification:** Confirm desired behavior for partial failures.

### Performance and Scalability

**Query Optimization:**
- Execute `COUNT(*)` queries in parallel for all collections to minimize latency.
- Use database connection pooling to avoid connection exhaustion.
- Cache the result for a configurable duration (e.g., 60 seconds) if performance is a concern.
  - **Assumption:** Caching is optional and should be configurable. If mandatory, mark as **Needs Clarification**.

**Limits:**
- No pagination is required; return all collections in a single response.
- If the number of collections exceeds a reasonable limit (e.g., 1000), document the performance implications.

### Security and Permissions

**Authentication:**
- Use existing authentication mechanisms (JWT, API Key).

**Authorization:**
- Require `collections:read` or equivalent permission.
- Return HTTP 403 if permission is missing.

**Data Privacy:**
- Collection names are considered non-sensitive metadata and can be returned without redaction.
- Record counts and field counts are aggregated statistics and do not expose sensitive data.

### Observability and Metrics

**Logging:**
- Log each request at INFO level with duration.
- Example log entry:
  ```json
  {
    "level": "info",
    "timestamp": "2026-02-10T11:15:00Z",
    "request_id": "req-456",
    "operation": "collections_list",
    "collections_count": 25,
    "duration_ms": 150
  }
  ```

**Metrics:**
- `collections_list.count`: Total number of requests.
- `collections_list.latency`: Histogram of request latency.
- `collections_list.errors.count`: Total number of failed requests.

### Backward Compatibility

**Existing Behavior:**
- If the current endpoint returns a different structure, ensure the new structure is backward-compatible or increment the API version.
- **Assumption:** The current endpoint is being enhanced, not replaced. If breaking changes are required, mark as **Needs Clarification**.

### Documentation Requirements

**API Documentation Updates:**
- Update the `GET /collections:list` endpoint documentation with:
  - Detailed response structure and field descriptions.
  - Example CURL request and response.
  - Error response examples (401, 403, 500).
  - Performance characteristics and recommended use cases.

**CURL Example:**
```bash
curl -s -X GET "http://localhost:6006/collections:list" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Example Response:**
```json
{
  "collections": [
    {
      "name": "orders",
      "records_count": 5672,
      "field_count": 15
    },
    {
      "name": "products",
      "records_count": 2340,
      "field_count": 12
    },
    {
      "name": "users",
      "records_count": 150,
      "field_count": 8
    }
  ],
  "count": 3
}
```

## Acceptance

### Functional Acceptance Criteria

- [ ] The `GET /collections:list` endpoint returns a JSON response with a `collections` array and a `count` field.
- [ ] Each item in the `collections` array contains `name`, `records_count`, and `field_count` fields.
- [ ] The `records_count` field reflects the total number of records in each collection (including soft-deleted records if applicable).
- [ ] The `field_count` field reflects the total number of fields defined in each collection's schema.
- [ ] The `count` field matches the length of the `collections` array.
- [ ] Collections are returned in alphabetical order by name (ascending).
- [ ] Collections with zero records return `records_count: 0`.
- [ ] If no collections exist, the response is `{"collections": [], "count": 0}` with HTTP 200.

### Authentication and Authorization Acceptance Criteria

- [ ] Requests without valid authentication return HTTP 401 with an appropriate error message.
- [ ] Requests from users without `collections:read` permission return HTTP 403 with an appropriate error message.

### Error Handling Acceptance Criteria

- [ ] If a database query fails, the endpoint returns HTTP 500 with an error message.
- [ ] Partial failures (e.g., one collection's count query fails) are handled according to the clarified strategy (fail-fast or graceful degradation).
- [ ] All errors are logged with sufficient detail for troubleshooting.

### Performance Acceptance Criteria

- [ ] The endpoint completes within 2 seconds for up to 100 collections on a typical database setup.
- [ ] If latency exceeds 2 seconds, a warning is logged.
- [ ] Record count queries are executed in parallel to minimize total request duration.

### Response Validation Acceptance Criteria

- [ ] The response structure matches the documented JSON schema exactly.
- [ ] All required fields (`name`, `records_count`, `field_count`, `count`) are present in the response.
- [ ] `records_count` and `field_count` are non-negative integers.
- [ ] `count` is a non-negative integer equal to the length of the `collections` array.

### Observability Acceptance Criteria

- [ ] Each request is logged at INFO level with operation name, collections count, and duration.
- [ ] Metrics are emitted for request count, latency, and error count.
- [ ] All logs include a `request_id` for correlation.

### Documentation Acceptance Criteria

- [ ] API documentation includes the updated response structure with field descriptions.
- [ ] CURL examples are provided for successful and error scenarios.
- [ ] Performance characteristics and recommended use cases are documented.

### Testing Acceptance Criteria

- [ ] Unit tests verify the response structure and field values for various scenarios (0 collections, 1 collection, multiple collections).
- [ ] Unit tests verify authentication and authorization enforcement (401, 403).
- [ ] Unit tests verify error handling for database query failures (500).
- [ ] Integration tests verify accurate record counts and field counts for test collections.
- [ ] Integration tests verify alphabetical sorting of collections.
- [ ] Performance tests measure latency for 10, 50, and 100 collections.

### Edge Cases and Negative Paths

- [ ] Collections with zero records return `records_count: 0` without error.
- [ ] Newly created collections (no records yet) return `records_count: 0`.
- [ ] Schema registry is empty (no collections): return `{"collections": [], "count": 0}`.
- [ ] A collection is deleted during request processing: handle gracefully (either include or exclude based on transaction isolation).

### Implementation Checklist

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Update API doc template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect these changes.
- [ ] Run all tests and ensure 100% pass rate.
- [ ] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.
