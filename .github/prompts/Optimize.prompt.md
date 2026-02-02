---
agent: agent
---

- Make sure you do not change the function or feature.
- For Go (Golang) code.
- If you have any doubts about a feature or code, always refer to the specification files: `SPEC.md`, `SPEC_AUTH.md`.
- Review and optimize the source code, then update it as needed.
- Run `go test` to ensure all tests pass.
- Ensure code is formatted using gofmt for consistency.
- Check for and remove unused code, variables, and imports.
- Profile performance (CPU, memory) if optimizing for speed or resource usage.
- Review for idiomatic Go patterns and best practices.
- Check for proper error handling and avoid swallowing errors.
- Ensure concurrency safety (goroutines, channels, mutexes) if applicable.
- Validate that logging and observability are not degraded.
- Confirm that documentation and comments are updated if code changes affect them.
- [ ] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
