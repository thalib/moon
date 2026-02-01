// Package registry provides in-memory schema caching for dynamic database
// management. It maintains a thread-safe registry of collection schemas using
// sync.Map for zero-latency validation before database operations.
package registry

import (
	"fmt"
	"sync"
)

// ColumnType represents the data type of a column
type ColumnType string

const (
	TypeString   ColumnType = "string"
	TypeInteger  ColumnType = "integer"
	TypeBoolean  ColumnType = "boolean"
	TypeDatetime ColumnType = "datetime"
	TypeJSON     ColumnType = "json"
	TypeDecimal  ColumnType = "decimal"
)

// Column represents a single column in a collection
type Column struct {
	Name         string     `json:"name"`
	Type         ColumnType `json:"type"`
	Nullable     bool       `json:"nullable"`
	Unique       bool       `json:"unique"`
	DefaultValue *string    `json:"default_value,omitempty"`
}

// Collection represents a database table schema
type Collection struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// SchemaRegistry manages the in-memory cache of collection schemas
type SchemaRegistry struct {
	collections sync.Map // map[string]*Collection
}

// NewSchemaRegistry creates a new schema registry
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{}
}

// Set stores or updates a collection schema in the registry
func (r *SchemaRegistry) Set(collection *Collection) error {
	if collection == nil {
		return fmt.Errorf("collection cannot be nil")
	}

	if collection.Name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	// Store a copy to prevent external modifications
	copy := &Collection{
		Name:    collection.Name,
		Columns: make([]Column, len(collection.Columns)),
	}
	for i, col := range collection.Columns {
		copy.Columns[i] = col
	}

	r.collections.Store(collection.Name, copy)
	return nil
}

// Get retrieves a collection schema from the registry
func (r *SchemaRegistry) Get(name string) (*Collection, bool) {
	value, ok := r.collections.Load(name)
	if !ok {
		return nil, false
	}

	collection, ok := value.(*Collection)
	if !ok {
		// This should never happen, but handle it safely
		return nil, false
	}

	// Return a copy to prevent external modifications
	copy := &Collection{
		Name:    collection.Name,
		Columns: make([]Column, len(collection.Columns)),
	}
	for i, col := range collection.Columns {
		copy.Columns[i] = col
	}

	return copy, true
}

// Delete removes a collection schema from the registry
func (r *SchemaRegistry) Delete(name string) error {
	if name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	r.collections.Delete(name)
	return nil
}

// Exists checks if a collection exists in the registry
func (r *SchemaRegistry) Exists(name string) bool {
	_, ok := r.collections.Load(name)
	return ok
}

// List returns all collection names in the registry
func (r *SchemaRegistry) List() []string {
	var names []string

	r.collections.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})

	return names
}

// GetAll returns all collections in the registry
func (r *SchemaRegistry) GetAll() []*Collection {
	var collections []*Collection

	r.collections.Range(func(key, value any) bool {
		collection := value.(*Collection)

		// Return a copy to prevent external modifications
		copy := &Collection{
			Name:    collection.Name,
			Columns: make([]Column, len(collection.Columns)),
		}
		for i, col := range collection.Columns {
			copy.Columns[i] = col
		}

		collections = append(collections, copy)
		return true
	})

	return collections
}

// Clear removes all collections from the registry
func (r *SchemaRegistry) Clear() {
	r.collections.Range(func(key, value any) bool {
		r.collections.Delete(key)
		return true
	})
}

// Count returns the number of collections in the registry
func (r *SchemaRegistry) Count() int {
	count := 0
	r.collections.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

// ValidateColumnType checks if a column type is valid
func ValidateColumnType(colType ColumnType) bool {
	switch colType {
	case TypeString, TypeInteger, TypeBoolean, TypeDatetime, TypeJSON, TypeDecimal:
		return true
	default:
		return false
	}
}

// MapGoTypeToColumnType maps Go types to ColumnType
func MapGoTypeToColumnType(goType string) (ColumnType, error) {
	switch goType {
	case "string":
		return TypeString, nil
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return TypeInteger, nil
	case "bool":
		return TypeBoolean, nil
	case "time.Time":
		return TypeDatetime, nil
	default:
		return "", fmt.Errorf("unsupported Go type: %s", goType)
	}
}
