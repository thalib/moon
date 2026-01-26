## Overview
- Initialize the Go project structure following idiomatic Go conventions and the Moon architecture
- Set up the foundational directory layout including cmd, internal, pkg directories
- Configure Go modules with proper dependency management

## Requirements
- Create `cmd/moon/main.go` as the application entry point
- Create `internal/` directory for private application code
- Create `pkg/` directory for reusable public packages (if needed)
- Initialize `go.mod` with appropriate module name
- Set up `.gitignore` for Go projects
- Create placeholder directories for: config, handlers, middleware, registry, database, models

## Acceptance
- Project compiles with `go build ./...`
- Directory structure follows Go community standards
- `go mod tidy` runs without errors
- Basic `main.go` runs and exits cleanly
