// Package handlers provides HTTP request handlers for the Moon API.
// This file implements documentation generation endpoints.
package handlers

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"net/http"
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
        pre { background: #2c3e50; color: #ecf0f1; padding: 15px; border-radius: 5px; overflow-x: auto; }
        pre code { background: none; color: inherit; padding: 0; }
        table { border-collapse: collapse; width: 100%; margin: 15px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background: #3498db; color: white; }
        tr:nth-child(even) { background: #f9f9f9; }
        a { color: #2980b9; text-decoration: none; }
        a:hover { text-decoration: underline; }
        ul { padding-left: 20px; }
        li { margin: 8px 0; }
    </style>
</head>
<body>
`)

	// Add the converted HTML body
	sb.WriteString(htmlBody.String())

	sb.WriteString("</body>\n</html>")

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
