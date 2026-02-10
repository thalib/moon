## Overview

- The `/collections:list` endpoint currently returns only collection names as an array of strings.
- API consumers need detailed schema information for all collections without making individual `:get` requests for each collection.
- This PRD adds an optional `detailed` query parameter to return full collection schemas including field definitions.

## Requirements

- Add an optional `?detailed=true` query parameter to the `/collections:list` endpoint.
- When `detailed=false` or not specified (default behavior), return the current response format with collection names only.
- When `detailed=true`, return full collection objects with complete schema information for each collection.
- Response structure for detailed mode must include:
  - `collections`: array of collection objects (not strings)
  - Each collection object contains:
    - `name`: string representing the collection name
    - `columns`: array of column definitions with name, type, nullable, unique, and default_value properties
  - `count`: integer representing total number of collections
- Maintain backward compatibility: existing clients without the query parameter continue to receive the simple response.
- Update `SPEC.md` to document the `detailed` query parameter and response structure.
- Write or update unit tests to verify both simple and detailed response modes.
- Update `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect the new query parameter and response examples.
- Ensure consistent response format with existing `:get` endpoint structure for individual collections.

## Acceptance

- `/collections:list` without query parameter returns current format:
  ```json
  {
    "collections": ["products", "customers"],
    "count": 2
  }
  ```
- `/collections:list?detailed=true` returns detailed format:
  ```json
  {
    "collections": [
      {
        "name": "products",
        "columns": [
          {"name": "name", "type": "string", "nullable": false},
          {"name": "price", "type": "decimal", "nullable": false}
        ]
      },
      {
        "name": "customers",
        "columns": [
          {"name": "name", "type": "string", "nullable": false},
          {"name": "email", "type": "string", "nullable": false}
        ]
      }
    ],
    "count": 2
  }
  ```
- `/collections:list?detailed=false` returns simple format (same as no parameter).
- Unit tests verify:
  - Simple mode returns array of strings with `count`
  - Detailed mode returns array of collection objects with complete schemas
  - `count` field is accurate in both modes
  - System tables are filtered out in both modes
  - Backward compatibility is maintained
- `SPEC.md` documents the `detailed` query parameter and both response formats.
- `doc.md.tmpl` includes updated examples with both simple and detailed response formats.
- All existing tests continue to pass.

---

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Ensure all unit tests and integration tests in `tests/*` are passing successfully.
