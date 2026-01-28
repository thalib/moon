package constants

// Database error detection patterns.
// These patterns are used to identify specific database error conditions
// by matching against error messages from different database drivers.
// Used in: errors/errors.go for error classification and handling
var (
	// DuplicateKeyPatterns contains error message patterns that indicate
	// a duplicate key or unique constraint violation across different databases.
	DuplicateKeyPatterns = []string{
		"duplicate",
		"unique constraint",
		"UNIQUE constraint",
	}

	// ConnectionErrorPatterns contains error message patterns that indicate
	// network or connection-related database errors.
	ConnectionErrorPatterns = []string{
		"connection refused",
		"no such host",
		"timeout",
	}
)
