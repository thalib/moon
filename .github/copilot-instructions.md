# Copilot Instructions for AI Agents

## IMPORTANT: Must Follow

- **Source of Truth:** Always refer to `SPEC.md` for all architecture, configuration, and operational details.
- Do not introduce patterns or workflows not present in `SPEC.md`.
- Never reference or use content from `docs/` or `example/` directories for production operations.

## Follow the Spec (`SPEC.md`) Exactly

When performing tasks, strictly adhere to the guidelines and structures defined in `SPEC.md`.

For all operations, consult these sections in `SPEC.md`:

## Best Practices

- Follow industry best practices for Go.
- Research the internet and use MCP servers (context7) for latest documentation.
- Keep all code, configuration, and documentation lean, simple, and clean.
- Avoid unnecessary complexity and duplication.
- **DO NOT** include commands unless very necessary for context.
- **Test-Driven Development (TDD) is mandatory:**
  - Every feature, bugfix, or refactor must be accompanied by one or more unit tests before implementation.
  - All major logic modules require corresponding `*_test.go` files.
  - No code is considered complete or production-ready without passing tests, as enforced in `SPEC.md`.
  - Documentation related to installation and usage must be included in `docs/INSTALL.md`, not in `README.md`.
  - Keep the `README.md` focused on the project overview and features.
