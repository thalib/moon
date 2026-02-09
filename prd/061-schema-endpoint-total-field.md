## Overview

- The DATA schema endpoint (e.g., `GET /products:schema`) does not include a `total` field in its response.
- API consumers need to know the total record count alongside schema information for UI rendering and data management.
- This PRD adds the `total` field to schema endpoint responses.

## Requirements

- Add a `total` field to all DATA schema endpoint responses (e.g., `/products:schema`, `/{collection}:schema`).
- The `total` field must contain the total number of records currently in the collection.
- Response structure must include:
  - `collection`: string representing the collection name
  - `fields`: array of field definitions with name, type, and nullable properties
  - `total`: integer representing total record count in the collection. `total = 0` when no record no
- Update `SPEC.md` to document the `total` field in schema endpoint responses.
- Write or update unit tests to verify the `total` field is present and accurate.
- Update `cmd\moon\internal\handlers\templates\doc.md.tmpl` to reflect the new response structure with examples.
- The `total` value represents the full record count regardless of any filters (since schema endpoint does not accept filters).

## Acceptance

- All DATA schema endpoints return a `total` field in the response.
- The `total` field accurately reflects the total number of records in the collection.
- Example request `GET /products:schema` returns:
  ```json
  {
    "collection": "products",
    "fields": [
      {"name": "id", "type": "string", "nullable": false},
      ...
    ],
    "total": 2
  }
  ```
- Unit tests verify:
  - `total` field is present in schema responses
  - `total` value matches the actual record count in the collection
- `SPEC.md` documents the `total` field in schema endpoint response schema.
- `doc.md.tmpl` includes updated examples with the `total` field.
- All existing tests continue to pass.

---

- [ ] Ensure all unit tests and integration tests in `tests/*` are passing successfully.
- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Update API doc template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect these changes.
- [ ] Run all tests and ensure 100% pass rate.
- [ ] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.
