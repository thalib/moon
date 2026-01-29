package constants

import "time"

// Timeout and duration constants used throughout the application.
// These constants define time limits for various operations to prevent
// indefinite blocking and ensure responsive behavior.
const (
	// ShutdownTimeout is the maximum time allowed for graceful shutdown.
	// Used in: shutdown/shutdown.go
	// Purpose: Allows in-flight requests to complete before forcing shutdown
	// Default: 30 seconds
	ShutdownTimeout = 30 * time.Second

	// HTTPReadTimeout is the maximum duration for reading the entire request,
	// including the body. A zero or negative value means there will be no timeout.
	// Used in: server/server.go
	// Purpose: Prevents slow-read attacks and resource exhaustion
	// Default: 15 seconds
	HTTPReadTimeout = 15 * time.Second

	// HTTPWriteTimeout is the maximum duration before timing out writes of the response.
	// It includes the time to read the request header.
	// Used in: server/server.go
	// Purpose: Prevents slow-write attacks and ensures timely responses
	// Default: 15 seconds
	HTTPWriteTimeout = 15 * time.Second

	// HTTPIdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alives are enabled.
	// Used in: server/server.go
	// Purpose: Prevents indefinite connection holding and resource exhaustion
	// Default: 60 seconds
	HTTPIdleTimeout = 60 * time.Second

	// HealthCheckTimeout is the maximum time allowed for a single health check operation.
	// Used in: health/health.go
	// Purpose: Ensures health checks don't block indefinitely
	// Default: 5 seconds
	HealthCheckTimeout = 5 * time.Second

	// JWTClockSkew is the tolerance for JWT token expiration time validation.
	// Used in: middleware/auth.go
	// Purpose: Accounts for clock drift between servers
	// Default: 30 seconds
	JWTClockSkew = 30 * time.Second

	// SlowQueryThreshold is the duration threshold for logging slow database queries.
	// Used in: logging/logger.go
	// Purpose: Identifies performance issues by logging queries that exceed this duration
	// Default: 500 milliseconds
	SlowQueryThreshold = 500 * time.Millisecond

	// ConsistencyCheckTimeout is the maximum time allowed for consistency check operations.
	// Used in: consistency/checker.go
	// Purpose: Prevents consistency checks from blocking startup indefinitely
	// Default: 5 seconds (configurable via recovery.check_timeout)
	ConsistencyCheckTimeout = 5 * time.Second
)
