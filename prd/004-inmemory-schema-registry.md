## Overview
- Implement In-Memory Schema Registry using sync.Map for zero-latency validation
- Cache database schema (tables, columns, types) in RAM for nanosecond lookups
- Support reactive updates when schema changes occur

## Requirements
- Create `internal/registry/registry.go` with SchemaRegistry struct
- Use `sync.Map` for thread-safe concurrent access
- Store collection metadata: name, columns (name, type, constraints)
- Implement methods: Get, Set, Delete, List, Exists
- Load schema from database on startup (warm cache)
- Reactive refresh: update cache immediately after schema modifications
- Support column type mapping between Go types and database types
- Memory footprint optimization (target under 50MB total for app)

## Acceptance
- Schema lookups complete in nanoseconds (benchmark tests)
- Registry is thread-safe for concurrent read/write access
- Cache stays in sync with database schema
- Unit tests cover all registry operations
- Benchmark tests demonstrate performance characteristics
