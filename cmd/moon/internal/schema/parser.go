package schema

import (
	"net/url"
	"strings"
)

// QueryMode represents how the schema should be returned
type QueryMode int

const (
	// ModeNone means no schema should be returned (default)
	ModeNone QueryMode = iota
	// ModeBoth means return both data and schema
	ModeBoth
	// ModeOnly means return only schema, no data
	ModeOnly
)

// ParseQueryParameter parses the ?schema query parameter
func ParseQueryParameter(values url.Values) QueryMode {
	// Check if the key exists at all
	if !values.Has("schema") {
		return ModeNone
	}

	schemaParam := values.Get("schema")

	// Normalize to lowercase
	schemaParam = strings.ToLower(schemaParam)

	// Empty value or "true" means include both data and schema
	if schemaParam == "" || schemaParam == "true" {
		return ModeBoth
	}

	// "only" means schema only
	if schemaParam == "only" {
		return ModeOnly
	}

	// "false" means no schema
	if schemaParam == "false" {
		return ModeNone
	}

	// Any other value is treated as ModeBoth
	return ModeBoth
}
