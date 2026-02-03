## Overview
- **Problem:** External monitoring services, documentation portals, and browser-based integration tools are currently blocked from accessing system health and documentation endpoints due to restrictive Cross-Origin Resource Sharing (CORS) policies.
- **Context:** The `/health`, `/doc`, and `/doc/md` endpoints provide public, non-sensitive information essential for operational visibility and developer integration.
- **Solution:** Explicitly configure the application's CORS middleware to permit requests from any origin (`*`) for these specific endpoints, bypassing standard security restrictions applied to data-sensitive API routes.

## Requirements

### Functional Requirements
- **Public Access:** The following endpoints must be publicly accessible from any origin:
  - `GET /health`
  - `GET /doc`
  - `GET /doc/md`
- **CORS Headers:** Responses from these endpoints must include the header `Access-Control-Allow-Origin: *`.
- **No Authentication:** These endpoints must bypass any global authentication middleware (JWT, API Key) if applied generally to the root.
- **Security Isolation:** This open CORS policy must **not** apply to other API endpoints (e.g., `/api/v1/*`), which must retain their strict CORS and authentication settings.

### Technical Requirements
- **Middleware Logic:** Implement a CORS exception or specific middleware configuration for the identified paths before applying the default restrictive policy.
- **Header Verification:** Ensure `Access-Control-Allow-Methods` includes `GET` (and `OPTIONS` if preflighting is relevant, though simple GETs often skip it).
- **Configuration:** While hardcoding for these system routes is acceptable, ensuring the logic is centralized in the router setup is required.

### API Specifications
**Endpoints affected:**
- `GET /health`
- `GET /doc`
- `GET /doc/md`

**Expected Header Output:**
```http
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Content-Type: application/json (or text/markdown for /doc/md)
...
```

## Acceptance

### Verification Steps
1. **Health Check via Curl:** Run `curl -I http://localhost:8080/health` and verify the output contains `Access-Control-Allow-Origin: *`.
2. **Docs Check via Browser Console:** Attempt to `fetch('http://localhost:8080/doc')` from a browser console on a different domain (e.g., `google.com`). It must succeed without a CORS error.
3. **Negative Test:** Attempt to access a protected endpoint (e.g., `/api/v1/users`) from a different origin without proper CORS configuration; it should fail or return restricted headers as per existing policy.
4. **Auth Bypass:** Confirm that accessing `/health` does not require a Bearer token or API key.

### Automated Testing
- **Integration Test:** Write a test case that sends a request to `/health` and asserts:
  - Status Code is 200.
  - Header `Access-Control-Allow-Origin` equals `*`.
- **Security Regression:** Ensure a test case confirms that `/api/v1/*` endpoints do **not** return `Access-Control-Allow-Origin: *` blindly.

---

- [x] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [x] Ensure all unit tests and integration tests are passing successfully.
