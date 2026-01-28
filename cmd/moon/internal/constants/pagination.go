package constants

// Pagination constants for list operations and data retrieval.
// These define default limits and offsets for paginated API responses.
const (
	// DefaultPaginationLimit is the default maximum number of records to return
	// in a single paginated response when no limit is specified.
	// Used in: handlers/data.go
	// Purpose: Prevents excessive memory usage and improves response times
	// Default: 100 records
	DefaultPaginationLimit = 100

	// DefaultPaginationOffset is the default starting position for pagination
	// when no offset is specified.
	// Used in: handlers/data.go
	// Purpose: Standard starting point for paginated queries
	// Default: 0 (start from beginning)
	DefaultPaginationOffset = 0
)

// Query parameter names for pagination.
const (
	// QueryParamLimit is the URL query parameter name for specifying page size.
	// Used in: handlers/data.go
	QueryParamLimit = "limit"

	// QueryParamOffset is the URL query parameter name for specifying page offset.
	// Used in: handlers/data.go
	QueryParamOffset = "offset"

	// QueryParamID is the URL query parameter name for resource ID.
	// Used in: handlers/data.go
	QueryParamID = "id"
)
