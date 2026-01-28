package constants

// Validation constants for input validation and data constraints.
const (
	// DefaultVarcharMaxLength is the default maximum length for VARCHAR fields
	// when no specific length is specified in the schema.
	// Used in: validation/validator.go
	// Purpose: Standard SQL VARCHAR constraint for string fields
	// Default: 255 characters
	DefaultVarcharMaxLength = 255

	// MinAPIKeyLength is the minimum required length for API keys.
	// Used in: middleware/apikey.go
	// Purpose: Security requirement to ensure API keys have sufficient entropy
	// Default: 40 characters
	MinAPIKeyLength = 40
)

// Regular expression patterns for validation.
const (
	// CollectionNamePattern is the regex pattern for valid collection names.
	// Pattern: Must start with a letter, followed by letters, numbers, or underscores.
	// Used in: handlers/collections.go
	// Purpose: Ensures collection names are valid SQL identifiers
	CollectionNamePattern = `^[a-zA-Z][a-zA-Z0-9_]*$`
)
