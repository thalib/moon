## Overview
- Refactor documentation generation to use Go's embed package with template files instead of inline strings.
- Maintain a single Markdown template as the source of truth for all documentation content.
- Auto-generate HTML from Markdown using a Markdown-to-HTML library to avoid duplicate maintenance.
- Keep single-file deployment capability by embedding templates into the binary.
- Improve maintainability by separating presentation (templates) from logic (Go code).

## Requirements
- Use Go's `embed` package to embed template files into the binary at compile time.
- Create a single Markdown template file (`templates/doc.md.tmpl`) as the source of truth for documentation content.
- Use Go's `text/template` package to render the Markdown template with dynamic data.
- Use a Markdown-to-HTML conversion library (e.g., `github.com/gomarkdown/markdown` or `github.com/yuin/goldmark`) to convert rendered Markdown to HTML.
- Wrap generated HTML with a minimal HTML structure including:
  - DOCTYPE, head, meta tags, and body wrapper
  - CSS for styling (inline or embedded stylesheet)
  - Proper character encoding (UTF-8)
- Template must accept data structure containing:
  - Service name and version
  - Base URL and configured prefix
  - Authentication configuration (JWT enabled, API key enabled, header name)
  - List of available collection names
  - Any other dynamic values needed for documentation
- Generated Markdown and HTML outputs must match the existing documentation structure defined in PRD 036.
- Caching behavior must remain unchanged:
  - In-memory cache for both Markdown and HTML
  - ETag and Last-Modified headers for client-side caching
  - Cache refresh endpoint
- Both `/doc/` (HTML) and `/doc/md` (Markdown) endpoints must continue to work.
- Template file must use Go template syntax (e.g., `{{.ServiceName}}`, `{{range .Collections}}`, etc.).
- Do not use conditional variable assignment inside the template for URL construction (e.g., `if prefix then api_url = base_url/prefix`). Compute `api_base_url` in Go and pass it to the template for consistent use.
- Error handling for template loading and parsing must be graceful:
  - Log errors during startup if templates fail to load
  - Return appropriate HTTP error if rendering fails
- Templates must be loaded once at startup, not on every request.
- No external template files should be required at runtime (all embedded in binary).
- CSS for HTML output should be minimal and functional, with clear section separation and code block styling.
- The Markdown-to-HTML converter must preserve:
  - Code blocks with syntax preservation
  - Tables
  - Headings with proper hierarchy
  - Lists (ordered and unordered)
  - Links and anchors

## Markdown Template Outline

The template file `doc.md.tmpl` must follow this structure:

```markdown
## overwivew
### Authentication
### Base URL and Prefix
### Response
- with table Common Status Codes
- Sample "Error Responses"
## Endpoints
### Collections
### Data Access
### Aggregation Operations
### Filtering
### Search
### Sorting
### Pagination
## Available Collections
## Example Requests

```

**Template Data Structure:**

The template expects a data structure with these fields:

```go
type DocData struct {
    ServiceName   string   // "moon"
    Version       string   // e.g., "1.99"
    BaseURL       string   // e.g., "http://localhost:6006"
    Prefix        string   // e.g., "/api/v1" or ""
    JWTEnabled    bool     // true if JWT authentication is configured
    APIKeyEnabled bool     // true if API key authentication is enabled
    APIKeyHeader  string   // e.g., "X-API-KEY"
    Collections   []string // List of available collection names
}
```

## Acceptance
- A single Markdown template file exists at `cmd/moon/internal/handlers/templates/doc.md.tmpl`.
- The template file contains all documentation content with placeholders for dynamic data.
- Go code uses `embed` package to embed the template file into the binary.
- Markdown template is parsed and cached at startup using `text/template`.
- `/doc/md` endpoint renders the Markdown template and serves it as `text/markdown`.
- `/doc/` endpoint renders the Markdown template, converts it to HTML, wraps it with HTML structure and CSS, and serves it as `text/html`.
- Generated documentation matches the structure and content defined in PRD 036.
- No inline HTML or Markdown strings remain in Go code (only template data and rendering logic).
- Binary size increase is minimal (templates are text files).
- All existing tests pass and cache behavior is unchanged.
- Documentation endpoints return the same HTTP status codes and headers as before.
- Template rendering errors are logged and result in appropriate HTTP error responses.

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Ensure all test scripts in `scripts/*.sh` are working properly and up to date with the latest code and API changes.
