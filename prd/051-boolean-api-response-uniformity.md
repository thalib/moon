## Overview
This PRD defines the requirements for ensuring uniformity in how boolean values are represented in API responses. Currently, depending on the underlying database driver (specifically SQLite), boolean values may be serialized as integers (`0` or `1`) instead of JSON boolean literals (`false` or `true`). This feature mandates that the internal storage format be abstracted away so that API consumers always receive standard JSON booleans, ensuring consistency across all supported database backends (SQLite, PostgreSQL, MySQL).

## Requirements

### Functional Requirements
- **Uniform Output:** All API responses containing boolean fields must serialize them as JSON `true` or `false`.
- **No Integer Leakage:** Under no circumstances should a boolean field be returned as `0` or `1` in the JSON response body.
- **Scope:** This rule applies to all API endpoints, including:
  - Single record retrieval
  - List/Collection retrieval
  - Aggregation results (where applicable)
  - Custom query outputs
- **Database Agnostic:** The API behavior must be identical regardless of whether the backend is SQLite (which natively uses integers for booleans), PostgreSQL, or MySQL.

### Technical Requirements
- **Data Abstraction:** The data access layer or response marshaling logic must detect fields defined as `boolean` in the schema and ensure they are converted to Go `bool` types before JSON encoding.
- **SQLite Handling:** Specifically for SQLite, the system must map the stored integer values (`0` for false, `1` for true) to Go boolean values during the data scanning/fetching process.
- **Schema Awareness:** The conversion logic should rely on the explicit schema definition of the field type, ensuring that actual integer fields remain integers and only intended boolean fields are converted.

### API Specifications
No new endpoints are created, but existing endpoints must conform to this format:

**GET /api/v1/users/1**
```json
{
  "id": 1,
  "is_active": true,      // CORRECT
  "has_verified": false   // CORRECT
}
```

**Invalid Response (to be fixed):**
```json
{
  "id": 1,
  "is_active": 1,         // INCORRECT
  "has_verified": 0       // INCORRECT
}
```

## Acceptance

### Verification Steps
1. **Define Schema:** Create a collection (e.g., `feature_flags`) with a field of type `boolean`.
2. **Insert Data:** Insert a record with the boolean field set to `true`.
3. **Retrieve Data:** Call the GET endpoint for that record.
4. **Validate JSON:** Ensure the response body contains the literal `true`, not `1` or `"1"`.
5. **Cross-Database Check:** Run the same test against an SQLite backend and verify the output is identical to PostgreSQL/MySQL.

### Automated Testing
- **Unit Tests:** Add test cases in the JSON marshaling or data mapping layer that explicitly assert boolean output for mock boolean inputs.
- **Integration Tests:** Create an end-to-end test case using the SQLite driver that inserts a row and verifies the raw HTTP response body contains valid JSON booleans.

### Edge Cases
- **Null Values:** If the boolean field is nullable and the value is `null`, it should remain `null` in JSON.
- **Aggregation:** Ensure `COUNT` or other aggregates distinct from the boolean value itself are handled correctly (usually integers), but if an aggregate returns a boolean concept (unlikely but possible in custom logic), it follows the rule.

---

- [x] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.MD`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [x] Ensure all unit tests and integration tests in `tests/*` are passing successfully.
