## Overview
- Implement schema management endpoints under `/collections` prefix
- Enable migration-less database management via REST API
- Support create, update, list, get, and destroy operations for database tables

## Requirements
- `GET /collections:list` - List all managed collections from cache
- `GET /collections:get` - Retrieve schema (fields/types) for one collection
- `POST /collections:create` - Create new table in database
- `POST /collections:update` - Modify table columns (add/remove/rename)
- `POST /collections:destroy` - Drop table and purge from cache
- Validate collection names (alphanumeric, no reserved words)
- Support column types: string, integer, float, boolean, datetime, text, json
- Support column constraints: nullable, unique, default values
- Update In-Memory Registry after each schema change
- Generate parameterized DDL statements per database dialect

## Acceptance
- All five endpoints respond correctly with proper HTTP status codes
- Schema changes reflect immediately in database and cache
- Invalid requests return descriptive error messages
- Unit tests cover validation and SQL generation
- Integration tests verify end-to-end schema operations
