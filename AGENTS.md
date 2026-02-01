# AGENTS.md

## Project Principles

- Ignore backwards compatibility and legacy concerns. This is a greenfield project for a single user—make any changes needed.
- Never introduce new compilation warnings. Fix any that appear.

## Mandatory Rules

- **SPEC.md is the only source of truth.**
  - Follow its architecture, configuration, and operational details exactly.
- Do not invent patterns or workflows not present in SPEC.md.
- Never use or reference content from `docs/` or `example/` for production.
- Flag missing information and unsupported assumptions.
- Be skeptical by default; state uncertainty clearly.
- Consider unconventional options, risks, and patterns when useful.
- Prefer simple, single-concern, untangled, and objective solutions.
- Invest in simplicity up front; process cannot fix complex designs.
- Design for human limits: keep components small and independent.
- Use only the Go standard library unless a third-party dependency is absolutely essential.

## SPEC.md Compliance

Strictly follow all guidelines and structures in SPEC.md for every task.

## Database Default

SQLite is the default database. For most development and testing, no connection string is needed unless using Postgres or MySQL.

## Best Practices

- Follow idiomatic Go and industry best practices.
- Research as needed; use MCP servers (context7) for up-to-date documentation.
- Keep code, configuration, and docs lean, simple, and clean.
- Avoid unnecessary complexity and duplication.
- **Do not include commands unless absolutely necessary for context.**
- **Test-Driven Development (TDD) is required:**
  - Every feature, bugfix, or refactor must have one or more unit tests before implementation.
  - All major logic modules must have corresponding `*_test.go` files.
  - No code is complete or production-ready without passing tests, as enforced in SPEC.md.
  - Installation and usage docs go in `INSTALL.md`, not `README.md`.
  - Keep `README.md` focused on project overview and features only.

## Workflow & Verification

- Plan: List minimal steps; note risks and edge cases.
- Patch: Make small, focused diffs; exclude unrelated changes.
- Test: Run tests with timeout; fix failures; add or update only minimal tests to cover new logic.
- Decompose: Split work into small, reviewable steps/commits.
- Double-check: Re-evaluate logic and trade-offs before finalizing.
- Verify: Briefly note how you validated; optionally record trade-offs and related follow-ups.
- If uncertain: Ask clarifying questions. If you must proceed, choose the conservative/simple path and state assumptions in the Task Summary.

## Code Quality & Style

- Keep code readable and easy to extend; follow project style.
- Use clear names; avoid magic values; extract constants when helpful.
- Keep functions small and single-purpose.
- Prefer the simplest working solution over cleverness.
- Add abstractions only when necessary.
- Fail fast; do not swallow errors—return or raise explicit, contextual errors.
- Handle errors and edge cases. No TODOs, dead code, or partial fixes.
- Aim for 90% test coverage
- Unless explicitly asked to create new documentation you should never create new documentation files
- Always keep the existing documentation and scripts in sync with code changes:
  - Documentation: `INSTALL.md`, `README.md`, `SPEC.md`, `AGENTS.md`
  - Scripts: `install.sh`, `build.sh`, `scripts/*`
  - Configuration: `samples/*`

### Format All Go Files

Format all Go files in the project using `gofmt` for consistent style:

1. Find all `*.go` files (including subdirectories).
2. Run `gofmt -w` on each file.
3. Fix any remaining formatting issues if needed.
