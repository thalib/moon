## Overview
- Implement JWT authentication middleware for securing API endpoints
- Validate tokens before requests reach dynamic handlers
- Support configurable token expiration and signing keys

## Requirements
- Create `internal/middleware/auth.go` with JWT middleware
- Parse JWT from Authorization header (Bearer token format)
- Validate token signature using configured secret
- Check token expiration
- Extract claims (user ID, roles) and attach to request context
- Support configurable protected/unprotected routes
- Return 401 Unauthorized for invalid/missing tokens
- Return 403 Forbidden for insufficient permissions
- Log authentication failures for security monitoring

## Acceptance
- Valid JWT tokens grant access to protected endpoints
- Invalid/expired tokens are rejected with proper error codes
- Claims are accessible in request handlers
- Unit tests cover token validation logic
- Integration tests verify authentication flow end-to-end
