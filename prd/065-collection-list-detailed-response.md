- What is this and why: Enhance the existing `collection:list` API so responses include useful metadata about each collection (record count). This improves observability and enables clients to display richer collection summaries without extra requests.
- Context: Current `collections:list` returns only collection names and a total count. Clients need record counts today, forcing extra calls or inefficient client-side estimation.
- High-level solution summary: Extend the API response to return for each collection: `name` and `records_count`. Keep the existing top-level `count` field.

example response:

```json
{
  "collections": 
  [
    { "name": "<collection_name>", records_count: <record_count> },
   { "name": "<collection_name>", records_count: <record_count> }
  ],
  "count": <collections_count>
}
```

## Requirements

- Functional requirements:
  - API: `GET /collections:list` (no behavior change to authentication or pagination).
  - Response shape: JSON object with `collections` (array) and `count` (integer).
  - Each item in `collections` must include:
    - `name` (string)
    - `records_count` (integer) — total number of records in the collection
  - Do not maintain *backwards compatibility*

- Technical requirements:
  - Server must compute `records_count` efficiently (use existing DB count APIs; avoid full table scans when possible).
  - `records_count` must be derived or computed from the authoritative data source (database).
  - Response must be produced within existing API latency SLAs; if computing counts is expensive, consider caching or async refresh with cached values returned.
  - Add unit tests for response shape and sample values. Add integration test exercising endpoint with multiple collections.
  - Keep API authentication, authorization, and rate limits unchanged.

## Acceptance

- Verification steps:
  - Call `GET /collections:list` with valid auth and confirm HTTP 200.
  - Validate JSON schema: top-level `collections` array and `count` integer.
  - For at least three collections, verify returned `records_count` matches DB `COUNT(*)`.
  - Confirm legacy clients that only parse `name` and `count` still work.

- Test scenarios / scripts:
  - Unit: mock DB layer to assert `collections` items contain `name` and `records_count`.
  - Integration: create 3 collections, insert known record counts; call endpoint and assert values.

- Edge cases and failure modes:
  - If `records_count` cannot be retrieved, API should still return the collection with `records_count: 0` and include a warning log entry.
  - If counts are stale due to caching, document that counts are eventual and include TTL notes in developer docs.

## Needs Clarification

- Performance SLA for computing counts under large scale (if strict, implement caching strategy).

- Implementation notes:
  - Prefer using existing DB metadata endpoints or efficient count strategies (e.g., indexed counters) to avoid full scans.
  - Add tests under the existing test harness and update docs template as noted.

- Checklist (must complete before merge):
  - [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
  - [ ] Run all tests and ensure 100% pass rate.
  - [ ] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.
