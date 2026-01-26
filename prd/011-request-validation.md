## Overview
- Implement request validation layer using In-Memory Schema Registry
- Validate JSON payloads against cached schema before database operations
- Provide clear, actionable error messages for validation failures

## Requirements
- Create `internal/validation/validator.go` with Validator interface
- Validate request body fields exist in collection schema
- Validate field types match expected column types
- Validate required fields are present
- Validate field value constraints (length, format, range)
- Support custom validation rules per column
- Return detailed validation errors (field name, expected type, actual value)
- Reject unknown fields (strict mode) or ignore them (permissive mode)
- Validate collection name exists in registry

## Acceptance
- Invalid payloads are rejected before database query
- Error messages clearly indicate validation failures
- Performance: validation completes in microseconds
- Unit tests cover all validation scenarios
- Integration tests verify validation in request flow
