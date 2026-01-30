// Package openapi provides dynamic OpenAPI/Swagger specification generation.
// It generates API documentation from the in-memory schema registry, ensuring
// the documentation always reflects the current database structure.
package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// OpenAPI represents an OpenAPI 3.0 specification
type OpenAPI struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers,omitempty"`
	Paths      map[string]PathItem   `json:"paths"`
	Components Components            `json:"components,omitempty"`
	Security   []SecurityRequirement `json:"security,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty"`
}

// Info represents the info section of the OpenAPI spec
type Info struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version"`
	Contact     *Contact `json:"contact,omitempty"`
	License     *License `json:"license,omitempty"`
}

// Contact represents contact information
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License represents license information
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server in the OpenAPI spec
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem represents a path item in the OpenAPI spec
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Summary string     `json:"summary,omitempty"`
}

// Operation represents an operation in the OpenAPI spec
type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []SecurityRequirement `json:"security,omitempty"`
}

// Parameter represents a parameter in the OpenAPI spec
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // query, header, path, cookie
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Example     any     `json:"example,omitempty"`
}

// RequestBody represents a request body in the OpenAPI spec
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
	Content     map[string]MediaType `json:"content"`
}

// Response represents a response in the OpenAPI spec
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty"`
}

// MediaType represents a media type in the OpenAPI spec
type MediaType struct {
	Schema  *Schema `json:"schema,omitempty"`
	Example any     `json:"example,omitempty"`
}

// Schema represents a JSON Schema in the OpenAPI spec
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
	Default     any                `json:"default,omitempty"`
	Nullable    bool               `json:"nullable,omitempty"`
	Example     any                `json:"example,omitempty"`
}

// Header represents a header in the OpenAPI spec
type Header struct {
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Components represents the components section of the OpenAPI spec
type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme represents a security scheme in the OpenAPI spec
type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Description  string `json:"description,omitempty"`
}

// SecurityRequirement represents a security requirement
type SecurityRequirement map[string][]string

// Tag represents a tag in the OpenAPI spec
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GeneratorConfig holds configuration for the OpenAPI generator
type GeneratorConfig struct {
	Title       string
	Description string
	Version     string
	Prefix      string
	Servers     []Server
	Contact     *Contact
	License     *License
}

// Generator generates OpenAPI specifications from the schema registry
type Generator struct {
	registry *registry.SchemaRegistry
	config   GeneratorConfig
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(reg *registry.SchemaRegistry, config GeneratorConfig) *Generator {
	if config.Title == "" {
		config.Title = "Moon API"
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.Description == "" {
		config.Description = "Dynamic Headless Engine API"
	}

	return &Generator{
		registry: reg,
		config:   config,
	}
}

// Generate generates the OpenAPI specification
func (g *Generator) Generate() *OpenAPI {
	spec := &OpenAPI{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       g.config.Title,
			Description: g.config.Description,
			Version:     g.config.Version,
			Contact:     g.config.Contact,
			License:     g.config.License,
		},
		Servers: g.config.Servers,
		Paths:   make(map[string]PathItem),
		Components: Components{
			Schemas:         make(map[string]*Schema),
			SecuritySchemes: make(map[string]SecurityScheme),
		},
		Security: []SecurityRequirement{},
		Tags:     []Tag{},
	}

	// Add security schemes
	g.addSecuritySchemes(spec)

	// Add health check endpoints
	g.addHealthEndpoints(spec)

	// Add schema management endpoints
	g.addCollectionEndpoints(spec)

	// Add data endpoints for each collection
	g.addDataEndpoints(spec)

	// Add common schemas
	g.addCommonSchemas(spec)

	return spec
}

// addSecuritySchemes adds the security schemes to the spec
func (g *Generator) addSecuritySchemes(spec *OpenAPI) {
	spec.Components.SecuritySchemes["bearerAuth"] = SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT Bearer token authentication",
	}

	spec.Components.SecuritySchemes["apiKeyAuth"] = SecurityScheme{
		Type:        "apiKey",
		In:          "header",
		Name:        constants.HeaderAPIKey,
		Description: "API Key authentication",
	}
}

// addHealthEndpoints adds the health check endpoints
func (g *Generator) addHealthEndpoints(spec *OpenAPI) {
	spec.Tags = append(spec.Tags, Tag{
		Name:        "Health",
		Description: "Health check endpoints",
	})

	healthPath := g.config.Prefix + "/health"
	spec.Paths[healthPath] = PathItem{
		Get: &Operation{
			Tags:        []string{"Health"},
			Summary:     "Liveness check",
			Description: "Check if the server is running",
			OperationID: "healthCheck",
			Responses: map[string]Response{
				"200": {
					Description: "Server is healthy",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/HealthResponse",
							},
							Example: map[string]any{
								"status":      "healthy",
								"database":    "sqlite",
								"collections": 5,
							},
						},
					},
				},
				"503": {
					Description: "Service unavailable",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
		},
	}

	spec.Paths["/health/ready"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Health"},
			Summary:     "Readiness check",
			Description: "Check if the server is ready to accept requests",
			OperationID: "readinessCheck",
			Responses: map[string]Response{
				"200": {
					Description: "Server is ready",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ReadinessResponse"},
						},
					},
				},
				"503": {
					Description: "Service not ready",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
		},
	}
}

// addCollectionEndpoints adds the collection management endpoints
func (g *Generator) addCollectionEndpoints(spec *OpenAPI) {
	spec.Tags = append(spec.Tags, Tag{
		Name:        "Collections",
		Description: "Schema management endpoints",
	})

	security := []SecurityRequirement{
		{"bearerAuth": {}},
		{"apiKeyAuth": {}},
	}

	// List collections
	spec.Paths[g.config.Prefix+"/collections:list"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Collections"},
			Summary:     "List all collections",
			Description: "Retrieve a list of all managed collections",
			OperationID: "listCollections",
			Security:    security,
			Responses: map[string]Response{
				"200": {
					Description: "List of collections",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/CollectionListResponse"},
							Example: map[string]any{
								"collections": []string{"users", "products"},
								"count":       2,
							},
						},
					},
				},
			},
		},
	}

	// Get collection
	spec.Paths[g.config.Prefix+"/collections:get"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Collections"},
			Summary:     "Get collection schema",
			Description: "Retrieve the schema for a specific collection",
			OperationID: "getCollection",
			Security:    security,
			Parameters: []Parameter{
				{
					Name:        "name",
					In:          "query",
					Description: "Collection name",
					Required:    true,
					Schema:      &Schema{Type: "string"},
					Example:     "users",
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Collection schema",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/CollectionResponse"},
						},
					},
				},
				"404": {
					Description: "Collection not found",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
		},
	}

	// Create collection
	spec.Paths[g.config.Prefix+"/collections:create"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Collections"},
			Summary:     "Create a collection",
			Description: "Create a new collection (database table)",
			OperationID: "createCollection",
			Security:    security,
			RequestBody: &RequestBody{
				Required:    true,
				Description: "Collection definition",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/CreateCollectionRequest"},
						Example: map[string]any{
							"name": "products",
							"columns": []map[string]any{
								{"name": "name", "type": "string", "nullable": false},
								{"name": "price", "type": "float", "nullable": false},
							},
						},
					},
				},
			},
			Responses: map[string]Response{
				"201": {
					Description: "Collection created",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/CreateCollectionResponse"},
						},
					},
				},
				"400": {
					Description: "Invalid request",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
				"409": {
					Description: "Collection already exists",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
		},
	}

	// Update collection
	spec.Paths[g.config.Prefix+"/collections:update"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Collections"},
			Summary:     "Update a collection",
			Description: "Modify a collection's schema (add columns)",
			OperationID: "updateCollection",
			Security:    security,
			RequestBody: &RequestBody{
				Required:    true,
				Description: "Update specification",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/UpdateCollectionRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Collection updated",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/UpdateCollectionResponse"},
						},
					},
				},
				"404": {
					Description: "Collection not found",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
		},
	}

	// Destroy collection
	spec.Paths[g.config.Prefix+"/collections:destroy"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Collections"},
			Summary:     "Destroy a collection",
			Description: "Drop a collection (database table)",
			OperationID: "destroyCollection",
			Security:    security,
			RequestBody: &RequestBody{
				Required:    true,
				Description: "Collection to destroy",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"name": {Type: "string", Description: "Collection name"},
							},
							Required: []string{"name"},
						},
						Example: map[string]any{"name": "products"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Collection destroyed",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]*Schema{
									"message": {Type: "string"},
								},
							},
						},
					},
				},
				"404": {
					Description: "Collection not found",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
		},
	}
}

// addDataEndpoints adds data endpoints for each collection
func (g *Generator) addDataEndpoints(spec *OpenAPI) {
	collections := g.registry.GetAll()

	// Sort collections for consistent output
	sort.Slice(collections, func(i, j int) bool {
		return collections[i].Name < collections[j].Name
	})

	security := []SecurityRequirement{
		{"bearerAuth": {}},
		{"apiKeyAuth": {}},
	}

	for _, collection := range collections {
		// Add tag for collection
		spec.Tags = append(spec.Tags, Tag{
			Name:        collection.Name,
			Description: fmt.Sprintf("Data operations for %s collection", collection.Name),
		})

		// Add schema for collection
		g.addCollectionSchema(spec, collection)

		basePath := fmt.Sprintf("%s/%s", g.config.Prefix, collection.Name)

		// List records
		spec.Paths[basePath+":list"] = PathItem{
			Get: &Operation{
				Tags:        []string{collection.Name},
				Summary:     fmt.Sprintf("List %s records", collection.Name),
				Description: fmt.Sprintf("Retrieve records from %s collection", collection.Name),
				OperationID: fmt.Sprintf("list%s", capitalize(collection.Name)),
				Security:    security,
				Parameters: []Parameter{
					{
						Name:        "limit",
						In:          "query",
						Description: "Maximum number of records to return",
						Schema:      &Schema{Type: "integer", Default: 100},
					},
					{
						Name:        "offset",
						In:          "query",
						Description: "Number of records to skip",
						Schema:      &Schema{Type: "integer", Default: 0},
					},
				},
				Responses: map[string]Response{
					"200": {
						Description: "List of records",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"data": {
											Type:  "array",
											Items: &Schema{Ref: fmt.Sprintf("#/components/schemas/%s", capitalize(collection.Name))},
										},
										"count":  {Type: "integer"},
										"limit":  {Type: "integer"},
										"offset": {Type: "integer"},
									},
								},
							},
						},
					},
				},
			},
		}

		// Get record
		spec.Paths[basePath+":get"] = PathItem{
			Get: &Operation{
				Tags:        []string{collection.Name},
				Summary:     fmt.Sprintf("Get %s record", collection.Name),
				Description: fmt.Sprintf("Retrieve a single record from %s collection", collection.Name),
				OperationID: fmt.Sprintf("get%s", capitalize(collection.Name)),
				Security:    security,
				Parameters: []Parameter{
					{
						Name:        "id",
						In:          "query",
						Description: "Record ID",
						Required:    true,
						Schema:      &Schema{Type: "integer"},
					},
				},
				Responses: map[string]Response{
					"200": {
						Description: "Record data",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"data": {Ref: fmt.Sprintf("#/components/schemas/%s", capitalize(collection.Name))},
									},
								},
							},
						},
					},
					"404": {
						Description: "Record not found",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
							},
						},
					},
				},
			},
		}

		// Create record
		spec.Paths[basePath+":create"] = PathItem{
			Post: &Operation{
				Tags:        []string{collection.Name},
				Summary:     fmt.Sprintf("Create %s record", collection.Name),
				Description: fmt.Sprintf("Create a new record in %s collection", collection.Name),
				OperationID: fmt.Sprintf("create%s", capitalize(collection.Name)),
				Security:    security,
				RequestBody: &RequestBody{
					Required:    true,
					Description: "Record data",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]*Schema{
									"data": {Ref: fmt.Sprintf("#/components/schemas/%sInput", capitalize(collection.Name))},
								},
								Required: []string{"data"},
							},
							Example: g.generateCreateExample(collection),
						},
					},
				},
				Responses: map[string]Response{
					"201": {
						Description: "Record created",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"data":    {Ref: fmt.Sprintf("#/components/schemas/%s", capitalize(collection.Name))},
										"message": {Type: "string"},
									},
								},
							},
						},
					},
					"400": {
						Description: "Validation error",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
							},
						},
					},
				},
			},
		}

		// Update record
		spec.Paths[basePath+":update"] = PathItem{
			Post: &Operation{
				Tags:        []string{collection.Name},
				Summary:     fmt.Sprintf("Update %s record", collection.Name),
				Description: fmt.Sprintf("Update a record in %s collection", collection.Name),
				OperationID: fmt.Sprintf("update%s", capitalize(collection.Name)),
				Security:    security,
				RequestBody: &RequestBody{
					Required:    true,
					Description: "Record data with ID",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]*Schema{
									"id":   {Type: "integer"},
									"data": {Ref: fmt.Sprintf("#/components/schemas/%sInput", capitalize(collection.Name))},
								},
								Required: []string{"id", "data"},
							},
						},
					},
				},
				Responses: map[string]Response{
					"200": {
						Description: "Record updated",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"data":    {Ref: fmt.Sprintf("#/components/schemas/%s", capitalize(collection.Name))},
										"message": {Type: "string"},
									},
								},
							},
						},
					},
					"404": {
						Description: "Record not found",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
							},
						},
					},
				},
			},
		}

		// Destroy record
		spec.Paths[basePath+":destroy"] = PathItem{
			Post: &Operation{
				Tags:        []string{collection.Name},
				Summary:     fmt.Sprintf("Delete %s record", collection.Name),
				Description: fmt.Sprintf("Delete a record from %s collection", collection.Name),
				OperationID: fmt.Sprintf("destroy%s", capitalize(collection.Name)),
				Security:    security,
				RequestBody: &RequestBody{
					Required:    true,
					Description: "Record ID to delete",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]*Schema{
									"id": {Type: "integer"},
								},
								Required: []string{"id"},
							},
						},
					},
				},
				Responses: map[string]Response{
					"200": {
						Description: "Record deleted",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"message": {Type: "string"},
									},
								},
							},
						},
					},
					"404": {
						Description: "Record not found",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
							},
						},
					},
				},
			},
		}
	}
}

// addCollectionSchema adds a schema for a collection
func (g *Generator) addCollectionSchema(spec *OpenAPI, collection *registry.Collection) {
	schemaName := capitalize(collection.Name)

	// Output schema (with ID)
	outputProps := map[string]*Schema{
		"id": {Type: "integer", Description: "Record ID"},
	}

	// Input schema (without ID)
	inputProps := map[string]*Schema{}
	required := []string{}

	for _, col := range collection.Columns {
		propSchema := g.columnToSchema(col)
		outputProps[col.Name] = propSchema
		inputProps[col.Name] = propSchema

		if !col.Nullable && col.DefaultValue == nil {
			required = append(required, col.Name)
		}
	}

	spec.Components.Schemas[schemaName] = &Schema{
		Type:        "object",
		Description: fmt.Sprintf("%s record", collection.Name),
		Properties:  outputProps,
	}

	spec.Components.Schemas[schemaName+"Input"] = &Schema{
		Type:        "object",
		Description: fmt.Sprintf("%s input data", collection.Name),
		Properties:  inputProps,
		Required:    required,
	}
}

// columnToSchema converts a column to an OpenAPI schema
func (g *Generator) columnToSchema(col registry.Column) *Schema {
	schema := &Schema{
		Description: col.Name,
		Nullable:    col.Nullable,
	}

	switch col.Type {
	case registry.TypeString:
		schema.Type = "string"
		schema.Example = "example string"
	case registry.TypeText:
		schema.Type = "string"
		schema.Example = "longer text content"
	case registry.TypeInteger:
		schema.Type = "integer"
		schema.Example = 42
	case registry.TypeFloat:
		schema.Type = "number"
		schema.Format = "double"
		schema.Example = 99.99
	case registry.TypeBoolean:
		schema.Type = "boolean"
		schema.Example = true
	case registry.TypeDatetime:
		schema.Type = "string"
		schema.Format = "date-time"
		schema.Example = "2024-01-15T10:30:00Z"
	case registry.TypeJSON:
		// JSON can be any type (object, array, string, number, boolean, null)
		// Using empty type to allow any JSON value; OpenAPI 3.1 supports this better
		schema.Type = "object"
		schema.Description = col.Name + " (accepts any valid JSON value)"
		schema.Example = map[string]any{"key": "value"}
	default:
		schema.Type = "string"
	}

	if col.DefaultValue != nil {
		schema.Default = *col.DefaultValue
	}

	return schema
}

// generateCreateExample generates an example for create operations
func (g *Generator) generateCreateExample(collection *registry.Collection) map[string]any {
	example := make(map[string]any)

	for _, col := range collection.Columns {
		switch col.Type {
		case registry.TypeString, registry.TypeText:
			example[col.Name] = "example value"
		case registry.TypeInteger:
			example[col.Name] = 42
		case registry.TypeFloat:
			example[col.Name] = 99.99
		case registry.TypeBoolean:
			example[col.Name] = true
		case registry.TypeDatetime:
			example[col.Name] = "2024-01-15T10:30:00Z"
		case registry.TypeJSON:
			example[col.Name] = map[string]any{"key": "value"}
		}
	}

	return map[string]any{"data": example}
}

// addCommonSchemas adds common schemas used across the API
func (g *Generator) addCommonSchemas(spec *OpenAPI) {
	spec.Components.Schemas["ErrorResponse"] = &Schema{
		Type:        "object",
		Description: "Error response",
		Properties: map[string]*Schema{
			"error":      {Type: "string", Description: "Error message"},
			"code":       {Type: "integer", Description: "HTTP status code"},
			"details":    {Type: "object", Description: "Additional error details"},
			"request_id": {Type: "string", Description: "Request correlation ID"},
		},
		Required: []string{"error", "code"},
	}

	spec.Components.Schemas["HealthResponse"] = &Schema{
		Type:        "object",
		Description: "Health check response",
		Properties: map[string]*Schema{
			"status":      {Type: "string", Description: "Health status"},
			"database":    {Type: "string", Description: "Database dialect"},
			"collections": {Type: "integer", Description: "Number of collections"},
		},
	}

	spec.Components.Schemas["ReadinessResponse"] = &Schema{
		Type:        "object",
		Description: "Readiness check response",
		Properties: map[string]*Schema{
			"status": {Type: "string"},
			"checks": {
				Type: "object",
				Properties: map[string]*Schema{
					"database": {Type: "string"},
					"registry": {Type: "string"},
				},
			},
		},
	}

	spec.Components.Schemas["Column"] = &Schema{
		Type:        "object",
		Description: "Column definition",
		Properties: map[string]*Schema{
			"name":          {Type: "string"},
			"type":          {Type: "string", Enum: []string{"string", "integer", "float", "boolean", "datetime", "text", "json"}},
			"nullable":      {Type: "boolean"},
			"unique":        {Type: "boolean"},
			"default_value": {Type: "string", Nullable: true},
		},
		Required: []string{"name", "type"},
	}

	spec.Components.Schemas["CollectionListResponse"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"collections": {Type: "array", Items: &Schema{Type: "string"}},
			"count":       {Type: "integer"},
		},
	}

	spec.Components.Schemas["CollectionResponse"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"collection": {
				Type: "object",
				Properties: map[string]*Schema{
					"name":    {Type: "string"},
					"columns": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Column"}},
				},
			},
		},
	}

	spec.Components.Schemas["CreateCollectionRequest"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"name":    {Type: "string", Description: "Collection name"},
			"columns": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Column"}},
		},
		Required: []string{"name", "columns"},
	}

	spec.Components.Schemas["CreateCollectionResponse"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"collection": {Ref: "#/components/schemas/CollectionResponse"},
			"message":    {Type: "string"},
		},
	}

	spec.Components.Schemas["UpdateCollectionRequest"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"name":        {Type: "string", Description: "Collection name"},
			"add_columns": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Column"}},
		},
		Required: []string{"name"},
	}

	spec.Components.Schemas["UpdateCollectionResponse"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"collection": {Ref: "#/components/schemas/CollectionResponse"},
			"message":    {Type: "string"},
		},
	}
}

// ToJSON converts the OpenAPI spec to JSON
func (spec *OpenAPI) ToJSON() ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

// Handler serves the OpenAPI specification as JSON
func (g *Generator) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spec := g.Generate()

		w.Header().Set(constants.HeaderContentType, constants.MIMEApplicationJSON)
		w.Header().Set("Access-Control-Allow-Origin", "*")

		data, err := spec.ToJSON()
		if err != nil {
			http.Error(w, "Failed to generate OpenAPI spec", http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(data); err != nil {
			// Log error but can't send HTTP error as headers already sent
			// This is acceptable as the write failure will be detected by the HTTP layer
		}
	}
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
