// Package handlers provides HTTP request handlers for the Moon API.
// This file implements documentation generation endpoints.
package handlers

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/registry"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed templates/doc.md.tmpl
var docTemplateContent string

// DocData holds the data passed to the documentation template
type DocData struct {
	ServiceName   string
	Version       string
	BaseURL       string
	Prefix        string
	JWTEnabled    bool
	APIKeyEnabled bool
	APIKeyHeader  string
	Collections   []string
	JSONAppendix  string
}

// DocHandler handles documentation endpoints
type DocHandler struct {
	registry     *registry.SchemaRegistry
	config       *config.AppConfig
	version      string
	cacheMutex   sync.RWMutex
	htmlCache    []byte
	mdCache      []byte
	htmlETag     string
	mdETag       string
	lastModified time.Time
	mdTemplate   *template.Template
	mdConverter  goldmark.Markdown
}

// NewDocHandler creates a new documentation handler
func NewDocHandler(reg *registry.SchemaRegistry, cfg *config.AppConfig, version string) *DocHandler {
	// Parse the markdown template
	tmpl, err := template.New("doc").Parse(docTemplateContent)
	if err != nil {
		log.Printf("ERROR: Failed to parse documentation template: %v", err)
		// Create an empty template as fallback
		tmpl = template.Must(template.New("doc").Parse("# Documentation Error\n\nFailed to load template."))
	}

	// Configure goldmark for markdown to HTML conversion
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,   // GitHub Flavored Markdown
			extension.Table, // Tables
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Auto-generate heading IDs for anchors
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Convert line breaks to <br>
			html.WithXHTML(),     // Use XHTML syntax
		),
	)

	return &DocHandler{
		registry:     reg,
		config:       cfg,
		version:      version,
		lastModified: time.Now(),
		mdTemplate:   tmpl,
		mdConverter:  md,
	}
}

// HTML serves the HTML documentation
func (h *DocHandler) HTML(w http.ResponseWriter, r *http.Request) {
	h.cacheMutex.RLock()
	cached := h.htmlCache
	etag := h.htmlETag
	h.cacheMutex.RUnlock()

	// Generate if not cached
	if cached == nil {
		h.cacheMutex.Lock()
		// Double-check after acquiring write lock
		if h.htmlCache == nil {
			html, err := h.generateHTML()
			if err != nil {
				log.Printf("ERROR: Failed to generate HTML documentation: %v", err)
				http.Error(w, "Failed to generate documentation", http.StatusInternalServerError)
				h.cacheMutex.Unlock()
				return
			}
			h.htmlCache = []byte(html)
			h.htmlETag = fmt.Sprintf(`"html-%d"`, time.Now().Unix())
		}
		cached = h.htmlCache
		etag = h.htmlETag
		h.cacheMutex.Unlock()
	}

	// Set cache headers
	w.Header().Set(constants.HeaderContentType, "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", h.lastModified.UTC().Format(http.TimeFormat))

	// Check If-None-Match header
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(cached)
}

// Markdown serves the Markdown documentation
func (h *DocHandler) Markdown(w http.ResponseWriter, r *http.Request) {
	h.cacheMutex.RLock()
	cached := h.mdCache
	etag := h.mdETag
	h.cacheMutex.RUnlock()

	// Generate if not cached
	if cached == nil {
		h.cacheMutex.Lock()
		// Double-check after acquiring write lock
		if h.mdCache == nil {
			md, err := h.generateMarkdown()
			if err != nil {
				log.Printf("ERROR: Failed to generate Markdown documentation: %v", err)
				http.Error(w, "Failed to generate documentation", http.StatusInternalServerError)
				h.cacheMutex.Unlock()
				return
			}
			h.mdCache = []byte(md)
			h.mdETag = fmt.Sprintf(`"md-%d"`, time.Now().Unix())
		}
		cached = h.mdCache
		etag = h.mdETag
		h.cacheMutex.Unlock()
	}

	// Set cache headers
	w.Header().Set(constants.HeaderContentType, "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", h.lastModified.UTC().Format(http.TimeFormat))

	// Check If-None-Match header
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(cached)
}

// RefreshCache clears the cached documentation
func (h *DocHandler) RefreshCache(w http.ResponseWriter, r *http.Request) {
	h.cacheMutex.Lock()
	h.htmlCache = nil
	h.mdCache = nil
	h.htmlETag = ""
	h.mdETag = ""
	h.lastModified = time.Now()
	h.cacheMutex.Unlock()

	w.Header().Set(constants.HeaderContentType, constants.MIMEApplicationJSON)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Documentation cache refreshed"}`))
}

// generateMarkdown generates the Markdown documentation from the template
func (h *DocHandler) generateMarkdown() (string, error) {
	data := h.buildDocData()

	var buf bytes.Buffer
	if err := h.mdTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// generateHTML generates the HTML documentation by converting rendered Markdown
func (h *DocHandler) generateHTML() (string, error) {
	// First generate the Markdown
	markdownContent, err := h.generateMarkdown()
	if err != nil {
		return "", fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Convert Markdown to HTML
	var htmlBody bytes.Buffer
	if err := h.mdConverter.Convert([]byte(markdownContent), &htmlBody); err != nil {
		return "", fmt.Errorf("failed to convert markdown to HTML: %w", err)
	}

	// Wrap HTML body with full HTML structure and CSS
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Moon API Documentation</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; max-width: 1200px; margin: 0 auto; padding: 20px; color: #333; }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; border-bottom: 2px solid #ecf0f1; padding-bottom: 5px; }
        h3 { color: #555; margin-top: 20px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-family: "Courier New", monospace; }
        pre { background: #2c3e50; color: #ecf0f1; padding: 15px; border-radius: 5px; overflow-x: auto; position: relative; padding-right: 80px; }
        pre code { background: none; color: inherit; padding: 0; }
        table { border-collapse: collapse; width: 100%; margin: 15px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background: #3498db; color: white; }
        tr:nth-child(even) { background: #f9f9f9; }
        a { color: #2980b9; text-decoration: none; }
        a:hover { text-decoration: underline; }
        ul { padding-left: 20px; }
        li { margin: 8px 0; }
        .copy-btn { position: absolute; top: 8px; right: 8px; padding: 6px 12px; font-size: 0.85em; font-weight: 500; background: #34495e; color: #ecf0f1; border: 1px solid #2c3e50; border-radius: 4px; cursor: pointer; transition: all 0.2s ease; z-index: 10; }
        .copy-btn:hover { background: #2c3e50; border-color: #1a252f; }
        .copy-btn.copied { background: #27ae60; border-color: #229954; }
    </style>
</head>
<body>
`)

	// Add the converted HTML body
	sb.WriteString(htmlBody.String())

	// Add copy button JavaScript
	sb.WriteString(`<script>
document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('pre > code').forEach(function(codeBlock) {
        var pre = codeBlock.parentNode;
        var button = document.createElement('button');
        button.innerText = 'Copy';
        button.className = 'copy-btn';
        
        pre.style.position = 'relative';
        pre.appendChild(button);
        
        button.addEventListener('click', function() {
            var text = codeBlock.innerText;
            navigator.clipboard.writeText(text).then(function() {
                button.innerText = 'Copied!';
                button.classList.add('copied');
                setTimeout(function() {
                    button.innerText = 'Copy';
                    button.classList.remove('copied');
                }, 1200);
            }).catch(function(err) {
                console.error('Failed to copy text: ', err);
            });
        });
    });
});
</script>
</body>
</html>`)

	return sb.String(), nil
}

// buildDocData constructs the data structure for the template
func (h *DocHandler) buildDocData() DocData {
	collections := h.getCollectionNames()
	baseURL := fmt.Sprintf("http://localhost:%d", h.config.Server.Port)

	return DocData{
		ServiceName:   "moon",
		Version:       h.version,
		BaseURL:       baseURL,
		Prefix:        h.config.Server.Prefix,
		JWTEnabled:    h.config.JWT.Secret != "",
		APIKeyEnabled: h.config.APIKey.Enabled,
		APIKeyHeader:  h.config.APIKey.Header,
		Collections:   collections,
		JSONAppendix:  h.buildJSONAppendix(),
	}
}

// getCollectionNames returns a sorted list of collection names
func (h *DocHandler) getCollectionNames() []string {
	collections := h.registry.GetAll()
	names := make([]string, 0, len(collections))
	for _, c := range collections {
		names = append(names, c.Name)
	}
	return names
}

// JSONAppendixData represents the structure of the JSON Appendix
type JSONAppendixData struct {
	Service         string             `json:"service"`
	Version         string             `json:"version"`
	DocumentVersion string             `json:"document_version"`
	BaseURL         string             `json:"base_url"`
	URLPrefix       *string            `json:"url_prefix"`
	Authentication  AuthInfo           `json:"authentication"`
	Collections     CollectionsInfo    `json:"collections"`
	DataTypes       []DataTypeInfo     `json:"data_types"`
	RegisteredColls []CollectionDetail `json:"registered_collections"`
	Endpoints       map[string]any     `json:"endpoints"`
	Query           map[string]any     `json:"query"`
	Aggregation     map[string]any     `json:"aggregation"`
	HTTPStatusCodes map[string]string  `json:"http_status_codes"`
	RateLimiting    map[string]any     `json:"rate_limiting"`
	CORS            map[string]any     `json:"cors"`
	Guarantees      map[string]bool    `json:"guarantees"`
	AIPStandards    map[string]string  `json:"aip_standards"`
}

// AuthInfo holds authentication configuration
type AuthInfo struct {
	Modes      []string          `json:"modes"`
	Headers    map[string]string `json:"headers"`
	RateLimits map[string]string `json:"rate_limits"`
	Rules      map[string]string `json:"rules"`
}

// CollectionsInfo holds collection metadata and constraints
type CollectionsInfo struct {
	Terminology map[string]string `json:"terminology"`
	Naming      map[string]any    `json:"naming"`
	Constraints map[string]bool   `json:"constraints"`
}

// DataTypeInfo describes a supported data type
type DataTypeInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SQLMapping  string `json:"sql_mapping"`
	Example     any    `json:"example"`
	Format      string `json:"format,omitempty"`
	Note        string `json:"note,omitempty"`
}

// CollectionDetail represents a registered collection with its fields
type CollectionDetail struct {
	Name   string      `json:"name"`
	Fields []FieldInfo `json:"fields"`
}

// FieldInfo represents a field within a collection
type FieldInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// buildJSONAppendix generates a dynamic JSON appendix from the registry and config
func (h *DocHandler) buildJSONAppendix() string {
	// Prepare authentication modes
	authModes := []string{}
	authHeaders := map[string]string{}
	if h.config.JWT.Secret != "" {
		authModes = append(authModes, "jwt")
		authHeaders["jwt"] = "Authorization: Bearer <token>"
	}
	if h.config.APIKey.Enabled {
		authModes = append(authModes, "api_key")
		authHeaders["api_key"] = fmt.Sprintf("%s: <key>", h.config.APIKey.Header)
	}

	// Prepare URL prefix (null if empty)
	var urlPrefix *string
	if h.config.Server.Prefix != "" {
		urlPrefix = &h.config.Server.Prefix
	}

	// Get registered collections with sorted fields
	collections := h.registry.GetAll()
	registeredColls := make([]CollectionDetail, 0, len(collections))

	// Sort collections by name for deterministic output
	sort.Slice(collections, func(i, j int) bool {
		return collections[i].Name < collections[j].Name
	})

	for _, coll := range collections {
		fields := make([]FieldInfo, 0, len(coll.Columns))

		// Sort fields by name for deterministic output
		sortedColumns := make([]registry.Column, len(coll.Columns))
		copy(sortedColumns, coll.Columns)
		sort.Slice(sortedColumns, func(i, j int) bool {
			return sortedColumns[i].Name < sortedColumns[j].Name
		})

		for _, col := range sortedColumns {
			fields = append(fields, FieldInfo{
				Name:     col.Name,
				Type:     string(col.Type),
				Nullable: col.Nullable,
			})
		}

		registeredColls = append(registeredColls, CollectionDetail{
			Name:   coll.Name,
			Fields: fields,
		})
	}

	// Build the appendix data structure
	appendix := JSONAppendixData{
		Service:         "moon",
		Version:         h.version,
		DocumentVersion: "1.5.1",
		BaseURL:         fmt.Sprintf("http://localhost:%d", h.config.Server.Port),
		URLPrefix:       urlPrefix,
		Authentication: AuthInfo{
			Modes:   authModes,
			Headers: authHeaders,
			RateLimits: map[string]string{
				"jwt":     "100 requests per minute per user",
				"api_key": "1000 requests per minute per key",
			},
			Rules: map[string]string{
				"jwt_for":     "user-facing apps with session management",
				"api_key_for": "server-to-server or backend services",
			},
		},
		Collections: CollectionsInfo{
			Terminology: map[string]string{
				"collection": "table/database collection",
				"field":      "column/table column",
				"record":     "row/table row",
			},
			Naming: map[string]any{
				"case":      "snake_case",
				"lowercase": true,
				"pattern":   "^[a-z][a-z0-9_]*$",
			},
			Constraints: map[string]bool{
				"joins_supported": false,
				"foreign_keys":    false,
				"transactions":    false,
				"triggers":        false,
				"background_jobs": false,
			},
		},
		DataTypes: []DataTypeInfo{
			{Name: "string", Description: "Text values of any length", SQLMapping: "TEXT", Example: "Wireless Mouse"},
			{Name: "integer", Description: "64-bit whole numbers", SQLMapping: "INTEGER", Example: 42},
			{Name: "boolean", Description: "true/false values", SQLMapping: "BOOLEAN", Example: true},
			{Name: "datetime", Description: "Date/time in RFC3339 or ISO 8601 format", SQLMapping: "DATETIME", Example: "2023-01-31T13:45:00Z"},
			{Name: "json", Description: "Arbitrary JSON object or array", SQLMapping: "JSON", Example: map[string]string{"key": "value"}},
			{Name: "decimal", Description: "Decimal values with precision", SQLMapping: "DECIMAL", Format: "string", Example: "199.99", Note: "API input/output uses strings, default 2 decimal places"},
		},
		RegisteredColls: registeredColls,
		Endpoints: map[string]any{
			"health": map[string]any{
				"path":          "/health",
				"method":        "GET",
				"auth_required": false,
				"description":   "Health check endpoint",
			},
			"authentication": map[string]any{
				"login": map[string]any{
					"path":          "/auth:login",
					"method":        "POST",
					"auth_required": false,
					"description":   "Authenticate user, receive JWT tokens",
				},
				"logout": map[string]any{
					"path":          "/auth:logout",
					"method":        "POST",
					"auth_required": true,
					"description":   "Invalidate current session's refresh token",
				},
				"refresh": map[string]any{
					"path":          "/auth:refresh",
					"method":        "POST",
					"auth_required": false,
					"description":   "Exchange refresh token for new access token",
				},
				"me": map[string]any{
					"path":          "/auth:me",
					"methods":       []string{"GET", "POST"},
					"auth_required": true,
					"description":   "Get or update current user profile",
				},
			},
			"user_management": map[string]any{
				"list": map[string]any{
					"path":          "/users:list",
					"method":        "GET",
					"auth_required": true,
					"role_required": "admin",
				},
				"get": map[string]any{
					"path":          "/users:get",
					"method":        "GET",
					"auth_required": true,
					"role_required": "admin",
					"params":        []string{"id"},
				},
				"create": map[string]any{
					"path":          "/users:create",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
				},
				"update": map[string]any{
					"path":          "/users:update",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
					"params":        []string{"id"},
					"actions":       []string{"reset_password", "revoke_sessions"},
				},
				"destroy": map[string]any{
					"path":          "/users:destroy",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
					"params":        []string{"id"},
				},
			},
			"apikey_management": map[string]any{
				"list": map[string]any{
					"path":          "/apikeys:list",
					"method":        "GET",
					"auth_required": true,
					"role_required": "admin",
				},
				"get": map[string]any{
					"path":          "/apikeys:get",
					"method":        "GET",
					"auth_required": true,
					"role_required": "admin",
					"params":        []string{"id"},
				},
				"create": map[string]any{
					"path":          "/apikeys:create",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
				},
				"update": map[string]any{
					"path":          "/apikeys:update",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
					"params":        []string{"id"},
					"actions":       []string{"rotate"},
				},
				"destroy": map[string]any{
					"path":          "/apikeys:destroy",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
					"params":        []string{"id"},
				},
			},
			"collection_management": map[string]any{
				"list": map[string]any{
					"path":          "/collections:list",
					"method":        "GET",
					"auth_required": true,
					"role_required": "admin",
				},
				"get": map[string]any{
					"path":          "/collections:get",
					"method":        "GET",
					"auth_required": true,
					"role_required": "admin",
					"params":        []string{"name"},
				},
				"create": map[string]any{
					"path":          "/collections:create",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
				},
				"update": map[string]any{
					"path":          "/collections:update",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
					"operations":    []string{"add_columns", "rename_columns", "modify_columns", "remove_columns"},
				},
				"destroy": map[string]any{
					"path":          "/collections:destroy",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
				},
			},
			"data_access": map[string]any{
				"list": map[string]any{
					"path":          "/{collection}:list",
					"method":        "GET",
					"auth_required": true,
					"description":   "List records in collection",
				},
				"get": map[string]any{
					"path":          "/{collection}:get",
					"method":        "GET",
					"auth_required": true,
					"params":        []string{"id"},
					"description":   "Get single record by ID",
				},
				"create": map[string]any{
					"path":          "/{collection}:create",
					"method":        "POST",
					"auth_required": true,
					"description":   "Create new record",
				},
				"update": map[string]any{
					"path":          "/{collection}:update",
					"method":        "POST",
					"auth_required": true,
					"description":   "Update existing record",
				},
				"destroy": map[string]any{
					"path":          "/{collection}:destroy",
					"method":        "POST",
					"auth_required": true,
					"description":   "Delete record",
				},
			},
			"aggregation": map[string]any{
				"count": map[string]any{
					"path":          "/{collection}:count",
					"method":        "GET",
					"auth_required": true,
					"description":   "Count records",
				},
				"sum": map[string]any{
					"path":          "/{collection}:sum",
					"method":        "GET",
					"auth_required": true,
					"params":        []string{"field"},
					"description":   "Sum numeric field",
				},
				"avg": map[string]any{
					"path":          "/{collection}:avg",
					"method":        "GET",
					"auth_required": true,
					"params":        []string{"field"},
					"description":   "Average numeric field",
				},
				"min": map[string]any{
					"path":          "/{collection}:min",
					"method":        "GET",
					"auth_required": true,
					"params":        []string{"field"},
					"description":   "Minimum value",
				},
				"max": map[string]any{
					"path":          "/{collection}:max",
					"method":        "GET",
					"auth_required": true,
					"params":        []string{"field"},
					"description":   "Maximum value",
				},
			},
			"documentation": map[string]any{
				"html": map[string]any{
					"path":          "/doc/",
					"method":        "GET",
					"auth_required": false,
					"description":   "HTML documentation",
				},
				"markdown": map[string]any{
					"path":          "/doc/md",
					"method":        "GET",
					"auth_required": false,
					"description":   "Markdown documentation",
				},
				"refresh": map[string]any{
					"path":          "/doc:refresh",
					"method":        "POST",
					"auth_required": true,
					"role_required": "admin",
					"description":   "Refresh documentation cache",
				},
			},
		},
		Query: map[string]any{
			"operators": []string{"eq", "ne", "gt", "lt", "gte", "lte", "like", "in"},
			"syntax": map[string]any{
				"filter": "?column[operator]=value",
				"examples": []string{
					"?price[gte]=100",
					"?category[eq]=electronics",
					"?name[like]=%mouse%",
				},
			},
			"sorting": map[string]any{
				"syntax":     "?sort={field1,-field2}",
				"ascending":  "field",
				"descending": "-field",
				"example":    "?sort=-price,name",
			},
			"pagination": map[string]any{
				"cursor_param": "after",
				"limit_param":  "limit",
				"example":      "?limit=10&after=01ARZ3NDEKTSV4RRFFQ69G5FBX",
			},
			"search": map[string]any{
				"full_text_param": "q",
				"description":     "Searches across all text/string columns",
				"example":         "?q=wireless",
			},
			"field_selection": map[string]any{
				"param":       "fields",
				"description": "Return only specified fields (id always included)",
				"example":     "?fields=name,price",
			},
		},
		Aggregation: map[string]any{
			"supported":     []string{"count", "sum", "avg", "min", "max"},
			"numeric_types": []string{"integer", "decimal"},
			"note":          "Aggregation functions work on integer and decimal field types only",
		},
		HTTPStatusCodes: map[string]string{
			"200": "OK - Successful GET request",
			"201": "Created - Successful POST request creating resource",
			"400": "Bad Request - Invalid input or parameters",
			"401": "Unauthorized - Missing or invalid authentication",
			"403": "Forbidden - Insufficient permissions",
			"404": "Not Found - Resource not found",
			"409": "Conflict - Resource already exists",
			"429": "Too Many Requests - Rate limit exceeded",
			"500": "Internal Server Error - Server error",
		},
		RateLimiting: map[string]any{
			"headers": map[string]string{
				"limit":     "X-RateLimit-Limit",
				"remaining": "X-RateLimit-Remaining",
				"reset":     "X-RateLimit-Reset",
			},
		},
		CORS: map[string]any{
			"allowed_methods": []string{"GET", "POST", "OPTIONS"},
			"allowed_headers": []string{"Authorization", "Content-Type", "X-API-Key"},
			"configurable":    true,
			"config_file":     "samples/moon.conf",
			"config_key":      "cors.allowed_origins",
		},
		Guarantees: map[string]bool{
			"transactions":    false,
			"joins":           false,
			"foreign_keys":    false,
			"triggers":        false,
			"background_jobs": false,
		},
		AIPStandards: map[string]string{
			"custom_actions": "AIP-136",
			"pattern":        "resource:action",
			"separator":      ":",
			"description":    "APIs use colon separator between resource and action for predictable interface",
		},
	}

	// Marshal to pretty JSON
	jsonBytes, err := json.MarshalIndent(appendix, "", "  ")
	if err != nil {
		log.Printf("ERROR: Failed to marshal JSON appendix: %v", err)
		return `{
  "error": "Failed to generate JSON appendix",
  "service": "moon"
}`
	}

	return string(jsonBytes)
}
