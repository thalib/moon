## Overview

- The DATA list endpoint (e.g., `GET /products:list`) does not include a `total` field in its response.
- API consumers need to know the total record count to implement UI features like pagination indicators and record count displays.
- This PRD adds the `total` field to list endpoint responses.

## Requirements

- Add a `total` field to all DATA list endpoint responses (e.g., `/products:list`, `/{collection}:list`).
- The `total` field must contain the total number of records in the collection that match the current query filters.
- The `total` value is independent of pagination (cursor, limit) and represents the full result set size.`total = 0` when no record no
- Response structure must include:
  - `data`: array of records
  - `total`: integer representing total record count
  - `next_cursor`: string or null for pagination
  - `limit`: integer for current page size
- Update `SPEC.md` to document the `total` field in list endpoint responses.
- Write or update unit tests to verify the `total` field is present and accurate.
- Update `cmd\moon\internal\handlers\templates\doc.md.tmpl` to reflect the new response structure with examples.
- Ensure `total` is calculated correctly when filters are applied.

## Acceptance

- All DATA list endpoints return a `total` field in the response.
- The `total` field accurately reflects the total number of records matching the query.
- Example request `GET /products:list` returns:
  ```json
  {
    "data": [...],
    "total": 2,
    "next_cursor": null,
    "limit": 15
  }
  ```
- Unit tests verify:
  - `total` field is present in list responses
  - `total` value matches the actual record count
  - `total` reflects filtered results when filters are applied
- `SPEC.md` documents the `total` field in list endpoint response schema.
- `doc.md.tmpl` includes updated examples with the `total` field.
- All existing tests continue to pass.

---

- [ ] Ensure all unit tests and integration tests in `tests/*` are passing successfully.
- [ ] Run all tests and ensure 100% pass rate.
- [ ] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.
