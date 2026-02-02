package constants

// System table names used by Moon's internal authentication and authorization system.
// These tables are prefixed with "moon_" to distinguish them from user-created collections
// and should never be exposed through the collections API endpoints.
const (
	// TableUsers is the system table for user accounts
	TableUsers = "moon_users"

	// TableRefreshTokens is the system table for JWT refresh tokens
	TableRefreshTokens = "moon_refresh_tokens"

	// TableAPIKeys is the system table for API key credentials
	TableAPIKeys = "moon_apikeys"

	// TableBlacklistedTokens is the system table for revoked JWT access tokens
	TableBlacklistedTokens = "moon_blacklisted_tokens"
)

// SystemTables is a list of all system tables that should be excluded from
// collections:list endpoint and collection management operations.
var SystemTables = []string{
	TableUsers,
	TableRefreshTokens,
	TableAPIKeys,
	TableBlacklistedTokens,
}

// systemTableMap is a map for O(1) lookup of system tables.
var systemTableMap = map[string]bool{
	TableUsers:             true,
	TableRefreshTokens:     true,
	TableAPIKeys:           true,
	TableBlacklistedTokens: true,
}

// IsSystemTable checks if a given table name is a system table.
func IsSystemTable(tableName string) bool {
	return systemTableMap[tableName]
}
