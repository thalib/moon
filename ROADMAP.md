## SPEC Compliance Audit (2026-01-29)

### Missing from Implementation (SPEC-required)
- **Auth middleware enforcement**: JWT and API key checks exist but are not applied before handlers. The SPEC requires a security layer that enforces allow/deny on every request.
