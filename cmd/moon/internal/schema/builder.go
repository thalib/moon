// Package schema provides schema metadata generation for API responses (PRD-053).
package schema

import (
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// FieldSchema represents the schema of a single field
type FieldSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable"`
	Default     *any   `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// Schema represents the complete schema metadata for a resource
type Schema struct {
	Collection string        `json:"collection"`
	Fields     []FieldSchema `json:"fields"`
	PrimaryKey string        `json:"primary_key"`
	Metadata   *Metadata     `json:"metadata,omitempty"`
}

// Metadata contains system-generated field information
type Metadata struct {
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Builder creates schema metadata from collection definitions
type Builder struct{}

// NewBuilder creates a new schema builder
func NewBuilder() *Builder {
	return &Builder{}
}

// FromCollection generates schema from a registry collection
func (b *Builder) FromCollection(collection *registry.Collection) *Schema {
	schema := &Schema{
		Collection: collection.Name,
		Fields:     make([]FieldSchema, 0, len(collection.Columns)),
		PrimaryKey: "id", // All collections use 'id' (ulid) as primary key
		Metadata: &Metadata{
			CreatedAt: "datetime",
			UpdatedAt: "datetime",
		},
	}

	// Add the primary key field first
	schema.Fields = append(schema.Fields, FieldSchema{
		Name:     "id",
		Type:     "string",
		Nullable: false,
	})

	// Add all other fields, excluding internal system columns (id, ulid)
	for _, col := range collection.Columns {
		// Skip internal system columns - they should never be exposed
		if col.Name == "id" || col.Name == "ulid" {
			continue
		}

		fieldSchema := FieldSchema{
			Name:     col.Name,
			Type:     string(col.Type),
			Nullable: col.Nullable,
		}

		// Only show default value for nullable fields
		if col.Nullable && col.DefaultValue != nil {
			var defaultVal any = *col.DefaultValue
			fieldSchema.Default = &defaultVal
		}

		schema.Fields = append(schema.Fields, fieldSchema)
	}

	return schema
}

// FromSystemResource generates schema for system resources (users, apikeys)
func (b *Builder) FromSystemResource(resourceName string) *Schema {
	switch resourceName {
	case "users":
		return &Schema{
			Collection: "users",
			Fields: []FieldSchema{
				{Name: "id", Type: "string", Nullable: false},
				{Name: "username", Type: "string", Nullable: false},
				{Name: "email", Type: "string", Nullable: false},
				{Name: "role", Type: "string", Nullable: false},
				{Name: "can_write", Type: "boolean", Nullable: false},
			},
			PrimaryKey: "id",
			Metadata: &Metadata{
				CreatedAt: "datetime",
				UpdatedAt: "datetime",
			},
		}
	case "apikeys":
		return &Schema{
			Collection: "apikeys",
			Fields: []FieldSchema{
				{Name: "id", Type: "string", Nullable: false},
				{Name: "name", Type: "string", Nullable: false},
				{Name: "description", Type: "string", Nullable: true},
				{Name: "role", Type: "string", Nullable: false},
				{Name: "can_write", Type: "boolean", Nullable: false},
			},
			PrimaryKey: "id",
			Metadata: &Metadata{
				CreatedAt: "datetime",
				UpdatedAt: "datetime",
			},
		}
	case "collections":
		return &Schema{
			Collection: "collections",
			Fields: []FieldSchema{
				{Name: "name", Type: "string", Nullable: false},
				{Name: "columns", Type: "json", Nullable: false},
			},
			PrimaryKey: "name",
		}
	default:
		return nil
	}
}
