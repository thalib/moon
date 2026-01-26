## Overview
- Implement API Key authentication as alternative to JWT
- Support simple allow/deny permissions per endpoint
- Enable machine-to-machine authentication scenarios

## Requirements
- Create API Key validation middleware in `internal/middleware/apikey.go`
- Accept API Key via header (X-API-Key) or query parameter
- Store and validate API keys securely (hashed storage)
- Support multiple API keys with different permission levels
- Implement per-endpoint permission configuration
- Support rate limiting per API key (optional, configurable)
- Log API key usage for audit trail
- Return 401 for invalid keys, 403 for insufficient permissions

## Acceptance
- Valid API keys grant access based on configured permissions
- Invalid keys are rejected with proper error codes
- Permissions are enforced per endpoint
- Unit tests cover key validation and permission logic
- Integration tests verify API key authentication flow
