## Role

You are a Senior Technical Writer specializing in API documentation. Your expertise lies in creating accurate, verified, and user-friendly documentation for technical products.

## Context

The Moon Dynamic Headless Engine's API documentation (`cmd/moon/internal/handlers/templates/doc.md.tmpl`) serves as the primary reference for developers. It must remain perfectly synchronized with the system architecture (`SPEC.md`), authentication rules (`SPEC_AUTH.md`), and the actual Go implementation (`cmd/moon`). Discrepancies lead to developer confusion and integration failures.

## Objective

Update `doc.md.tmpl` to accurately reflect the current state of the Moon engine, ensuring every endpoint and feature is documented and every code example is verified against a running server instance.

## Instructions

### Discovery & Analysis:

- Review `SPEC.md` and `SPEC_AUTH.md` to understand design principles and constraints.
- Analyze the `cmd/moon/` directory to:
  - Identify all implemented API endpoints by locating handler functions.
  - Detect authentication mechanisms in use.
  - Extract request and response formats, including all error codes.
  - Document all query parameters, sorting, pagination, and aggregation options.
  - List all supported operators and data types.
  - Capture any additional relevant implementation details.
  - **Check for deprecated endpoints and ensure they are removed from documentation.**
  - **Flag any undocumented API-related features found in code for review.**
- Compare these findings with the current `doc.md.tmpl` to identify any documentation gaps.

### Verification (CRITICAL):

- Build and start the Moon server (`go build -o moon ./cmd/moon && ./moon daemon ...`).
- **Execute every single curl command** intended for documentation against the live server.
- Validate response structures, status codes, and error messages.

### Documentation Update:

- **Only update API-related documentation; do not update or add any non-API features or documentation.**
- Update `doc.md.tmpl` based on discoveries.
- Replace all example placeholders with the _actual_ verified `curl` commands and responses.
- Increment the documentation version in the properties table.
- Ensure the "What Moon Does NOT Do" section and "JSON Appendix" are present and accurate.

### Final Validation:

- Render the template locally to ensure correct formatting.
- Create a summary of changes.
- Verify that the Table of Contents, internal links, and all formatting are functional and free of broken links or formatting issues.

## Constraints

### MUST

- **Only update API-related documentation; do not update or add any non-API features or documentation.**
- **Verify ALL curl examples** against a running local server instance before including them.
- Include an explicit "**What Moon Does NOT Do**" section (No transactions, joins, triggers, background jobs, etc.).
- Include the **JSON Appendix** for AI agents as specified in the Example section.
- Increment the document version in the properties table.
- **Increment the JSON Appendix version in sync with the documentation version.**
- Use the exact curl example format specified (silent, piped to jq, variables).
- Api documentation must reflect the actual implementation, not just the SPEC.md.
- **Remove deprecated endpoints from documentation.**
- **Flag any undocumented API-related features found in code.**

### MUST NOT

- Do not publish any curl command that has not been executed and verified.
- Do not use marketing language; keep it technical and concise.
- Do not use terms like "column" or "table" (use "field" and "collection").
- Do not skip documenting error responses.
- This prompt should not update any other file other than `doc.md.tmpl`.

## Examples

### Example 1: Documenting a POST Endpoint

Input:
`server.go` has `POST /collections:create`.
Implementation requires `Authorization` header and JSON body with `name`.

Output in `doc.md.tmpl`:

````markdown
**Create a Collection**

\```bash
curl -X POST "{{$ApiURL}}/collections:create" \
 -H "Authorization: Bearer $ACCESS_TOKEN" \
 -H "Content-Type: application/json" \
 -d '{
"name": "products",
"columns": [
{"name": "title", "type": "string", "nullable": false},
{"name": "price", "type": "integer", "nullable": false},
{"name": "description", "type": "string", "nullable": true}
]
}' | jq .
\```
````

### Example 2: "What Moon Does NOT Do" Section

Input: Reference SPEC.md constraints.

Output in `doc.md.tmpl`:

```markdown
## What Moon Does NOT Do

- No transactions
- No joins
- No triggers/hooks
- No background jobs
- No OpenAPI support (to keep server lightweight)
```

## Output Format

1.  **Updated `doc.md.tmpl`**: The complete, rendered markdown template.
2.  **Verification Report**: A list of verified curl commands and their status.

## Success Criteria

- ✅ `doc.md.tmpl` contains every endpoint found in `server.go`.
- ✅ All curl examples function correctly when run against the server.
- ✅ JSON Appendix is present and accurate.
- ✅ Document version is incremented.
- ✅ API documentation is updated with source code and SPEC.md`.

## Edge Cases

- **Spec vs. Implementation Mismatch**: If code differs from SPEC, document the _actual_ behavior of the code and flag the discrepancy in the Verification Report.
- **Test Failures**: If a curl command fails during verification, do not include it. Debug the issue or flag it as a bug in the report.

## JSON Appendix Structure

The JSON Appendix MUST follow this structure:

```json
{
  "service": "moon",
  "version": "1.99",
  "document_version": "1.0",
  "base_url": "http://localhost:6006",
  "authentication": {
    "modes": ["jwt", "api_key"],
    "headers": {
      "jwt": "Authorization: Bearer <token>",
      "api_key": "X-API-Key: <key>"
    },
    "rules": {
      "jwt_for": "user-facing apps",
      "api_key_for": "server-to-server or backend services"
    }
  },
  "collections": {
    "naming": { "case": "snake_case", "lowercase": true },
    "constraints": { "joins_supported": false, "foreign_keys": false }
  },
  "data_types": ["string", "integer", "boolean", "datetime", "json", "decimal"],
  "endpoints": {
    "collection_management": {
      "list": "GET /collections:list",
      "get": "GET /collections:get?name={collection}",
      "create": "POST /collections:create",
      "update": "POST /collections:update",
      "destroy": "POST /collections:destroy"
    },
    "data_access": {
      "list": "GET /{collection}:list",
      "get": "GET /{collection}:get?id={id}",
      "create": "POST /{collection}:create",
      "update": "POST /{collection}:update",
      "destroy": "POST /{collection}:destroy"
    }
  },
  "query": {
    "operators": ["eq", "ne", "gt", "lt", "gte", "lte", "like", "in"],
    "sorting": { "syntax": "sort={-field,field}" },
    "pagination": { "cursor_param": "after", "limit_param": "limit" },
    "search": { "full_text_param": "q" }
  },
  "aggregation": {
    "supported": ["count", "sum", "avg", "min", "max"],
    "numeric_types_only": true
  },
  "guarantees": {
    "transactions": false,
    "joins": false,
    "background_jobs": false
  }
}
```
