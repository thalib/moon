## Overview
- Implement CRUD data access endpoints for dynamic collections
- Validate requests against In-Memory Schema Registry before database operations
- Support any collection created via schema management API

## Requirements
- `GET /{name}:list` - Fetch all records from specified table
- `GET /{name}:get` - Fetch single record by unique ID
- `POST /{name}:create` - Insert new record (validated against cache)
- `POST /{name}:update` - Update existing record
- `POST /{name}:destroy` - Delete record from table
- Validate collection exists in registry before processing
- Validate JSON body fields match cached column types
- Generate parameterized DML statements (SELECT, INSERT, UPDATE, DELETE)
- Support query parameters for filtering on `:list` endpoint
- Support pagination on `:list` endpoint (limit, offset)
- Return proper HTTP status codes (200, 201, 400, 404, 500)

## Acceptance
- All five CRUD endpoints work for any dynamically created collection
- Request validation rejects invalid field names and types
- SQL injection is prevented via parameterized queries
- Unit tests cover validation logic and SQL generation
- Integration tests verify CRUD operations end-to-end
