## Overview

- The API documentation template at `cmd\moon\internal\handlers\templates\doc.md.tmpl` contains inaccuracies and contradictions with the actual API responses.
- Documentation must reflect the real behavior of the API to be useful for developers and integrators.
- This PRD defines the process to verify and correct all API documentation by testing against a live moon server.
- Every curl command and endpoint example in `doc.md.tmpl` must be tested for accuracy against the live moon server.

## Requirements

- Start a moon server instance in a test environment.
- Test every API endpoint documented in `doc.md.tmpl` against the live server.
- Test each curl command in `doc.md.tmpl` to ensure it works as documented.
- Capture the actual request/response pairs for each endpoint and curl command.
- Compare actual responses with documented responses in `doc.md.tmpl`.
- Update `doc.md.tmpl` to match actual API behavior, including:
  - Response status codes
  - Response body structure and field names
  - Response data types
  - Error responses
  - Header values
  - Query parameters and their effects
  - Request body schemas
  - Curl command syntax and endpoint URLs
- Do not invent or guess API behavior; document only what is verified through actual testing.
- Preserve the existing template structure and formatting conventions of `doc.md.tmpl`.
- Ensure all examples use correct curl syntax and actual working endpoints.

## Acceptance

- Every endpoint and curl command documented in `doc.md.tmpl` has been tested against a running moon server.
- All response examples in `doc.md.tmpl` match actual server responses.
- All request examples and curl commands in `doc.md.tmpl` produce the documented responses when executed.
- No contradictions exist between documented and actual API behavior.
- The updated `doc.md.tmpl` can be used as a reliable reference for API consumers.
- [ ] Ensure all unit tests and integration tests in `tests/*` are passing successfully.

---

- [ ] Ensure all unit tests and integration tests in `tests/*` are passing successfully.
