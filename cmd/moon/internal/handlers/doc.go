// Package handlers provides HTTP request handlers for the Moon API.
// This file implements documentation generation endpoints.
package handlers

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

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
}

// NewDocHandler creates a new documentation handler
func NewDocHandler(reg *registry.SchemaRegistry, cfg *config.AppConfig, version string) *DocHandler {
	return &DocHandler{
		registry:     reg,
		config:       cfg,
		version:      version,
		lastModified: time.Now(),
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
			h.htmlCache = []byte(h.generateHTML())
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
			h.mdCache = []byte(h.generateMarkdown())
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

// generateHTML generates the HTML documentation
func (h *DocHandler) generateHTML() string {
	prefix := h.config.Server.Prefix
	baseURL := fmt.Sprintf("http://localhost:%d", h.config.Server.Port)
	collections := h.getCollectionNames()

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
        pre { background: #2c3e50; color: #ecf0f1; padding: 15px; border-radius: 5px; overflow-x: auto; }
        pre code { background: none; color: inherit; padding: 0; }
        table { border-collapse: collapse; width: 100%; margin: 15px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background: #3498db; color: white; }
        tr:nth-child(even) { background: #f9f9f9; }
        .toc { background: #ecf0f1; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .toc ul { list-style: none; padding-left: 0; }
        .toc li { margin: 8px 0; }
        .toc a { text-decoration: none; color: #2980b9; }
        .toc a:hover { text-decoration: underline; }
        .badge { display: inline-block; padding: 3px 8px; border-radius: 3px; font-size: 0.85em; font-weight: bold; margin-right: 5px; }
        .badge-get { background: #3498db; color: white; }
        .badge-post { background: #2ecc71; color: white; }
        .error-example { background: #fee; border-left: 4px solid #e74c3c; padding: 10px; margin: 10px 0; }
    </style>
</head>
<body>
`)

	sb.WriteString(fmt.Sprintf("<h1>Moon API Documentation</h1>\n"))
	sb.WriteString(fmt.Sprintf("<p><strong>Version:</strong> %s</p>\n", html.EscapeString(h.version)))
	sb.WriteString(fmt.Sprintf("<p><strong>Service:</strong> moon</p>\n"))

	// Table of Contents
	sb.WriteString(`<div class="toc">
<h2>Table of Contents</h2>
<ul>
    <li><a href="#overview">Overview</a></li>
    <li><a href="#authentication">Authentication</a></li>
    <li><a href="#base-url">Base URL and Prefix</a></li>
    <li><a href="#quickstart">Quickstart</a></li>
    <li><a href="#collections">Available Collections</a></li>
    <li><a href="#schema">Schema Management</a></li>
    <li><a href="#data">Data Access</a></li>
    <li><a href="#aggregation">Aggregation Operations</a></li>
    <li><a href="#filtering">Filtering, Sorting & Pagination</a></li>
    <li><a href="#errors">Error Responses</a></li>
    <li><a href="#examples">Example Requests</a></li>
</ul>
</div>
`)

	// Overview
	sb.WriteString(`<h2 id="overview">Overview</h2>
<p>Moon is a dynamic headless engine that provides a RESTful API for managing database tables and records without manual migrations. All endpoints follow the AIP-136 custom actions pattern using a colon separator between the resource and action.</p>
`)

	// Authentication
	sb.WriteString(`<h2 id="authentication">Authentication</h2>
`)
	if h.config.JWT.Secret != "" || h.config.APIKey.Enabled {
		sb.WriteString("<p>This API requires authentication for most endpoints.</p>\n")
		if h.config.JWT.Secret != "" {
			sb.WriteString("<p><strong>JWT Authentication:</strong> Include a bearer token in the Authorization header:</p>\n")
			sb.WriteString("<pre><code>Authorization: Bearer &lt;your-jwt-token&gt;</code></pre>\n")
		}
		if h.config.APIKey.Enabled {
			sb.WriteString(fmt.Sprintf("<p><strong>API Key Authentication:</strong> Include your API key in the %s header:</p>\n", html.EscapeString(h.config.APIKey.Header)))
			sb.WriteString(fmt.Sprintf("<pre><code>%s: &lt;your-api-key&gt;</code></pre>\n", html.EscapeString(h.config.APIKey.Header)))
		}
	} else {
		sb.WriteString("<p>Authentication is currently disabled for this instance.</p>\n")
	}

	// Base URL and Prefix
	sb.WriteString(`<h2 id="base-url">Base URL and Prefix</h2>
`)
	sb.WriteString(fmt.Sprintf("<p><strong>Base URL:</strong> <code>%s</code></p>\n", html.EscapeString(baseURL)))
	if prefix != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>URL Prefix:</strong> <code>%s</code></p>\n", html.EscapeString(prefix)))
		sb.WriteString(fmt.Sprintf("<p>All endpoints are prefixed with <code>%s</code>. For example:</p>\n", html.EscapeString(prefix)))
		sb.WriteString(fmt.Sprintf("<pre><code>%s%s/health\n%s%s/collections:list</code></pre>\n",
			html.EscapeString(baseURL), html.EscapeString(prefix),
			html.EscapeString(baseURL), html.EscapeString(prefix)))
	} else {
		sb.WriteString("<p><strong>URL Prefix:</strong> None (endpoints start at root)</p>\n")
		sb.WriteString(fmt.Sprintf("<pre><code>%s/health\n%s/collections:list</code></pre>\n",
			html.EscapeString(baseURL), html.EscapeString(baseURL)))
	}

	// Quickstart
	sb.WriteString(`<h2 id="quickstart">Quickstart</h2>
<p>Get started with Moon in 5 simple steps:</p>
<ol>
<li><strong>Create a collection</strong>
<pre><code>curl -X POST `)
	sb.WriteString(fmt.Sprintf(`"%s%s/collections:create" \
  -H "Content-Type: application/json" \
  -d '{"name":"products","columns":[{"name":"title","type":"string","nullable":false},{"name":"price","type":"float","nullable":false}]}'`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
</li>
<li><strong>Insert a record</strong>
<pre><code>curl -X POST `)
	sb.WriteString(fmt.Sprintf(`"%s%s/{collection}:create" \
  -H "Content-Type: application/json" \
  -d '{"data":{"title":"Laptop","price":999.99}}'`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
</li>
<li><strong>List records</strong>
<pre><code>curl -s `)
	sb.WriteString(fmt.Sprintf(`"%s%s/{collection}:list" | jq .`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
</li>
<li><strong>Update a record</strong>
<pre><code>curl -X POST `)
	sb.WriteString(fmt.Sprintf(`"%s%s/{collection}:update" \
  -H "Content-Type: application/json" \
  -d '{"id":"01ARZ3NDEKTSV4RRFFQ69G5FBX","data":{"price":899.99}}'`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
</li>
<li><strong>Delete a record</strong>
<pre><code>curl -X POST `)
	sb.WriteString(fmt.Sprintf(`"%s%s/{collection}:destroy" \
  -H "Content-Type: application/json" \
  -d '{"id":"01ARZ3NDEKTSV4RRFFQ69G5FBX"}'`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
</li>
</ol>
`)

	// Available Collections
	sb.WriteString(`<h2 id="collections">Available Collections</h2>
`)
	if len(collections) > 0 {
		sb.WriteString("<p>The following collections are currently available:</p>\n<ul>\n")
		for _, name := range collections {
			sb.WriteString(fmt.Sprintf("<li><code>%s</code></li>\n", html.EscapeString(name)))
		}
		sb.WriteString("</ul>\n")
	} else {
		sb.WriteString("<p>No collections have been created yet.</p>\n")
	}

	// Schema Management
	sb.WriteString(`<h2 id="schema">Schema Management Endpoints</h2>
<p>These endpoints manage database tables (collections) and their schemas.</p>
<table>
<thead><tr><th>Endpoint</th><th>Method</th><th>Description</th></tr></thead>
<tbody>
`)
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/collections:list</code></td><td><span class=\"badge badge-get\">GET</span></td><td>List all collections</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/collections:get</code></td><td><span class=\"badge badge-get\">GET</span></td><td>Get collection schema (requires <code>?name=...</code>)</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/collections:create</code></td><td><span class=\"badge badge-post\">POST</span></td><td>Create a new collection</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/collections:update</code></td><td><span class=\"badge badge-post\">POST</span></td><td>Update collection schema</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/collections:destroy</code></td><td><span class=\"badge badge-post\">POST</span></td><td>Delete a collection</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString("</tbody>\n</table>\n")

	// Data Access
	sb.WriteString(`<h2 id="data">Data Access Endpoints</h2>
<p>These endpoints manage records within a specific collection. Replace <code>{collection}</code> with your collection name.</p>
<table>
<thead><tr><th>Endpoint</th><th>Method</th><th>Description</th></tr></thead>
<tbody>
`)
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:list</code></td><td><span class=\"badge badge-get\">GET</span></td><td>List all records</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:get</code></td><td><span class=\"badge badge-get\">GET</span></td><td>Get a single record (requires <code>?id=...</code>)</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:create</code></td><td><span class=\"badge badge-post\">POST</span></td><td>Create a new record</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:update</code></td><td><span class=\"badge badge-post\">POST</span></td><td>Update an existing record</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:destroy</code></td><td><span class=\"badge badge-post\">POST</span></td><td>Delete a record</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString("</tbody>\n</table>\n")

	// Aggregation
	sb.WriteString(`<h2 id="aggregation">Aggregation Operations</h2>
<p>Server-side aggregation endpoints for analytics. Replace <code>{collection}</code> with your collection name.</p>
<table>
<thead><tr><th>Endpoint</th><th>Method</th><th>Description</th></tr></thead>
<tbody>
`)
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:count</code></td><td><span class=\"badge badge-get\">GET</span></td><td>Count records</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:sum</code></td><td><span class=\"badge badge-get\">GET</span></td><td>Sum numeric field (requires <code>?field=...</code>)</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:avg</code></td><td><span class=\"badge badge-get\">GET</span></td><td>Average numeric field (requires <code>?field=...</code>)</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:min</code></td><td><span class=\"badge badge-get\">GET</span></td><td>Minimum value (requires <code>?field=...</code>)</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString(fmt.Sprintf("<tr><td><code>%s/{collection}:max</code></td><td><span class=\"badge badge-get\">GET</span></td><td>Maximum value (requires <code>?field=...</code>)</td></tr>\n", html.EscapeString(prefix)))
	sb.WriteString("</tbody>\n</table>\n")

	// Filtering, Sorting & Pagination
	sb.WriteString(`<h2 id="filtering">Filtering, Sorting & Pagination</h2>
<h3>Filtering</h3>
<p>Use query parameters to filter results:</p>
<pre><code>?column[operator]=value</code></pre>
<p><strong>Operators:</strong> <code>eq</code>, <code>ne</code>, <code>gt</code>, <code>lt</code>, <code>gte</code>, <code>lte</code>, <code>like</code>, <code>in</code></p>
<p><strong>Example:</strong></p>
<pre><code>`)
	sb.WriteString(fmt.Sprintf(`%s%s/{collection}:list?price[gt]=100&category[eq]=electronics`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>

<h3>Sorting</h3>
<p>Sort by field (ascending) or <code>-field</code> (descending):</p>
<pre><code>?sort=-price,name</code></pre>

<h3>Full-Text Search</h3>
<p>Search across all text columns:</p>
<pre><code>?q=laptop</code></pre>

<h3>Field Selection</h3>
<p>Return only specific fields (always includes id):</p>
<pre><code>?fields=name,price</code></pre>

<h3>Pagination</h3>
<p>Use cursor-based pagination:</p>
<pre><code>?after=01ARZ3NDEKTSV4RRFFQ69G5FBX&limit=10</code></pre>
<p>Response includes <code>next_cursor</code> when more results are available.</p>
`)

	// Error Responses
	sb.WriteString(`<h2 id="errors">Error Responses</h2>
<p>All errors follow a consistent JSON structure:</p>
<div class="error-example">
<pre><code>{
  "error": "Error message describing what went wrong",
  "code": 400
}</code></pre>
</div>
<p><strong>Common Status Codes:</strong></p>
<ul>
<li><code>400 Bad Request</code> - Invalid input, missing required field, invalid filter operator</li>
<li><code>404 Not Found</code> - Collection or record not found</li>
<li><code>409 Conflict</code> - Collection already exists</li>
<li><code>500 Internal Server Error</code> - Server error</li>
</ul>
`)

	// Example Requests
	sb.WriteString(`<h2 id="examples">Example Requests</h2>
<h3>Schema Management Example</h3>
<p><strong>Create a collection:</strong></p>
<pre><code>curl -X POST "`)
	sb.WriteString(fmt.Sprintf(`%s%s/collections:create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "products",
    "columns": [
      {"name": "title", "type": "string", "nullable": false},
      {"name": "price", "type": "float", "nullable": false},
      {"name": "description", "type": "text", "nullable": true}
    ]
  }'`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
<p><strong>Response (201 Created):</strong></p>
<pre><code>{
  "message": "Collection created successfully",
  "collection": {
    "name": "products",
    "columns": [...]
  }
}</code></pre>

<h3>Data Access Example</h3>
<p><strong>Create a record:</strong></p>
<pre><code>curl -X POST "`)
	sb.WriteString(fmt.Sprintf(`%s%s/{collection}:create" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "title": "Wireless Mouse",
      "price": 29.99,
      "description": "Ergonomic wireless mouse"
    }
  }'`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
<p><strong>Response (201 Created):</strong></p>
<pre><code>{
  "message": "Record created successfully",
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FBX",
    "title": "Wireless Mouse",
    "price": 29.99,
    "description": "Ergonomic wireless mouse"
  }
}</code></pre>

<p><strong>List records with filtering:</strong></p>
<pre><code>curl -s "`)
	sb.WriteString(fmt.Sprintf(`%s%s/{collection}:list?price[gt]=20&sort=-price" | jq .`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>

<h3>Aggregation Example</h3>
<p><strong>Count records:</strong></p>
<pre><code>curl -s "`)
	sb.WriteString(fmt.Sprintf(`%s%s/{collection}:count"`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
<p><strong>Response (200 OK):</strong></p>
<pre><code>{"value": 42}</code></pre>

<p><strong>Sum with filter:</strong></p>
<pre><code>curl -s "`)
	sb.WriteString(fmt.Sprintf(`%s%s/{collection}:sum?field=price&category[eq]=electronics"`, html.EscapeString(baseURL), html.EscapeString(prefix)))
	sb.WriteString(`</code></pre>
<p><strong>Response (200 OK):</strong></p>
<pre><code>{"value": 1599.99}</code></pre>
`)

	sb.WriteString("</body>\n</html>")

	return sb.String()
}

// generateMarkdown generates the Markdown documentation
func (h *DocHandler) generateMarkdown() string {
	prefix := h.config.Server.Prefix
	baseURL := fmt.Sprintf("http://localhost:%d", h.config.Server.Port)
	collections := h.getCollectionNames()

	var sb strings.Builder

	sb.WriteString("# Moon API Documentation\n\n")
	sb.WriteString(fmt.Sprintf("**Version:** %s  \n", h.version))
	sb.WriteString("**Service:** moon\n\n")

	// Table of Contents
	sb.WriteString("## Table of Contents\n\n")
	sb.WriteString("- [Overview](#overview)\n")
	sb.WriteString("- [Authentication](#authentication)\n")
	sb.WriteString("- [Base URL and Prefix](#base-url-and-prefix)\n")
	sb.WriteString("- [Quickstart](#quickstart)\n")
	sb.WriteString("- [Available Collections](#available-collections)\n")
	sb.WriteString("- [Schema Management](#schema-management)\n")
	sb.WriteString("- [Data Access](#data-access)\n")
	sb.WriteString("- [Aggregation Operations](#aggregation-operations)\n")
	sb.WriteString("- [Filtering, Sorting & Pagination](#filtering-sorting--pagination)\n")
	sb.WriteString("- [Error Responses](#error-responses)\n")
	sb.WriteString("- [Example Requests](#example-requests)\n\n")

	// Overview
	sb.WriteString("## Overview\n\n")
	sb.WriteString("Moon is a dynamic headless engine that provides a RESTful API for managing database tables and records without manual migrations. All endpoints follow the AIP-136 custom actions pattern using a colon separator between the resource and action.\n\n")

	// Authentication
	sb.WriteString("## Authentication\n\n")
	if h.config.JWT.Secret != "" || h.config.APIKey.Enabled {
		sb.WriteString("This API requires authentication for most endpoints.\n\n")
		if h.config.JWT.Secret != "" {
			sb.WriteString("**JWT Authentication:** Include a bearer token in the Authorization header:\n\n")
			sb.WriteString("```\nAuthorization: Bearer <your-jwt-token>\n```\n\n")
		}
		if h.config.APIKey.Enabled {
			sb.WriteString(fmt.Sprintf("**API Key Authentication:** Include your API key in the %s header:\n\n", h.config.APIKey.Header))
			sb.WriteString(fmt.Sprintf("```\n%s: <your-api-key>\n```\n\n", h.config.APIKey.Header))
		}
	} else {
		sb.WriteString("Authentication is currently disabled for this instance.\n\n")
	}

	// Base URL and Prefix
	sb.WriteString("## Base URL and Prefix\n\n")
	sb.WriteString(fmt.Sprintf("**Base URL:** `%s`\n\n", baseURL))
	if prefix != "" {
		sb.WriteString(fmt.Sprintf("**URL Prefix:** `%s`\n\n", prefix))
		sb.WriteString(fmt.Sprintf("All endpoints are prefixed with `%s`. For example:\n\n", prefix))
		sb.WriteString(fmt.Sprintf("```\n%s%s/health\n%s%s/collections:list\n```\n\n", baseURL, prefix, baseURL, prefix))
	} else {
		sb.WriteString("**URL Prefix:** None (endpoints start at root)\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s/health\n%s/collections:list\n```\n\n", baseURL, baseURL))
	}

	// Quickstart
	sb.WriteString("## Quickstart\n\n")
	sb.WriteString("Get started with Moon in 5 simple steps:\n\n")
	sb.WriteString("### 1. Create a collection\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -X POST "%s%s/collections:create" \
  -H "Content-Type: application/json" \
  -d '{"name":"products","columns":[{"name":"title","type":"string","nullable":false},{"name":"price","type":"float","nullable":false}]}'
`, baseURL, prefix))
	sb.WriteString("```\n\n")

	sb.WriteString("### 2. Insert a record\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -X POST "%s%s/{collection}:create" \
  -H "Content-Type: application/json" \
  -d '{"data":{"title":"Laptop","price":999.99}}'
`, baseURL, prefix))
	sb.WriteString("```\n\n")

	sb.WriteString("### 3. List records\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -s "%s%s/{collection}:list" | jq .
`, baseURL, prefix))
	sb.WriteString("```\n\n")

	sb.WriteString("### 4. Update a record\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -X POST "%s%s/{collection}:update" \
  -H "Content-Type: application/json" \
  -d '{"id":"01ARZ3NDEKTSV4RRFFQ69G5FBX","data":{"price":899.99}}'
`, baseURL, prefix))
	sb.WriteString("```\n\n")

	sb.WriteString("### 5. Delete a record\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -X POST "%s%s/{collection}:destroy" \
  -H "Content-Type: application/json" \
  -d '{"id":"01ARZ3NDEKTSV4RRFFQ69G5FBX"}'
`, baseURL, prefix))
	sb.WriteString("```\n\n")

	// Available Collections
	sb.WriteString("## Available Collections\n\n")
	if len(collections) > 0 {
		sb.WriteString("The following collections are currently available:\n\n")
		for _, name := range collections {
			sb.WriteString(fmt.Sprintf("- `%s`\n", name))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("No collections have been created yet.\n\n")
	}

	// Schema Management
	sb.WriteString("## Schema Management\n\n")
	sb.WriteString("These endpoints manage database tables (collections) and their schemas.\n\n")
	sb.WriteString("| Endpoint | Method | Description |\n")
	sb.WriteString("|----------|--------|-------------|\n")
	sb.WriteString(fmt.Sprintf("| `%s/collections:list` | GET | List all collections |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/collections:get` | GET | Get collection schema (requires `?name=...`) |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/collections:create` | POST | Create a new collection |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/collections:update` | POST | Update collection schema |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/collections:destroy` | POST | Delete a collection |\n", prefix))
	sb.WriteString("\n")

	// Data Access
	sb.WriteString("## Data Access\n\n")
	sb.WriteString("These endpoints manage records within a specific collection. Replace `{collection}` with your collection name.\n\n")
	sb.WriteString("| Endpoint | Method | Description |\n")
	sb.WriteString("|----------|--------|-------------|\n")
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:list` | GET | List all records |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:get` | GET | Get a single record (requires `?id=...`) |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:create` | POST | Create a new record |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:update` | POST | Update an existing record |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:destroy` | POST | Delete a record |\n", prefix))
	sb.WriteString("\n")

	// Aggregation
	sb.WriteString("## Aggregation Operations\n\n")
	sb.WriteString("Server-side aggregation endpoints for analytics. Replace `{collection}` with your collection name.\n\n")
	sb.WriteString("| Endpoint | Method | Description |\n")
	sb.WriteString("|----------|--------|-------------|\n")
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:count` | GET | Count records |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:sum` | GET | Sum numeric field (requires `?field=...`) |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:avg` | GET | Average numeric field (requires `?field=...`) |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:min` | GET | Minimum value (requires `?field=...`) |\n", prefix))
	sb.WriteString(fmt.Sprintf("| `%s/{collection}:max` | GET | Maximum value (requires `?field=...`) |\n", prefix))
	sb.WriteString("\n")

	// Filtering, Sorting & Pagination
	sb.WriteString("## Filtering, Sorting & Pagination\n\n")
	sb.WriteString("### Filtering\n\n")
	sb.WriteString("Use query parameters to filter results:\n\n")
	sb.WriteString("```\n?column[operator]=value\n```\n\n")
	sb.WriteString("**Operators:** `eq`, `ne`, `gt`, `lt`, `gte`, `lte`, `like`, `in`\n\n")
	sb.WriteString("**Example:**\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`%s%s/{collection}:list?price[gt]=100&category[eq]=electronics
`, baseURL, prefix))
	sb.WriteString("```\n\n")

	sb.WriteString("### Sorting\n\n")
	sb.WriteString("Sort by field (ascending) or `-field` (descending):\n\n")
	sb.WriteString("```\n?sort=-price,name\n```\n\n")

	sb.WriteString("### Full-Text Search\n\n")
	sb.WriteString("Search across all text columns:\n\n")
	sb.WriteString("```\n?q=laptop\n```\n\n")

	sb.WriteString("### Field Selection\n\n")
	sb.WriteString("Return only specific fields (always includes id):\n\n")
	sb.WriteString("```\n?fields=name,price\n```\n\n")

	sb.WriteString("### Pagination\n\n")
	sb.WriteString("Use cursor-based pagination:\n\n")
	sb.WriteString("```\n?after=01ARZ3NDEKTSV4RRFFQ69G5FBX&limit=10\n```\n\n")
	sb.WriteString("Response includes `next_cursor` when more results are available.\n\n")

	// Error Responses
	sb.WriteString("## Error Responses\n\n")
	sb.WriteString("All errors follow a consistent JSON structure:\n\n")
	sb.WriteString("```json\n{\n  \"error\": \"Error message describing what went wrong\",\n  \"code\": 400\n}\n```\n\n")
	sb.WriteString("**Common Status Codes:**\n\n")
	sb.WriteString("- `400 Bad Request` - Invalid input, missing required field, invalid filter operator\n")
	sb.WriteString("- `404 Not Found` - Collection or record not found\n")
	sb.WriteString("- `409 Conflict` - Collection already exists\n")
	sb.WriteString("- `500 Internal Server Error` - Server error\n\n")

	// Example Requests
	sb.WriteString("## Example Requests\n\n")
	sb.WriteString("### Schema Management Example\n\n")
	sb.WriteString("**Create a collection:**\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -X POST "%s%s/collections:create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "products",
    "columns": [
      {"name": "title", "type": "string", "nullable": false},
      {"name": "price", "type": "float", "nullable": false},
      {"name": "description", "type": "text", "nullable": true}
    ]
  }'
`, baseURL, prefix))
	sb.WriteString("```\n\n")
	sb.WriteString("**Response (201 Created):**\n\n```json\n")
	sb.WriteString("{\n  \"message\": \"Collection created successfully\",\n  \"collection\": {\n    \"name\": \"products\",\n    \"columns\": [...]\n  }\n}\n```\n\n")

	sb.WriteString("### Data Access Example\n\n")
	sb.WriteString("**Create a record:**\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -X POST "%s%s/{collection}:create" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "title": "Wireless Mouse",
      "price": 29.99,
      "description": "Ergonomic wireless mouse"
    }
  }'
`, baseURL, prefix))
	sb.WriteString("```\n\n")
	sb.WriteString("**Response (201 Created):**\n\n```json\n")
	sb.WriteString("{\n  \"message\": \"Record created successfully\",\n  \"data\": {\n    \"id\": \"01ARZ3NDEKTSV4RRFFQ69G5FBX\",\n    \"title\": \"Wireless Mouse\",\n    \"price\": 29.99,\n    \"description\": \"Ergonomic wireless mouse\"\n  }\n}\n```\n\n")

	sb.WriteString("**List records with filtering:**\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -s "%s%s/{collection}:list?price[gt]=20&sort=-price" | jq .
`, baseURL, prefix))
	sb.WriteString("```\n\n")

	sb.WriteString("### Aggregation Example\n\n")
	sb.WriteString("**Count records:**\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -s "%s%s/{collection}:count"
`, baseURL, prefix))
	sb.WriteString("```\n\n")
	sb.WriteString("**Response (200 OK):**\n\n```json\n{\"value\": 42}\n```\n\n")

	sb.WriteString("**Sum with filter:**\n\n```bash\n")
	sb.WriteString(fmt.Sprintf(`curl -s "%s%s/{collection}:sum?field=price&category[eq]=electronics"
`, baseURL, prefix))
	sb.WriteString("```\n\n")
	sb.WriteString("**Response (200 OK):**\n\n```json\n{\"value\": 1599.99}\n```\n")

	return sb.String()
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
