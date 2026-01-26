## Overview
- Implement dynamic SQL query builder for generating sanitized, parameterized statements
- Support multiple database dialects (PostgreSQL, MySQL, SQLite)
- Generate both DDL (schema) and DML (data) statements

## Requirements
- Create `internal/query/builder.go` with QueryBuilder interface
- Support DDL operations: CREATE TABLE, ALTER TABLE, DROP TABLE
- Support DML operations: SELECT, INSERT, UPDATE, DELETE
- Generate dialect-specific syntax (e.g., SERIAL vs AUTO_INCREMENT)
- Always use parameterized queries (never string interpolation for values)
- Support WHERE clause building with multiple conditions
- Support ORDER BY, LIMIT, OFFSET for SELECT queries
- Escape identifiers (table/column names) properly per dialect
- Handle NULL values correctly

## Acceptance
- Generated SQL is valid for each supported database
- All queries use parameterized placeholders
- SQL injection attacks are prevented
- Unit tests cover query generation for all operations and dialects
- Integration tests verify generated queries execute correctly
