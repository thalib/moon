## Overview

- Make the API URL prefix configurable via the configuration file instead of hardcoding `/api/v1/`
- Default behavior should be **no prefix** (empty string), allowing direct access to endpoints like `/health` and `/collections:list`
- When a prefix is configured (e.g., `/moon/api/`), all endpoints are mounted under that prefix
- This provides deployment flexibility for reverse proxy setups, multi-tenant routing, or organizational URL standards

## Requirements

- Add a new `prefix` field to the `server` section of the configuration file (YAML)
- Default value must be an empty string (`""`) representing no prefix
- If a prefix is configured, it must be normalized:
  - Leading slash is required (add if missing)
  - Trailing slash is optional (preserve if present, but normalize consistently)
- Update the router setup to mount all routes under the configured prefix
- All route definitions must be prefix-agnostic (relative to the mount point)
- Health endpoint, collections endpoints, and data endpoints must all respect the configured prefix
- OpenAPI documentation must reflect the configured prefix in all paths
- Configuration validation must ensure the prefix does not conflict with reserved paths
- Update all sample configuration files to show the new `prefix` field with empty default
- Update all test scripts to support an optional `PREFIX` environment variable for testing with custom prefixes
- Update documentation (`SPEC.md`, `USAGE.md`, `INSTALL.md`) to explain the prefix configuration
- Add unit tests verifying prefix behavior (empty, with trailing slash, without trailing slash)
- Add integration tests verifying all endpoints work correctly with and without prefix

## Acceptance

- Configuration file accepts `server.prefix` field with default value `""`
- When `prefix` is empty or not set:
  - `http://localhost:6006/health` returns health status
  - `http://localhost:6006/collections:list` lists collections
  - `http://localhost:6006/{collection}:list` lists records
- When `prefix` is set to `/moon/api/`:
  - `http://localhost:6006/moon/api/health` returns health status
  - `http://localhost:6006/moon/api/collections:list` lists collections
  - `http://localhost:6006/moon/api/{collection}:list` lists records
- OpenAPI documentation shows correct paths with configured prefix
- Test scripts can be run with `PREFIX=/moon/api ./scripts/collection.sh` to test prefixed endpoints
- All documentation is updated to reflect the configurable prefix
- All tests pass with both empty prefix and custom prefix configurations
- No hardcoded `/api/v1/` references remain in the codebase

## Implementation Checklist

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [ ] Ensure all test scripts in `scripts/*.sh` are working properly and up to date with the latest code and API changes.
