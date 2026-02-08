# PRD-060: Documented Curl Response Examples

## Overview

**Problem Statement:**
Moon's API documentation at `/doc` endpoint currently shows comprehensive curl request examples but omits response examples. Developers integrating with Moon must execute live API calls to understand response structures, status codes, headers, and error formats. This lack of response documentation:
- Increases integration time and developer friction
- Requires a running Moon instance to understand API behavior
- Makes it difficult to anticipate error handling requirements
- Reduces documentation utility for AI coding agents and automated client generation
- Creates ambiguity about exact response formats, field types, and optional fields

**Context:**
- Moon provides dynamic API documentation via `/doc` endpoint (HTML) and `/doc/llms-full.txt` (Markdown)
- Documentation template is located at `cmd/moon/internal/handlers/templates/doc.md.tmpl`
- Current documentation shows curl request examples for all endpoints but no response examples
- Moon uses standard JSON response format with consistent error structures
- The API returns various HTTP status codes (200, 201, 400, 401, 403, 404, 409, 422, 429, 500)
- Response format includes both success and error cases with specific field structures
- Moon's documentation is used by:
  - Human developers integrating Moon into applications
  - AI coding agents generating client code
  - Automated tools creating API client libraries
  - QA teams writing integration tests

**Solution:**
Enhance the API documentation template to include comprehensive response examples for every endpoint. Each endpoint documentation should show:
- **Success responses** with actual JSON structure, field names, and sample data
- **HTTP status codes** (200 OK, 201 Created, etc.)
- **Response headers** where relevant (rate limiting, pagination, CORS)
- **Common error responses** (401, 403, 404, 422, 500) with error format
- **Response field descriptions** where necessary for clarity
- **Pagination metadata** for list endpoints (next_cursor, has_more)
- **Empty result cases** (empty arrays, null values)

Response examples must:
- Match actual API implementation (verified through testing)
- Use realistic sample data consistent with request examples
- Include all response fields (required and optional)
- Show proper JSON formatting with syntax highlighting
- Include HTTP status line and relevant headers
- Cover both happy path and error scenarios

**Benefits:**
- **Faster Integration:** Developers understand API behavior without trial-and-error
- **Better Testing:** Clear expectations enable writing comprehensive test cases
- **Reduced Support Load:** Self-service documentation answers most questions
- **AI Agent Friendly:** Complete request/response pairs improve automated code generation
- **Error Handling Clarity:** Explicit error examples show how to handle failures
- **Documentation Completeness:** Industry-standard API documentation includes responses
- **Validation:** Response examples serve as reference for implementation correctness
- **Consistency:** Standardized response format documentation ensures API consistency

**Breaking Change:**
This is **NOT a breaking change**. This PRD only enhances documentation; no API behavior is modified.

---

## Requirements

### FR-1: Response Example Format Specification

**FR-1.1: Standard Response Example Structure**
Each endpoint's response example must follow this structure:

```markdown
**Request:**
```bash
curl -X POST "http://localhost:6006/collections:create" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "products",
    "columns": [
      {"name": "title", "type": "string", "nullable": false}
    ]
  }' | jq .
```

**Response (201 Created):**
```json
{
  "message": "Collection created successfully",
  "collection": {
    "name": "products",
    "columns": [
      {"name": "id", "type": "string", "nullable": false},
      {"name": "title", "type": "string", "nullable": false},
      {"name": "created_at", "type": "datetime", "nullable": false},
      {"name": "updated_at", "type": "datetime", "nullable": false}
    ]
  }
}
```

**Response Headers:**
```
HTTP/1.1 201 Created
Content-Type: application/json
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1706875200
```

**Error Response (409 Conflict):**
```json
{
  "error": "Collection already exists: products",
  "code": 409
}
```
```

**FR-1.2: Response Example Requirements**
Each response example must include:
- **HTTP status line:** `HTTP/1.1 <code> <status>` (e.g., `HTTP/1.1 200 OK`)
- **Relevant headers:** Content-Type (always), rate limiting (for authenticated endpoints), CORS (for public endpoints)
- **Complete JSON body:** All fields that would be returned, including optional fields when applicable
- **Realistic data:** Sample values consistent with request examples (e.g., matching IDs, usernames, collection names)
- **Proper formatting:** Valid JSON with 2-space indentation
- **Field type representation:** Strings in quotes, numbers unquoted, booleans as `true`/`false`, null as `null`

**FR-1.3: Multiple Response Scenarios Per Endpoint**
Document at least the following response types per endpoint:
1. **Primary success response** (200 OK or 201 Created)
2. **Most common error response** for that endpoint (e.g., 404 for :get, 409 for :create)
3. **Authentication error** (401 Unauthorized) for protected endpoints
4. **Authorization error** (403 Forbidden) for admin-only endpoints
5. **Validation error** (400 Bad Request or 422 Unprocessable Entity) for endpoints with complex input

**FR-1.4: Error Response Documentation**
All error responses must follow Moon's standard error format:
```json
{
  "error": "Human-readable error message describing what went wrong",
  "code": 400
}
```

Document common errors:
- **400 Bad Request:** Invalid input, malformed JSON, invalid filter operators
- **401 Unauthorized:** Missing or invalid authentication token
- **403 Forbidden:** Valid token but insufficient permissions (e.g., `"Forbidden: admin role required"`)
- **404 Not Found:** Collection or record not found (e.g., `"Collection not found: products"`)
- **409 Conflict:** Resource already exists (e.g., `"Collection already exists: products"`)
- **422 Unprocessable Entity:** Validation failure (e.g., `"Field 'email' is required"`)
- **429 Too Many Requests:** Rate limit exceeded
- **500 Internal Server Error:** Server error (e.g., `"Database connection failed"`)

### FR-2: Authentication Endpoints Response Examples

**FR-2.1: POST /auth:login Response**
Success response (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFKWFI3...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFKWFI3...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "01JXRZ7HDKTSV4RRFFQ69G5FAA",
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "created_at": "2026-01-15T10:30:00Z"
  }
}
```

Error response (401 Unauthorized):
```json
{
  "error": "Invalid username or password",
  "code": 401
}
```

**FR-2.2: POST /auth:logout Response**
Success response (200 OK):
```json
{
  "message": "Logged out successfully"
}
```

**FR-2.3: POST /auth:refresh Response**
Success response (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFKWFI3...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDFKWFI3...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

Error response (401 Unauthorized):
```json
{
  "error": "Invalid or expired refresh token",
  "code": 401
}
```

**FR-2.4: GET /auth:me Response**
Success response (200 OK):
```json
{
  "id": "01JXRZ7HDKTSV4RRFFQ69G5FAA",
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin",
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-02-03T14:20:00Z"
}
```

**FR-2.5: POST /auth:me Response**
Success response (200 OK) for email update:
```json
{
  "message": "User updated successfully",
  "user": {
    "id": "01JXRZ7HDKTSV4RRFFQ69G5FAA",
    "username": "admin",
    "email": "newemail@example.com",
    "role": "admin",
    "created_at": "2026-01-15T10:30:00Z",
    "updated_at": "2026-02-03T14:25:00Z"
  }
}
```

Success response (200 OK) for password change:
```json
{
  "message": "Password updated successfully"
}
```

Error response (401 Unauthorized) for incorrect old password:
```json
{
  "error": "Old password is incorrect",
  "code": 401
}
```

### FR-3: User Management Endpoints Response Examples

**FR-3.1: GET /users:list Response**
Success response (200 OK):
```json
{
  "users": [
    {
      "id": "01JXRZ7HDKTSV4RRFFQ69G5FAA",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z"
    },
    {
      "id": "01JXRZ9ABCDEFGH123456789",
      "username": "johndoe",
      "email": "john@example.com",
      "role": "user",
      "created_at": "2026-02-01T09:15:00Z",
      "updated_at": "2026-02-01T09:15:00Z"
    }
  ],
  "total": 2
}
```

Error response (403 Forbidden) for non-admin user:
```json
{
  "error": "Forbidden: admin role required",
  "code": 403
}
```

**FR-3.2: GET /users:get Response**
Success response (200 OK):
```json
{
  "id": "01JXRZ9ABCDEFGH123456789",
  "username": "johndoe",
  "email": "john@example.com",
  "role": "user",
  "created_at": "2026-02-01T09:15:00Z",
  "updated_at": "2026-02-01T09:15:00Z"
}
```

Error response (404 Not Found):
```json
{
  "error": "User not found",
  "code": 404
}
```

**FR-3.3: POST /users:create Response**
Success response (201 Created):
```json
{
  "message": "User created successfully",
  "user": {
    "id": "01KGF1ABCDEFGH123456789",
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "user",
    "created_at": "2026-02-03T10:30:00Z",
    "updated_at": "2026-02-03T10:30:00Z"
  }
}
```

Error response (409 Conflict):
```json
{
  "error": "Username already exists",
  "code": 409
}
```

Error response (422 Unprocessable Entity):
```json
{
  "error": "Password must be at least 8 characters and contain uppercase, lowercase, number, and special character",
  "code": 422
}
```

**FR-3.4: POST /users:update Response**
Success response (200 OK):
```json
{
  "message": "User updated successfully",
  "user": {
    "id": "01KGF1ABCDEFGH123456789",
    "username": "newuser",
    "email": "updateduser@example.com",
    "role": "admin",
    "created_at": "2026-02-03T10:30:00Z",
    "updated_at": "2026-02-03T14:45:00Z"
  }
}
```

Success response (200 OK) for password reset:
```json
{
  "message": "Password reset successfully"
}
```

Success response (200 OK) for session revocation:
```json
{
  "message": "All user sessions revoked successfully"
}
```

**FR-3.5: POST /users:destroy Response**
Success response (200 OK):
```json
{
  "message": "User deleted successfully"
}
```

### FR-4: API Key Management Endpoints Response Examples

**FR-4.1: GET /apikeys:list Response**
Success response (200 OK):
```json
{
  "apikeys": [
    {
      "id": "01KGF0NGEHYKJR8PV5KHQBDHKB",
      "name": "Integration Service",
      "description": "Key for integration",
      "role": "user",
      "can_write": false,
      "key_prefix": "moon_live_ArPaQD...",
      "created_at": "2026-02-02T11:09:07Z",
      "last_used": "2026-02-03T10:15:00Z"
    },
    {
      "id": "01KGF2XYZABCDEFGHIJKLMNOPQ",
      "name": "Test Service",
      "description": "Key for testing",
      "role": "user",
      "can_write": true,
      "key_prefix": "moon_live_Xyz123...",
      "created_at": "2026-02-03T08:30:00Z",
      "last_used": null
    }
  ],
  "total": 2
}
```

**FR-4.2: GET /apikeys:get Response**
Success response (200 OK):
```json
{
  "id": "01KGF0NGEHYKJR8PV5KHQBDHKB",
  "name": "Integration Service",
  "description": "Key for integration",
  "role": "user",
  "can_write": false,
  "key_prefix": "moon_live_ArPaQD...",
  "created_at": "2026-02-02T11:09:07Z",
  "last_used": "2026-02-03T10:15:00Z"
}
```

Error response (404 Not Found):
```json
{
  "error": "API key not found",
  "code": 404
}
```

**FR-4.3: POST /apikeys:create Response**
Success response (201 Created):
```json
{
  "message": "API key created successfully",
  "warning": "Store this key securely. It will not be shown again.",
  "apikey": {
    "id": "01KGF0NGEHYKJR8PV5KHQBDHKB",
    "name": "Integration Service",
    "description": "Key for integration",
    "role": "user",
    "can_write": false,
    "created_at": "2026-02-02T11:09:07Z"
  },
  "key": "moon_live_ArPaQDdyh4gp9EKF42Xrq7AOAAJZmKiCtTCv1m2rj931QxmPkiXuSV8Un0qllMiN"
}
```

Error response (403 Forbidden):
```json
{
  "error": "Forbidden: admin role required",
  "code": 403
}
```

**FR-4.4: POST /apikeys:update Response**
Success response (200 OK) for metadata update:
```json
{
  "message": "API key updated successfully",
  "apikey": {
    "id": "01KGF0NGEHYKJR8PV5KHQBDHKB",
    "name": "Updated Service Name",
    "description": "Updated description",
    "role": "user",
    "can_write": true,
    "key_prefix": "moon_live_ArPaQD...",
    "created_at": "2026-02-02T11:09:07Z",
    "updated_at": "2026-02-03T15:00:00Z"
  }
}
```

Success response (200 OK) for key rotation:
```json
{
  "message": "API key rotated successfully",
  "warning": "Old key is now invalid. Store this new key securely.",
  "apikey": {
    "id": "01KGF0NGEHYKJR8PV5KHQBDHKB",
    "name": "Integration Service",
    "description": "Key for integration",
    "role": "user",
    "can_write": false,
    "created_at": "2026-02-02T11:09:07Z",
    "updated_at": "2026-02-03T15:05:00Z"
  },
  "key": "moon_live_NewKeyValueHere12345678901234567890123456789012345678901234"
}
```

**FR-4.5: POST /apikeys:destroy Response**
Success response (200 OK):
```json
{
  "message": "API key deleted successfully"
}
```

### FR-5: Collection Management Response Examples

**FR-5.1: GET /collections:list Response**
Success response (200 OK):
```json
{
  "collections": [
    {
      "name": "products",
      "columns": [
        {"name": "id", "type": "string", "nullable": false},
        {"name": "title", "type": "string", "nullable": false},
        {"name": "price", "type": "decimal", "nullable": false},
        {"name": "quantity", "type": "integer", "nullable": false},
        {"name": "brand", "type": "string", "nullable": false},
        {"name": "created_at", "type": "datetime", "nullable": false},
        {"name": "updated_at", "type": "datetime", "nullable": false}
      ]
    },
    {
      "name": "customers",
      "columns": [
        {"name": "id", "type": "string", "nullable": false},
        {"name": "name", "type": "string", "nullable": false},
        {"name": "email", "type": "string", "nullable": false},
        {"name": "created_at", "type": "datetime", "nullable": false},
        {"name": "updated_at", "type": "datetime", "nullable": false}
      ]
    }
  ],
  "total": 2
}
```

Empty response when no collections exist:
```json
{
  "collections": [],
  "total": 0
}
```

**FR-5.2: GET /collections:get Response**
Success response (200 OK):
```json
{
  "name": "products",
  "columns": [
    {"name": "id", "type": "string", "nullable": false},
    {"name": "title", "type": "string", "nullable": false},
    {"name": "price", "type": "decimal", "nullable": false},
    {"name": "quantity", "type": "integer", "nullable": false},
    {"name": "brand", "type": "string", "nullable": false},
    {"name": "created_at", "type": "datetime", "nullable": false},
    {"name": "updated_at", "type": "datetime", "nullable": false}
  ]
}
```

Error response (404 Not Found):
```json
{
  "error": "Collection not found: products",
  "code": 404
}
```

**FR-5.3: POST /collections:create Response**
Success response (201 Created):
```json
{
  "message": "Collection created successfully",
  "collection": {
    "name": "products",
    "columns": [
      {"name": "id", "type": "string", "nullable": false},
      {"name": "title", "type": "string", "nullable": false},
      {"name": "price", "type": "integer", "nullable": false},
      {"name": "description", "type": "string", "nullable": true},
      {"name": "created_at", "type": "datetime", "nullable": false},
      {"name": "updated_at", "type": "datetime", "nullable": false}
    ]
  }
}
```

Error response (409 Conflict):
```json
{
  "error": "Collection already exists: products",
  "code": 409
}
```

Error response (400 Bad Request):
```json
{
  "error": "Invalid collection name: must be lowercase, snake_case, and alphanumeric",
  "code": 400
}
```

**FR-5.4: POST /collections:update Response**
Success response (200 OK) for adding columns:
```json
{
  "message": "Collection updated successfully",
  "collection": {
    "name": "products",
    "columns": [
      {"name": "id", "type": "string", "nullable": false},
      {"name": "title", "type": "string", "nullable": false},
      {"name": "price", "type": "integer", "nullable": false},
      {"name": "description", "type": "string", "nullable": true},
      {"name": "stock", "type": "integer", "nullable": false},
      {"name": "category", "type": "string", "nullable": false},
      {"name": "created_at", "type": "datetime", "nullable": false},
      {"name": "updated_at", "type": "datetime", "nullable": false}
    ]
  }
}
```

Success response (200 OK) for renaming columns:
```json
{
  "message": "Collection updated successfully",
  "collection": {
    "name": "products",
    "columns": [
      {"name": "id", "type": "string", "nullable": false},
      {"name": "title", "type": "string", "nullable": false},
      {"name": "price", "type": "integer", "nullable": false},
      {"name": "details", "type": "string", "nullable": true},
      {"name": "created_at", "type": "datetime", "nullable": false},
      {"name": "updated_at", "type": "datetime", "nullable": false}
    ]
  }
}
```

Error response (400 Bad Request):
```json
{
  "error": "Column 'invalid_column' does not exist",
  "code": 400
}
```

**FR-5.5: POST /collections:destroy Response**
Success response (200 OK):
```json
{
  "message": "Collection deleted successfully"
}
```

Error response (404 Not Found):
```json
{
  "error": "Collection not found: products",
  "code": 404
}
```

### FR-6: Data Access Endpoints Response Examples

**FR-6.1: GET /{collection}:list Response**
Success response (200 OK):
```json
{
  "data": [
    {
      "id": "01KGD5E74RYS0WTNARQC92S1P7",
      "title": "Wireless Mouse",
      "price": "29.99",
      "details": "Ergonomic wireless mouse",
      "quantity": 10,
      "brand": "Wow",
      "created_at": "2026-02-03T10:45:00Z",
      "updated_at": "2026-02-03T10:45:00Z"
    },
    {
      "id": "01KGD5FGHIJK123456789ABCDE",
      "title": "USB Keyboard",
      "price": "49.99",
      "details": "Mechanical keyboard",
      "quantity": 5,
      "brand": "Wow",
      "created_at": "2026-02-03T11:00:00Z",
      "updated_at": "2026-02-03T11:00:00Z"
    }
  ],
  "total": 2
}
```

Success response (200 OK) with pagination:
```json
{
  "data": [
    {
      "id": "01KGD5E74RYS0WTNARQC92S1P7",
      "title": "Wireless Mouse",
      "price": "29.99",
      "quantity": 10,
      "brand": "Wow"
    }
  ],
  "total": 1,
  "next_cursor": "01KGD5FGHIJK123456789ABCDE",
  "has_more": true
}
```

Empty response (200 OK):
```json
{
  "data": [],
  "total": 0
}
```

Error response (404 Not Found):
```json
{
  "error": "Collection not found: products",
  "code": 404
}
```

**FR-6.2: GET /{collection}:schema Response**
Success response (200 OK):
```json
{
  "collection": "products",
  "fields": [
    {"name": "id", "type": "string", "nullable": false},
    {"name": "title", "type": "string", "nullable": false},
    {"name": "price", "type": "integer", "nullable": false},
    {"name": "description", "type": "string", "nullable": true},
    {"name": "created_at", "type": "datetime", "nullable": false},
    {"name": "updated_at", "type": "datetime", "nullable": false}
  ]
}
```

Error response (404 Not Found):
```json
{
  "error": "Collection not found: products",
  "code": 404
}
```

**FR-6.3: GET /{collection}:get Response**
Success response (200 OK):
```json
{
  "id": "01KGD5E74RYS0WTNARQC92S1P7",
  "title": "Wireless Mouse",
  "price": "29.99",
  "details": "Ergonomic wireless mouse",
  "quantity": 10,
  "brand": "Wow",
  "created_at": "2026-02-03T10:45:00Z",
  "updated_at": "2026-02-03T10:45:00Z"
}
```

Error response (404 Not Found):
```json
{
  "error": "Record not found",
  "code": 404
}
```

**FR-6.4: POST /{collection}:create Response**
Success response (201 Created):
```json
{
  "message": "Record created successfully",
  "data": {
    "id": "01KGD5E74RYS0WTNARQC92S1P7",
    "title": "Wireless Mouse",
    "price": "29.99",
    "details": "Ergonomic wireless mouse",
    "quantity": 10,
    "brand": "Wow",
    "created_at": "2026-02-03T10:45:00Z",
    "updated_at": "2026-02-03T10:45:00Z"
  }
}
```

Error response (422 Unprocessable Entity):
```json
{
  "error": "Field 'title' is required and cannot be null",
  "code": 422
}
```

Error response (400 Bad Request):
```json
{
  "error": "Invalid data type for field 'price': expected integer, got string",
  "code": 400
}
```

**FR-6.5: POST /{collection}:update Response**
Success response (200 OK):
```json
{
  "message": "Record updated successfully",
  "data": {
    "id": "01KGD5E74RYS0WTNARQC92S1P7",
    "title": "Wireless Mouse",
    "price": "39.99",
    "details": "Ergonomic wireless mouse",
    "quantity": 10,
    "brand": "Wow",
    "created_at": "2026-02-03T10:45:00Z",
    "updated_at": "2026-02-03T14:30:00Z"
  }
}
```

Error response (404 Not Found):
```json
{
  "error": "Record not found",
  "code": 404
}
```

**FR-6.6: POST /{collection}:destroy Response**
Success response (200 OK):
```json
{
  "message": "Record deleted successfully"
}
```

Error response (404 Not Found):
```json
{
  "error": "Record not found",
  "code": 404
}
```

### FR-7: Query Options Response Examples

**FR-7.1: Filtering Response**
Success response (200 OK) for `?quantity[gt]=5&brand[eq]=Wow`:
```json
{
  "data": [
    {
      "id": "01KGD5E74RYS0WTNARQC92S1P7",
      "title": "Wireless Mouse",
      "price": "29.99",
      "quantity": 10,
      "brand": "Wow",
      "created_at": "2026-02-03T10:45:00Z",
      "updated_at": "2026-02-03T10:45:00Z"
    }
  ],
  "total": 1
}
```

Error response (400 Bad Request) for invalid operator:
```json
{
  "error": "Invalid filter operator: invalid. Supported operators: eq, ne, gt, lt, gte, lte, like, in",
  "code": 400
}
```

**FR-7.2: Sorting Response**
Success response (200 OK) for `?sort=-quantity,title`:
```json
{
  "data": [
    {
      "id": "01KGD5E74RYS0WTNARQC92S1P7",
      "title": "Wireless Mouse",
      "quantity": 10
    },
    {
      "id": "01KGD5FGHIJK123456789ABCDE",
      "title": "USB Keyboard",
      "quantity": 5
    }
  ],
  "total": 2
}
```

**FR-7.3: Full-Text Search Response**
Success response (200 OK) for `?q=mouse`:
```json
{
  "data": [
    {
      "id": "01KGD5E74RYS0WTNARQC92S1P7",
      "title": "Wireless Mouse",
      "details": "Ergonomic wireless mouse",
      "price": "29.99",
      "quantity": 10,
      "brand": "Wow"
    }
  ],
  "total": 1
}
```

**FR-7.4: Field Selection Response**
Success response (200 OK) for `?fields=title,price`:
```json
{
  "data": [
    {
      "id": "01KGD5E74RYS0WTNARQC92S1P7",
      "title": "Wireless Mouse",
      "price": "29.99"
    },
    {
      "id": "01KGD5FGHIJK123456789ABCDE",
      "title": "USB Keyboard",
      "price": "49.99"
    }
  ],
  "total": 2
}
```

**FR-7.5: Pagination Response**
Success response (200 OK) for `?limit=1`:
```json
{
  "data": [
    {
      "id": "01KGD5E74RYS0WTNARQC92S1P7",
      "title": "Wireless Mouse",
      "price": "29.99",
      "quantity": 10,
      "brand": "Wow"
    }
  ],
  "total": 1,
  "next_cursor": "01KGD5FGHIJK123456789ABCDE",
  "has_more": true
}
```

Success response (200 OK) for `?after=01KGD5E74RYS0WTNARQC92S1P7&limit=1`:
```json
{
  "data": [
    {
      "id": "01KGD5FGHIJK123456789ABCDE",
      "title": "USB Keyboard",
      "price": "49.99",
      "quantity": 5,
      "brand": "Wow"
    }
  ],
  "total": 1,
  "next_cursor": null,
  "has_more": false
}
```

### FR-8: Aggregation Endpoints Response Examples

**FR-8.1: GET /{collection}:count Response**
Success response (200 OK):
```json
{
  "count": 42
}
```

Success response (200 OK) with filter `?quantity[gt]=5`:
```json
{
  "count": 15
}
```

**FR-8.2: GET /{collection}:sum Response**
Success response (200 OK) for `?field=quantity`:
```json
{
  "sum": 150
}
```

Success response (200 OK) for decimal field `?field=price`:
```json
{
  "sum": "1234.56"
}
```

Error response (400 Bad Request):
```json
{
  "error": "Field 'quantity' is required for sum operation",
  "code": 400
}
```

Error response (422 Unprocessable Entity):
```json
{
  "error": "Field 'title' is not a numeric type. Sum operation requires integer or decimal fields.",
  "code": 422
}
```

**FR-8.3: GET /{collection}:avg Response**
Success response (200 OK) for `?field=quantity`:
```json
{
  "avg": 25.5
}
```

Success response (200 OK) for decimal field `?field=price`:
```json
{
  "avg": "49.99"
}
```

**FR-8.4: GET /{collection}:min Response**
Success response (200 OK) for `?field=quantity`:
```json
{
  "min": 1
}
```

Success response (200 OK) for decimal field `?field=price`:
```json
{
  "min": "9.99"
}
```

**FR-8.5: GET /{collection}:max Response**
Success response (200 OK) for `?field=quantity`:
```json
{
  "max": 100
}
```

Success response (200 OK) for decimal field `?field=price`:
```json
{
  "max": "999.99"
}
```

### FR-9: Health and System Endpoints Response Examples

**FR-9.1: GET /health Response**
Success response (200 OK):
```json
{
  "name": "moon",
  "status": "live",
  "version": "1.99"
}
```

**Response Headers:**
```
HTTP/1.1 200 OK
Content-Type: application/json
Access-Control-Allow-Origin: *
```

Failure response (503 Service Unavailable):
```json
{
  "name": "moon",
  "status": "down",
  "version": "1.99",
  "error": "Database connection failed"
}
```

**FR-9.2: GET /doc Response**
Success response (200 OK):
```
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8
Access-Control-Allow-Origin: *

<!DOCTYPE html>
<html>
<head><title>Moon API Documentation</title></head>
...
</html>
```

**FR-9.3: GET /doc/llms-full.txt Response**
Success response (200 OK):
```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Access-Control-Allow-Origin: *

# Moon – Instructions and API Documentation for AI Coding Agents
...
```

**FR-9.4: POST /doc:refresh Response**
Success response (200 OK):
```json
{
  "message": "Documentation cache refreshed successfully"
}
```

Error response (401 Unauthorized):
```json
{
  "error": "Unauthorized",
  "code": 401
}
```

### FR-10: Rate Limiting Response Headers

**FR-10.1: Successful Request Headers**
All authenticated endpoint responses include rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1706875200
```

**FR-10.2: Rate Limit Exceeded Response**
Error response (429 Too Many Requests):
```json
{
  "error": "Rate limit exceeded. Try again later.",
  "code": 429
}
```

**Response Headers:**
```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1706875260
Retry-After: 60
```

### FR-11: CORS Preflight Response Examples

**FR-11.1: OPTIONS Request Response**
Success response (200 OK) for `OPTIONS /collections:list`:
```
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 3600
Content-Length: 0
```

Success response (200 OK) for specific origin `OPTIONS /data/products:list`:
```
HTTP/1.1 200 OK
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 3600
Content-Length: 0
```

### FR-12: Documentation Template Implementation

**FR-12.1: Template Structure Updates**
File: `cmd/moon/internal/handlers/templates/doc.md.tmpl`

For each endpoint section, update the template to include:
1. **Request example** (existing, keep as-is)
2. **Success response example** with status code and JSON body (NEW)
3. **Response headers example** showing relevant headers (NEW)
4. **Common error response examples** (NEW)

Example template pattern:
```markdown
**List Collections**

```bash
curl -s {{$ApiURL}}/collections:list \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**
```json
{
  "collections": [
    {
      "name": "products",
      "columns": [
        {"name": "id", "type": "string", "nullable": false},
        {"name": "title", "type": "string", "nullable": false}
      ]
    }
  ],
  "total": 1
}
```

**Response Headers:**
```
HTTP/1.1 200 OK
Content-Type: application/json
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1706875200
```

**Error Response (401 Unauthorized):**
```json
{
  "error": "Unauthorized",
  "code": 401
}
```
```

**FR-12.2: Response Example Placeholders**
Use consistent placeholder values across all examples:
- User IDs: `01JXRZ7HDKTSV4RRFFQ69G5FAA`, `01JXRZ9ABCDEFGH123456789`
- Record IDs: `01KGD5E74RYS0WTNARQC92S1P7`, `01KGD5FGHIJK123456789ABCDE`
- API Key IDs: `01KGF0NGEHYKJR8PV5KHQBDHKB`, `01KGF2XYZABCDEFGHIJKLMNOPQ`
- JWT tokens: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...` (truncated)
- API keys: `moon_live_ArPaQDdyh4gp9EKF42Xrq7AOAAJZmKiCtTCv1m2rj931QxmPkiXuSV8Un0qllMiN`
- Timestamps: Use consistent format `2026-02-03T10:45:00Z` with logical progression
- Usernames: `admin`, `johndoe`, `newuser`
- Emails: `admin@example.com`, `john@example.com`, `newuser@example.com`
- Collection names: `products`, `customers`, `orders`
- Product data: `Wireless Mouse`, `USB Keyboard`, `Laptop Stand`

**FR-12.3: JSON Appendix Enhancement**
Update the JSON appendix section in `doc.md.tmpl` to include response schemas:
```json
{
  "endpoints": {
    "/auth:login": {
      "method": "POST",
      "description": "Authenticate user and receive tokens",
      "request_body": {
        "username": "string (required)",
        "password": "string (required)"
      },
      "responses": {
        "200": {
          "description": "Successful authentication",
          "body": {
            "access_token": "string (JWT)",
            "refresh_token": "string (JWT)",
            "token_type": "string",
            "expires_in": "integer (seconds)",
            "user": {
              "id": "string (ULID)",
              "username": "string",
              "email": "string",
              "role": "string (user|admin)",
              "created_at": "string (ISO 8601)"
            }
          }
        },
        "401": {
          "description": "Invalid credentials",
          "body": {
            "error": "string",
            "code": "integer"
          }
        }
      }
    }
  }
}
```

### FR-13: Testing and Validation Requirements

**FR-13.1: Response Example Accuracy**
All response examples must be validated against actual API behavior:
- Execute actual API calls with test data
- Capture real responses (headers + body)
- Verify field names, types, and structures match documentation
- Test edge cases (empty results, pagination, errors)
- Validate error codes and messages match implementation

**FR-13.2: Documentation Generation Tests**
Create test script: `scripts/test-doc-response-examples.sh`

Test scenarios:
1. Generate documentation via `/doc:refresh`
2. Verify all endpoints include response examples
3. Check that response JSON is valid (parseable)
4. Verify status codes are correct (200, 201, 400, 401, etc.)
5. Confirm error examples follow standard format
6. Validate that pagination examples include next_cursor and has_more
7. Check rate limit headers are documented
8. Verify CORS headers are shown for public endpoints

**FR-13.3: Response Format Consistency Check**
Automated validation that all response examples:
- Include HTTP status line
- Show Content-Type header
- Use proper JSON formatting (valid syntax, 2-space indent)
- Include error code in error responses
- Show rate limit headers for authenticated endpoints
- Include pagination metadata where applicable

**FR-13.4: Documentation Diff Testing**
Before/after comparison:
- Capture current documentation output
- Implement changes
- Generate new documentation
- Compare: Only response examples added, no existing content removed or altered
- Verify all curl examples remain unchanged

### FR-14: Documentation Maintenance

**FR-14.1: Automated Response Example Updates**
When API response format changes:
- Update corresponding response examples in `doc.md.tmpl`
- Update JSON appendix schema definitions
- Regenerate documentation via `/doc:refresh`
- Run validation tests to ensure accuracy

**FR-14.2: Version Control for Response Examples**
Track response example changes:
- Document response format changes in commit messages
- Include PRD references when response format evolves
- Update response examples as part of feature implementation PRDs
- Maintain backward compatibility notes if response format changes

**FR-14.3: Response Example Coverage Checklist**
Every endpoint must have:
- [ ] At least one success response example (200/201)
- [ ] Primary error response example (most common error for that endpoint)
- [ ] Authentication error example (401) if endpoint is protected
- [ ] Authorization error example (403) if endpoint requires admin role
- [ ] HTTP status line and relevant headers
- [ ] Valid JSON with proper formatting

---

## Acceptance

### AC-1: Functional Requirements

- [ ] All authentication endpoints (`/auth:*`) include success and error response examples
- [ ] All user management endpoints (`/users:*`) include response examples with 200/201/401/403/404 status codes
- [ ] All API key management endpoints (`/apikeys:*`) include response examples with full JSON structure
- [ ] All collection management endpoints (`/collections:*`) include response examples for create/read/update/delete operations
- [ ] All data access endpoints (`/{collection}:*`) include response examples with sample data
- [ ] Query option response examples show filtering, sorting, search, field selection, and pagination results
- [ ] Aggregation endpoint response examples show count, sum, avg, min, max operations with proper data types
- [ ] Health and system endpoint response examples include both success and failure cases
- [ ] Rate limiting response headers documented for all authenticated endpoints
- [ ] CORS preflight response examples included for public endpoints
- [ ] All error responses follow standard format: `{"error": "...", "code": 400}`
- [ ] Response examples include HTTP status line (e.g., `HTTP/1.1 200 OK`)
- [ ] Relevant headers documented (Content-Type, rate limiting, CORS, pagination)
- [ ] Decimal field responses shown as strings (e.g., `"49.99"`)
- [ ] Empty result cases documented (empty arrays, no pagination cursor)

### AC-2: Documentation Template

- [ ] `cmd/moon/internal/handlers/templates/doc.md.tmpl` updated with response examples for all endpoints
- [ ] Response example format is consistent across all endpoints
- [ ] HTTP status codes shown for all response examples
- [ ] Response headers section added where relevant
- [ ] JSON formatting is valid and consistent (2-space indentation)
- [ ] Placeholder values are consistent across all examples (IDs, tokens, timestamps)
- [ ] Success and error response examples clearly labeled
- [ ] Multi-scenario responses shown (metadata update vs key rotation, email vs password change)
- [ ] Pagination responses include next_cursor and has_more fields
- [ ] Field selection responses show only requested fields (plus id)

### AC-3: JSON Appendix Enhancement

- [ ] JSON appendix updated to include response schemas for all endpoints
- [ ] Response schemas show all fields with types and descriptions
- [ ] Error response schemas documented separately
- [ ] Status codes documented for each endpoint (success + error cases)
- [ ] Optional vs required fields clearly indicated
- [ ] Nested object structures properly represented (user object in login response, collection object in create response)

### AC-4: Error Response Coverage

- [ ] 400 Bad Request examples shown for validation errors and invalid input
- [ ] 401 Unauthorized examples shown for missing/invalid authentication
- [ ] 403 Forbidden examples shown for admin-only endpoints accessed by users
- [ ] 404 Not Found examples shown for missing collections/records/users
- [ ] 409 Conflict examples shown for duplicate creation attempts
- [ ] 422 Unprocessable Entity examples shown for field validation failures
- [ ] 429 Too Many Requests example shown with rate limit headers and Retry-After
- [ ] 500 Internal Server Error example shown (generic server error)
- [ ] Error messages are descriptive and actionable

### AC-5: Response Header Documentation

- [ ] Content-Type header shown in all response examples
- [ ] X-RateLimit-* headers documented for authenticated endpoints
- [ ] Access-Control-Allow-* headers shown for public endpoint responses
- [ ] Retry-After header shown in 429 rate limit response
- [ ] HTTP status line included in all response examples
- [ ] Headers shown in consistent format across all examples

### AC-6: Data Type Representation

- [ ] String values shown in double quotes
- [ ] Integer values shown without quotes
- [ ] Boolean values shown as `true` or `false` (not quoted)
- [ ] Decimal values shown as quoted strings (e.g., `"29.99"`)
- [ ] Datetime values in ISO 8601 format with quotes (e.g., `"2026-02-03T10:45:00Z"`)
- [ ] JSON objects and arrays properly formatted
- [ ] Null values shown as `null` (not quoted)
- [ ] ULID identifiers shown as 26-character alphanumeric strings

### AC-7: Pagination Response Examples

- [ ] List responses include total count
- [ ] Paginated responses include next_cursor when more results available
- [ ] has_more field included in paginated responses (true/false)
- [ ] Last page response shows next_cursor as null and has_more as false
- [ ] Cursor-based pagination example shows ?after parameter usage
- [ ] Limit parameter effect shown in response examples

### AC-8: Testing and Validation

- [ ] All response examples validated against actual API behavior
- [ ] Test script created to verify documentation accuracy
- [ ] JSON syntax validated in all response examples
- [ ] Response examples tested with actual curl commands
- [ ] Error responses tested by intentionally triggering each error condition
- [ ] Pagination tested to verify cursor and has_more behavior
- [ ] Rate limiting tested to capture 429 response
- [ ] Documentation regeneration tested via /doc:refresh endpoint

### AC-9: Consistency and Quality

- [ ] Response examples use consistent placeholder values (IDs, names, timestamps)
- [ ] All timestamps follow logical chronological order (created_at before updated_at)
- [ ] Sample data is realistic and internally consistent (matching IDs in request/response)
- [ ] Field names in responses match collection schema definitions
- [ ] Response structure matches JSON appendix schema definitions
- [ ] No placeholder text like "..." or "<replace_me>" in final documentation
- [ ] All response examples properly formatted with markdown code blocks
- [ ] Language specifier used for JSON code blocks: ```json

### AC-10: Documentation Completeness

- [ ] Every curl request example followed by at least one response example
- [ ] Multi-step operations show intermediate responses (login → save token → use token)
- [ ] Both happy path and error path documented for each endpoint
- [ ] Edge cases documented (empty lists, last page, no search results)
- [ ] All query parameters shown with corresponding response examples
- [ ] Aggregation operations shown with different field types (integer and decimal)
- [ ] Public endpoints response examples show CORS headers
- [ ] Admin-only endpoints show 403 error for non-admin users

### AC-11: Backward Compatibility

- [ ] All existing curl request examples remain unchanged
- [ ] No breaking changes to API behavior
- [ ] Documentation template changes are additive only
- [ ] JSON appendix maintains existing structure, adds response schemas
- [ ] Existing documentation sections (intro, authentication, query options) unchanged
- [ ] Public endpoint behavior remains the same

### AC-12: Code Quality

- [ ] Template file (`doc.md.tmpl`) follows Go template best practices
- [ ] Response examples use template variables where appropriate (e.g., `{{$ApiURL}}`)
- [ ] No hardcoded URLs or ports in response examples
- [ ] Template changes do not break documentation generation
- [ ] All Go template syntax is valid and properly escaped
- [ ] Comments added to template explaining response example sections

---

## Checklist

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Update API doc template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect these changes.
- [ ] Run all tests and ensure 100% pass rate.
- [ ] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.
