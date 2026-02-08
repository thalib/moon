// Package constants provides centralized constant definitions for the Moon application.
// All hardcoded values that are reused across the codebase should be defined here
// to ensure consistency, maintainability, and ease of configuration.
package constants

// HTTP header names used throughout the application.
// These constants ensure consistent header naming across all components.
const (
	// HeaderRequestID is the HTTP header used for request tracking and correlation.
	// Used in: errors/errors.go, logging/logger.go
	// Purpose: Enables request tracing across distributed systems and logs
	HeaderRequestID = "X-Request-ID"

	// HeaderAuthorization is the standard HTTP Authorization header.
	// Used in: middleware/auth.go
	// Purpose: Contains JWT bearer tokens for authentication
	HeaderAuthorization = "Authorization"

	// HeaderAPIKey is the legacy custom header for API key authentication.
	// DEPRECATED: X-API-Key header is no longer supported (removed in PRD-059).
	// This constant is retained only for reference in error messages and migration documentation.
	// All authentication now uses Authorization: Bearer <token> exclusively.
	HeaderAPIKey = "X-API-Key"

	// HeaderContentType is the standard HTTP Content-Type header.
	// Used in: multiple handlers and middleware
	// Purpose: Specifies the media type of the request/response body
	HeaderContentType = "Content-Type"
)

// MIME types used in HTTP responses.
const (
	// MIMEApplicationJSON is the MIME type for JSON responses.
	// Used throughout the API for JSON content encoding
	MIMEApplicationJSON = "application/json"

	// MIMETextPlain is the MIME type for plain text responses.
	// Used for simple text responses like the root message
	MIMETextPlain = "text/plain; charset=utf-8"
)

// Authentication schemes and prefixes.
const (
	// AuthSchemeBearer is the authentication scheme for JWT tokens.
	// Used in: middleware/auth.go
	// Format: "Bearer <token>" in Authorization header
	AuthSchemeBearer = "Bearer"

	// APIKeyPrefix is the required prefix for all API keys.
	// Used in: middleware/auth.go, handlers/apikeys.go
	// Format: "moon_live_<64_chars>" for production keys
	// Purpose: Allows easy identification and validation of API keys
	APIKeyPrefix = "moon_live_"

	// MinAPIKeyLengthWithPrefix is the minimum total length of an API key including prefix.
	// Used in: validation logic for API key format
	// Value: 74 characters (10 char prefix + 64 char key)
	MinAPIKeyLengthWithPrefix = 74
)
