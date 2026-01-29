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

	// ConsistencyErrorMessages contains error messages for consistency check failures.
	// Used in: consistency/checker.go for reporting consistency issues
	ConsistencyErrorMessages = struct {
		OrphanedTable    string
		OrphanedRegistry string
		RepairFailed     string
		CheckTimeout     string
	}{
		OrphanedTable:    "table exists in database but not in registry",
		OrphanedRegistry: "collection registered but table does not exist",
		RepairFailed:     "failed to repair consistency issues",
		CheckTimeout:     "consistency check timed out",
	}
)
