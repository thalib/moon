# 034-log-invalid-url-requests.md

## 1. Overview

### Problem Statement
Currently, the server logs only valid URL requests and their responses. There is no visibility into invalid or unhandled URL requests, making it difficult to debug or monitor unexpected or erroneous client behavior.

### Context and Background
The log file shows only successful or valid endpoint accesses. For debugging and operational monitoring, it is necessary to capture and optionally enable/disable logging of invalid URL requests (e.g., 404 Not Found, unregistered routes) to the log file.

### High-Level Solution Summary
Introduce a configuration option to enable or disable logging of invalid URL requests. When enabled, any request to an invalid or unregistered URL should be logged with relevant details (timestamp, method, URL, status code, and latency if available).

## 2. Requirements

### Functional Requirements
- Add a configuration option (e.g., `log_invalid_url_requests`) to the main configuration file.
- When enabled, log all invalid/unhandled URL requests to the log file.
- When disabled, do not log invalid URL requests (current behavior).
- Log entry for invalid URL requests must include:
  - Timestamp
  - HTTP method
  - Requested URL
  - Status code (e.g., 404)
  - Latency (if available)
- The configuration must be hot-reloadable if the system supports config reloads; otherwise, require restart.

### Technical Requirements
- The configuration option must be documented in `moon.conf` and `INSTALL.md`.
- The logging format for invalid URLs must match the style of valid request logs for consistency.
- The implementation must not introduce any new dependencies.
- The feature must be covered by unit tests.

### API Specifications
- No new API endpoints are required.
- Configuration example:
  ```
  # Enable or disable logging of invalid URL requests
  log_invalid_url_requests = true
  ```

### Validation Rules and Constraints
- Only log requests that result in a 404 or similar unhandled status.
- Do not log valid or handled requests as invalid.
- The configuration must default to `false` if not specified.

### Error Handling and Failure Modes
- If the configuration value is invalid or missing, default to `false` (do not log invalid URLs).
- If logging fails (e.g., file write error), do not affect request handling; log the error if possible.

### Filtering, Sorting, Permissions, and Limits
- No special filtering or permissions required.
- No rate limiting for invalid URL logging, but log volume should be considered in documentation.

## 3. Acceptance Criteria

### Verification Steps
- Set `log_invalid_url_requests = true` in the config and restart the server.
- Send requests to both valid and invalid URLs.
- Confirm that valid requests are logged as before.
- Confirm that invalid requests (e.g., 404) are logged with the required details.
- Set `log_invalid_url_requests = false` and restart the server.
- Confirm that invalid requests are not logged.

### Test Scenarios
- [x] Valid URL request is logged as before.
- [x] Invalid URL request is logged when config is enabled.
- [x] Invalid URL request is not logged when config is disabled.
- [x] Log entry for invalid URL includes timestamp, method, URL, status code, and latency.
- [x] Configuration defaults to `false` if not set.
- [x] Invalid config value does not break logging; defaults to `false`.

### Expected API Responses
- Not applicable (no new API endpoints).

### Edge Cases and Negative Paths
- [x] Rapid/frequent invalid requests do not cause log flooding (documented, not enforced).
- [x] Logging failure does not impact request handling.
- [x] Invalid config value (e.g., non-boolean) is handled gracefully.

---

- [x] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [x] Ensure all test scripts in `scripts/*.sh` are working properly and up to date with the latest code and API changes.
