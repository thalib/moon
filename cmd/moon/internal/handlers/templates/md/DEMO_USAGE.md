# Markdown Include Feature - Usage Demonstration

This document demonstrates how to use the Markdown file inclusion feature in Moon's documentation template.

## Quick Example

To include this file in the main template, add:

```go
{{ include "DEMO_USAGE.md" }}
```

## Practical Examples

### 1. Including a Footer

Create `footer.md`:
```markdown
---
Made with ❤️ by the Moon team
```

Use in template:
```go
{{ include "footer.md" }}
```

### 2. Modular Documentation Sections

Split large documentation into manageable files:

**authentication.md**:
```markdown
## Authentication

Moon supports JWT and API Key authentication...
```

**data-access.md**:
```markdown
## Data Access

Access your data using RESTful endpoints...
```

**Template usage**:
```go
{{ include "authentication.md" }}
{{ include "data-access.md" }}
```

### 3. Conditional Includes

Include content based on configuration:

```go
{{if .JWTEnabled}}
{{ include "jwt-guide.md" }}
{{end}}

{{if .APIKeyEnabled}}
{{ include "apikey-guide.md" }}
{{end}}
```

### 4. Multiple Includes in Sequence

```go
{{ include "introduction.md" }}

## Main Content

Your main documentation here...

{{ include "advanced-topics.md" }}
{{ include "troubleshooting.md" }}
{{ include "footer.md" }}
```

## Best Practices

1. **Keep Files Focused**: Each include should cover one topic
2. **Use Descriptive Names**: `quickstart.md` not `file1.md`
3. **Document Dependencies**: Note if includes reference each other
4. **Test Changes**: Always rebuild and test after modifying includes
5. **Version Control**: Commit all `.md` files in the `md/` directory

## File Structure Example

```
templates/md/
├── README.md                 # Documentation
├── DEMO_USAGE.md            # This file
├── example.md               # Basic example
├── footer.md                # Footer content
├── troubleshooting.md       # Help section
├── authentication.md        # Auth guide
├── data-access.md           # Data API guide
└── advanced-topics.md       # Advanced features
```

## Error Handling Demo

If you try to include a non-existent file:

```go
{{ include "nonexistent.md" }}
```

You'll see:
- Console warning: `WARNING: Failed to read markdown file nonexistent.md`
- Output contains: `<!-- Error: Failed to include nonexistent.md -->`
- Template continues rendering normally

## Integration Example

Here's a complete example showing how the main template might be structured:

```go
{{- $ApiURL := .BaseURL -}}
{{- if .Prefix }}{{- $ApiURL = printf "%s%s" .BaseURL .Prefix -}}{{- end -}}

# Moon API Documentation

{{ include "introduction.md" }}

## Quick Start

{{ include "quickstart.md" }}

## Authentication

{{if .JWTEnabled}}
{{ include "jwt-authentication.md" }}
{{end}}

{{if .APIKeyEnabled}}
{{ include "apikey-authentication.md" }}
{{end}}

## API Reference

{{ include "collections-api.md" }}
{{ include "data-api.md" }}
{{ include "aggregation-api.md" }}

## Advanced Topics

{{ include "advanced-topics.md" }}

{{ include "troubleshooting.md" }}

{{ include "footer.md" }}
```

## Building and Testing

After creating or modifying include files:

1. **Build**: `go build ./cmd/moon`
2. **Test**: `go test ./cmd/moon/internal/handlers -v`
3. **Run**: Start the server and visit `/doc/`
4. **Verify**: Check that included content appears correctly

## Performance Notes

- Files are embedded at compile time (no runtime I/O)
- No performance penalty vs inline content
- Cached after first render
- Efficient for production use

## Next Steps

1. Create your own include files in `templates/md/`
2. Reference them in `doc.md.tmpl` using `{{ include "filename.md" }}`
3. Build and test
4. Deploy with confidence!

For more information, see:
- `templates/md/README.md` - Feature documentation
- `templates/MARKDOWN_INCLUDES.md` - Technical details
- `doc_test.go` - Test examples
