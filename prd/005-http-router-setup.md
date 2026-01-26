## Overview
- Set up HTTP router with support for AIP-136 custom action pattern (colon separator)
- Configure versioned API routes under `/api/v1`
- Implement request parsing for `/:name:action` pattern

## Requirements
- Use Gin or Chi router (evaluate performance characteristics)
- Parse routes with pattern: `/{collection}:{action}` (e.g., `/products:list`)
- Support `/collections:{action}` for schema management
- Support `/{collectionName}:{action}` for data access
- Create `internal/handlers/` package for HTTP handlers
- Implement middleware chain support
- Configure graceful shutdown
- Set up request/response logging
- Return consistent JSON error responses

## Acceptance
- Router correctly parses `:action` suffix from paths
- Routes dispatch to correct handlers
- Graceful shutdown completes in-flight requests
- Unit tests verify route parsing logic
- Integration tests verify endpoint routing
