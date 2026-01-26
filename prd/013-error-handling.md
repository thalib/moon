## Overview
- Implement consistent error handling and response format across all endpoints
- Provide structured error responses with codes, messages, and details
- Support debugging while protecting sensitive information in production

## Requirements
- Create `internal/errors/errors.go` with custom error types
- Define standard error codes for common scenarios (validation, auth, not found, etc.)
- Implement error response struct: code, message, details, request_id
- Create error handling middleware for catching panics
- Log errors with appropriate severity levels
- Hide internal error details in production (expose only in development)
- Include request correlation ID in error responses
- Map database errors to appropriate HTTP status codes
- Support error wrapping for context preservation

## Acceptance
- All endpoints return consistent error format
- Sensitive details are hidden in production
- Error logs include sufficient debugging information
- Unit tests cover error handling scenarios
- Integration tests verify error responses
