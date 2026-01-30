# 031-health-endpoint-minimal-output.md

## 1. Overview

### Problem Statement

The current health endpoint exposes internal implementation details, such as the database type and collection count, which can be a security risk and is unnecessary for external consumers. The health endpoint should provide only minimal, non-sensitive information to indicate service liveness and version.

### Context and Background

The `/health` endpoint is used for liveness/readiness checks by external systems and users. Exposing internal details (e.g., database type) is not required and may increase the attack surface. Industry best practices recommend minimal, non-sensitive health check responses.

### High-Level Solution Summary

Redesign the health endpoint to return only the following fields:

- `status`: Service liveness (e.g., `live`, `down`, etc.)
- `name`: Service name (e.g., `moon`)
- `version`: Service version, formatted as `{major}-{short-git-commit}` (e.g., `1-87811bc`)

## 2. Requirements

### Functional Requirements

- The `/health` endpoint MUST return a JSON object with only the following fields:
  - `status`: One of `live`, `down`, or other predefined statuses
  - `name`: The string `moon`
  - `version`: The current service version in the format `{major}-{short-git-commit}`
- The endpoint MUST NOT expose internal details such as database type, collection count, or consistency status.
- The endpoint MUST always return HTTP 200, regardless of service state.
- The response message (`status` field) MUST indicate the service state (e.g., `live`, `down`).
- Users MUST check the `status` field in the response to determine service health.

### Technical Requirements

- The `version` field MUST be set as follows:
  - The major version number is set in the build configuration or a version file (e.g., `VERSION` or via build flags)
  - The minor version is the short (7-character) git commit hash from the current build
  - The final version string is formatted as `{major}-{short-git-commit}` (e.g., `1-87811bc`)
- The build process MUST inject the git commit hash into the binary (e.g., using Go build flags: `-ldflags "-X main.GitCommit=$(git rev-parse --short HEAD)"`)
- The service MUST log detailed health check diagnostics internally, but not expose them via the endpoint.

### API Specification

- **Endpoint:** `GET /health`
- **Response (200):**
  ```json
  {
    "status": "live", // or "down", etc.
    "name": "moon",
    "version": "1-87811bc"
  }
  ```

  - The HTTP status code is always 200. The `status` field in the response indicates the actual service state. The user must check the `status` field to determine if the service is live or down.

### Validation Rules and Constraints

- Only the specified fields are allowed in the response.
- The `version` field MUST match the required format.
- The endpoint MUST NOT leak any internal or sensitive information.

### Error Handling and Failure Modes

- If the service is not live, set `status` to `down` in the response, but still return HTTP 200.
- All errors and diagnostics MUST be logged internally, not exposed in the response.

### Permissions and Limits

- The endpoint is public and requires no authentication.

## 3. Acceptance Criteria

### Verification Steps

- [ ] Deploy the service and call `GET /health` when live. Verify the response contains only `status`, `name`, and `version`.
- [ ] Confirm the `version` field is in the format `{major}-{short-git-commit}`.
- [ ] Simulate a down state and verify the endpoint returns HTTP 200 and `status: down`.
- [ ] Confirm no internal details (e.g., database type) are present in the response.
- [ ] Check logs to ensure detailed diagnostics are recorded internally.
- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.

### Test Scenarios

- **Scenario 1:** Service is live
  - Request: `GET /health`
  - Response: HTTP 200, `{ "status": "live", "name": "moon", "version": "1-87811bc" }`
- **Scenario 2:** Service is down
  - Request: `GET /health`
  - Response: HTTP 200, `{ "status": "down", "name": "moon", "version": "1-87811bc" }`
- **Scenario 3:** Attempt to access internal details
  - Request: `GET /health`
  - Response: No internal details present

### Edge Cases and Negative Paths

- If the version cannot be determined, set `version` to `unknown` and log a warning.
- If the service is in a degraded state, return `status: down` and HTTP 200.

---

**How to Set the Version:**

- Set the major version in a `VERSION` file or as a build flag (e.g., `-X main.MajorVersion=1`).
- Inject the short git commit hash at build time using:
  ```sh
  go build -ldflags "-X main.GitCommit=$(git rev-parse --short HEAD)"
  ```
- The application should concatenate these as `{major}-{short-git-commit}` for the `version` field.
- Example: If `VERSION` is `1` and git commit is `87811bc`, the version is `1-87811bc`.
