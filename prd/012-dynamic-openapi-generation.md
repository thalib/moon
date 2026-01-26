## Overview
- Generate OpenAPI/Swagger documentation dynamically from In-Memory Schema Registry
- Documentation reflects current database state automatically
- Include authentication requirements and example payloads

## Requirements
- Create `internal/openapi/generator.go` for dynamic spec generation
- Generate OpenAPI 3.0 specification from cached schema
- Include all collection endpoints with proper request/response schemas
- Document authentication requirements (JWT, API Key)
- Generate example payloads for each endpoint
- Serve Swagger UI at `/docs` or `/swagger`
- Serve OpenAPI JSON/YAML at `/openapi.json` or `/openapi.yaml`
- Regenerate spec when schema changes (lazy or eager refresh)
- Include proper HTTP status codes and error responses

## Acceptance
- OpenAPI spec accurately reflects current database schema
- Swagger UI is accessible and functional
- New collections appear in docs after creation
- Example payloads are valid and helpful
- Unit tests verify spec generation logic
