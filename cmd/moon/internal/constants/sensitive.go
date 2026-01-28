package constants

// Sensitive field names that should be masked/redacted in logs and error messages
// to prevent accidental exposure of credentials or secrets.
// Used in: logging/logger.go for automatic field masking
var SensitiveFields = []string{
	"password",
	"token",
	"secret",
	"api_key",
	"apikey",
	"authorization",
}

// RedactedPlaceholder is the string used to replace sensitive values in logs.
// Used in: logging/logger.go
const RedactedPlaceholder = "***REDACTED***"
