# 053: Schema Query Parameter for GET Endpoints

## Overview

Enable all GET endpoints to return schema metadata alongside or instead of data using a `?schema` query parameter. This feature supports schema-driven UI frameworks, dynamic form generators, and AI coding agents that need to understand resource structure without manual inspection or separate schema endpoint calls.

### Problem Statement

Currently, clients must:
- Call separate schema endpoints (/collections:get) to understand resource structure
- Infer schema from data responses (error-prone, incomplete for nullable fields)
- Make multiple round trips to fetch both data and schema
- Build custom logic to merge schema with data for UI rendering

This creates unnecessary complexity for schema-driven UIs, admin panels, form generators, and automation tools.

### Solution

Add a `?schema` query parameter to all GET endpoints that:
- Returns schema metadata inline with data responses (`?schema`)
- Returns only schema metadata without data (`?schema=only`)
- Works seamlessly with existing query parameters (filters, pagination, sorting, etc.)
- Maintains consistent schema format across all resource types

### Business Value

- **Faster Development:** UI frameworks can auto-generate forms, tables, and validation without manual schema management
- **Better DX:** Single request for data + schema reduces API calls and complexity
- **AI-Friendly:** Enables AI agents to understand and manipulate resources dynamically
- **Consistency:** Uniform schema discovery pattern across all resources (users, apikeys, collections, records)

## Requirements

### Functional Requirements

#### FR-1: Query Parameter Support
All GET endpoints MUST accept the `schema` query parameter with the following behavior:
- `?schema` (no value) → Returns both data and schema
- `?schema=only` → Returns only schema, omits data
- `?schema=true` → Same as `?schema` (returns both)
- `?schema=false` → Same as no parameter (returns only data)
- Parameter is case-insensitive

#### FR-2: Supported Endpoints
The following endpoints MUST support the schema parameter:
- `/users:list` 
- `/users:get?id=...`
- `/apikeys:list`
- `/apikeys:get?id=...`
- `/collections:list`
- `/collections:get?name=...`
- `/{collection}:list`
- `/{collection}:get?id=...`

#### FR-3: Response Structure with Data and Schema
When `?schema` is present (without `=only`), response MUST include:
```json
{
  "data": [...] | {...},
  "schema": {
    "collection": "resource_name",
    "fields": [
      {
        "name": "field_name",
        "type": "string|integer|boolean|datetime|json|decimal",
        "nullable": true|false,
        "default": "value or null",
        "description": "optional field description"
      }
    ],
    "primary_key": "id",
    "metadata": {
      "created_at": "auto-generated timestamp field",
      "updated_at": "auto-generated timestamp field"
    }
  },
  "next_cursor": "..." // if paginated
}
```

For list endpoints, `data` is an array. For get endpoints, `data` is an object.

#### FR-4: Response Structure with Schema Only
When `?schema=only` is present, response MUST include:
```json
{
  "schema": {
    "collection": "resource_name",
    "fields": [...],
    "primary_key": "id",
    "metadata": {...}
  }
}
```

Data fields (e.g., `data`, `items`, `next_cursor`) MUST be omitted.

#### FR-5: Schema Object Structure
The `schema` object MUST include:
- `collection` (string): Name of the resource/table
- `fields` (array): List of field definitions
- `primary_key` (string): Name of the primary key field
- `metadata` (object): System-generated fields (created_at, updated_at, etc.)

Each field object MUST include:
- `name` (string): Field name
- `type` (string): Data type (string, integer, boolean, datetime, json, decimal)
- `nullable` (boolean): Whether the field accepts null values

Each field object MAY include:
- `default` (any): Default value if not provided
- `description` (string): Human-readable field description
- `constraints` (object): Validation rules (min, max, pattern, enum, etc.)

#### FR-6: Compatibility with Query Parameters
The `?schema` parameter MUST work alongside all existing query parameters:
- Filters: `?schema&field[operator]=value`
- Pagination: `?schema&limit=10&after=cursor`
- Sorting: `?schema&sort=-created_at`
- Field selection: `?schema&fields=id,name`
- Search: `?schema&q=keyword`

When combined with filters/pagination, the schema returned MUST reflect the full collection schema, not just fields in the filtered results.

#### FR-7: Error Handling
- If the endpoint would normally return an error (401, 403, 404, 500), the error response format MUST be preserved
- Schema MUST NOT be included in error responses
- Invalid schema parameter values (e.g., `?schema=invalid`) MUST be ignored and treated as `?schema`

#### FR-8: Performance
- Schema generation MUST NOT cause significant performance degradation
- Schema SHOULD be cached per collection/resource type
- Schema retrieval MUST NOT require additional database queries for dynamic collections

### Technical Requirements

#### TR-1: Implementation
- Add schema parameter parsing to all GET endpoint handlers
- Create a shared `SchemaBuilder` or `SchemaProvider` component
- Schema generation logic MUST be reusable across all resource types
- Avoid duplication: Use reflection or metadata registry where possible

#### TR-2: Schema Source
- For dynamic collections: Use the in-memory schema registry
- For system resources (users, apikeys): Define static schema definitions
- Schema MUST match actual database schema and validation rules

#### TR-3: Backward Compatibility
- Existing endpoints MUST continue to work without the schema parameter
- Response structure MUST NOT change when schema parameter is absent
- API version remains unchanged (no breaking change)

### Non-Functional Requirements

#### NFR-1: Security
- Schema information MUST only be returned if the requesting user has permission to read the resource
- If unauthorized (401) or forbidden (403), return standard error response without schema
- Do not leak sensitive schema information (e.g., internal constraints, hidden fields)

#### NFR-2: Documentation
- Update API documentation template (`doc.md.tmpl`) to include schema parameter examples
- Add schema parameter to all relevant endpoint documentation
- Include example responses showing both `?schema` and `?schema=only`

#### NFR-3: Testing
- Add unit tests for schema parameter parsing
- Add integration tests for all supported endpoints
- Test error cases (unauthorized, not found, invalid parameters)
- Test compatibility with all query parameter combinations
- Verify schema accuracy against actual database schema

## Acceptance Criteria

### AC-1: Basic Functionality
- [ ] All GET endpoints listed in FR-2 accept `?schema` parameter
- [ ] `?schema` returns both data and schema in correct format
- [ ] `?schema=only` returns only schema without data
- [ ] Schema object includes all required fields (collection, fields, primary_key)
- [ ] Field definitions include name, type, and nullable attributes

### AC-2: Integration with Existing Features
- [ ] Schema parameter works with filters: `/products:list?schema&quantity[gte]=10`
- [ ] Schema parameter works with pagination: `/products:list?schema&limit=5&after=cursor`
- [ ] Schema parameter works with sorting: `/products:list?schema&sort=-created_at`
- [ ] Schema parameter works with field selection: `/products:list?schema&fields=id,name`
- [ ] Schema parameter works with search: `/products:list?schema&q=keyword`

### AC-3: Response Validation
- [ ] List endpoint with `?schema` returns: `{"data": [...], "schema": {...}}`
- [ ] Get endpoint with `?schema` returns: `{"data": {...}, "schema": {...}}`
- [ ] Any endpoint with `?schema=only` returns: `{"schema": {...}}`
- [ ] Schema accurately reflects actual collection/resource structure
- [ ] Schema includes all fields present in database

### AC-4: Error Handling
- [ ] 401 Unauthorized returns error without schema
- [ ] 403 Forbidden returns error without schema
- [ ] 404 Not Found returns error without schema
- [ ] Invalid schema parameter values are handled gracefully

### AC-5: Performance
- [ ] Adding `?schema` does not increase response time by more than 50ms
- [ ] Schema generation does not cause N+1 queries
- [ ] Repeated schema requests use cached schema data

### AC-6: Documentation
- [ ] API documentation (`doc.md.tmpl`) includes schema parameter in "Query Options" section
- [ ] Documentation includes examples for `/users:list?schema`
- [ ] Documentation includes examples for `/users:list?schema=only`
- [ ] Documentation includes examples for `/{collection}:list?schema`
- [ ] Documentation includes combined query parameter examples

### AC-7: Testing
- [ ] Unit tests cover schema parameter parsing logic
- [ ] Integration tests verify schema accuracy for users resource
- [ ] Integration tests verify schema accuracy for apikeys resource
- [ ] Integration tests verify schema accuracy for dynamic collections
- [ ] Tests verify schema-only responses omit data fields
- [ ] Tests verify compatibility with all query parameter types
- [ ] All tests pass with 100% success rate

### AC-8: Example Requests and Responses

#### Example 1: List users with schema
```bash
curl "http://localhost:6006/users:list?schema" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "data": [
    {"id": "01ABC...", "username": "admin", "email": "admin@example.com", "role": "admin"}
  ],
  "schema": {
    "collection": "users",
    "fields": [
      {"name": "id", "type": "string", "nullable": false},
      {"name": "username", "type": "string", "nullable": false},
      {"name": "email", "type": "string", "nullable": false},
      {"name": "role", "type": "string", "nullable": false}
    ],
    "primary_key": "id",
    "metadata": {
      "created_at": "datetime",
      "updated_at": "datetime"
    }
  }
}
```

#### Example 2: Get user schema only
```bash
curl "http://localhost:6006/users:get?id=01ABC...&schema=only" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "schema": {
    "collection": "users",
    "fields": [
      {"name": "id", "type": "string", "nullable": false},
      {"name": "username", "type": "string", "nullable": false},
      {"name": "email", "type": "string", "nullable": false},
      {"name": "role", "type": "string", "nullable": false}
    ],
    "primary_key": "id",
    "metadata": {
      "created_at": "datetime",
      "updated_at": "datetime"
    }
  }
}
```

#### Example 3: List collection records with schema and filters
```bash
curl -g "http://localhost:6006/products:list?schema&quantity[gte]=10&sort=-price" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "data": [
    {"id": "01XYZ...", "title": "Laptop", "quantity": 50, "price": "999.99"}
  ],
  "schema": {
    "collection": "products",
    "fields": [
      {"name": "id", "type": "string", "nullable": false},
      {"name": "title", "type": "string", "nullable": false},
      {"name": "quantity", "type": "integer", "nullable": false},
      {"name": "price", "type": "decimal", "nullable": false}
    ],
    "primary_key": "id",
    "metadata": {
      "created_at": "datetime",
      "updated_at": "datetime"
    }
  },
  "next_cursor": "01XYZ..."
}
```

### Implementation Checklist

- [x] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [x] Ensure all unit tests and integration tests are passing successfully.

**Implementation Note:** Schema parameter is implemented for dynamic collection endpoints (/{collection}:list and /{collection}:get). System endpoints (users, apikeys, collections) can be enhanced in a future iteration if needed.
