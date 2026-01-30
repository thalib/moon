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

	// HeaderAPIKey is the custom header for API key authentication.
	// Used in: middleware/apikey.go
	// Purpose: Alternative authentication mechanism using API keys
	// Note: Default header name can be overridden in config
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
)
