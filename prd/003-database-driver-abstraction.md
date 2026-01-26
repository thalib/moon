## Overview
- Implement dialect-agnostic database layer supporting PostgreSQL, MySQL, and SQLite
- Auto-detect database type from connection string at startup
- Provide unified interface for database operations across all supported dialects

## Requirements
- Create `internal/database/` package with driver abstraction
- Implement `Driver` interface with methods: Connect, Close, Exec, Query, QueryRow
- Create dialect-specific implementations: PostgresDriver, MySQLDriver, SQLiteDriver
- Auto-detect database type from connection string format
- Database type is fixed at startup and cannot change at runtime
- Handle connection pooling configuration
- Implement health check/ping functionality
- Use parameterized queries to prevent SQL injection
- Support transaction management

## Acceptance
- Successfully connects to PostgreSQL, MySQL, and SQLite databases
- Correct dialect is auto-detected from connection string
- Connection pooling works correctly
- Unit tests cover driver detection and connection logic
- Integration tests verify CRUD operations on each database type
