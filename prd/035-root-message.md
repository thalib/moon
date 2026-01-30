## Overview
- Return a simple plain-text message when a client visits the root path `/`.
- Provide a fun, human-friendly response at the server root without affecting other routes.
- Ensure behavior is explicit, testable, and compatible with existing routing.

## Requirements
- When a request is made to `/`, respond with HTTP 200 and the exact body: "Darling, the Moon is still the Moon in all of its phases."
- The response must be plain text (no JSON wrapper) unless SPEC.md requires a different default for root responses. If conflicting, mark as **Needs Clarification**.
- Only the root path `/` should return this message; all other routes must behave as defined in SPEC.md.
- The response must be deterministic and not depend on configuration or runtime state.
- Add unit tests covering the root response and ensuring other routes are unchanged.
- If the router already has a root handler defined in SPEC.md, this must not override it without explicit approval (**Needs Clarification**).

## Acceptance
- Requesting `/` returns HTTP 200 and the exact message body.
- Content type is `text/plain; charset=utf-8` unless SPEC.md mandates a different default (**Needs Clarification** if different).
- Existing routes continue to function without behavior changes.
- Unit tests validate the root response and do not introduce new failures.

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
