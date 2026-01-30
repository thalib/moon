## Overview
- Replace build-time versioning derived from VERSION file and git commit hash with static version constants in configuration.
- Move versioning into `cmd/moon/internal/config/config.go` as two integers: `version_major` and `version_minor`, and expose a version string formatted as `{major}.{minor}`.
- This simplifies version management, removes dependency on external text files, and eliminates git commit hash usage.

## Requirements
- Remove usage of VERSION file and git commit hash in build versioning (no `{major}-{git-commit}` output).
- Add two integer constants in `cmd/moon/internal/config/config.go`: `version_major` and `version_minor`.
- Provide a version string formatted exactly as `{major}.{minor}` using the two constants (e.g., `1.99`).
- Ensure any existing version output in runtime or CLI uses the new `{major}.{minor}` format.
- If any build flags currently inject version, they must be removed or ignored in favor of config constants.
- Update or remove any references to VERSION file as a source of truth for versioning.
- Do not introduce new warnings or dependencies.
- Needs Clarification: identify all current code paths where version is displayed or used (e.g., CLI flags, logs, HTTP headers) and confirm they should all switch to `{major}.{minor}`.

## Acceptance
- Build no longer reads VERSION file or git commit hash for versioning.
- `version_major=1` and `version_minor=99` in `cmd/moon/internal/config/config.go` result in runtime version output `1.99`.
- Any version output surfaces the `{major}.{minor}` format consistently across the application.
- Tests updated/added to verify version formatting and removal of VERSION/git-based behavior (where applicable).
- No new compiler warnings introduced.

- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
