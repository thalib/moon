## Overview
- Implement structured logging for observability and debugging
- Support request tracing with correlation IDs
- Enable performance monitoring and metrics collection

## Requirements
- Create `internal/logging/logger.go` with structured logger
- Use structured logging library (zerolog or zap)
- Log format: JSON in production, human-readable in development
- Include fields: timestamp, level, message, request_id, duration, status
- Implement request logging middleware (method, path, status, duration)
- Support log levels: debug, info, warn, error
- Configure log level via environment/config
- Add correlation/request ID to all logs within a request
- Log slow queries (configurable threshold)
- Avoid logging sensitive data (passwords, tokens)

## Acceptance
- All requests are logged with relevant metadata
- Logs are parseable by log aggregation tools
- Request correlation works across the request lifecycle
- Unit tests verify logging behavior
- Log output is configurable per environment
