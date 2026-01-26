## Overview
- Implement health check endpoints for container orchestration and monitoring
- Provide liveness and readiness probes for Kubernetes compatibility
- Report component health status (database, cache)

## Requirements
- Create `GET /health` endpoint for basic liveness check
- Create `GET /health/ready` endpoint for readiness check
- Check database connectivity in readiness probe
- Check registry initialization in readiness probe
- Return HTTP 200 for healthy, 503 for unhealthy
- Include component status in response body
- Support configurable health check timeout
- Do not require authentication for health endpoints
- Keep health checks lightweight (minimal resource usage)

## Acceptance
- Liveness endpoint responds quickly (< 100ms)
- Readiness endpoint accurately reflects system state
- Unhealthy components cause appropriate status code
- Unit tests cover health check logic
- Integration tests verify health endpoints
