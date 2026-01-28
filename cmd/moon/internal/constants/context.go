package constants

// Context keys for storing and retrieving values from request contexts.
// Using typed constants prevents key collisions and provides type safety.
//
// Note: In production code, it's recommended to use unexported custom types
// for context keys to prevent collisions. However, for simplicity and
// backwards compatibility, we're using string constants here.
const (
	// ContextKeyRequestID is the context key for storing request IDs.
	// Used in: errors/errors.go, logging/logger.go
	// Purpose: Request tracking and correlation across logs
	ContextKeyRequestID = "request_id"

	// ContextKeyUserClaims is the context key for storing authenticated user JWT claims.
	// Used in: middleware/auth.go
	// Purpose: Passing authenticated user information through middleware chain
	ContextKeyUserClaims = "user_claims"

	// ContextKeyAPIKeyInfo is the context key for storing API key information.
	// Used in: middleware/apikey.go
	// Purpose: Passing API key metadata through middleware chain
	ContextKeyAPIKeyInfo = "api_key_info"
)
