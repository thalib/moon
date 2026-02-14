# Bug Report: Adding a UNIQUE column fails on SQLite/PostgreSQL in `/collections:update`

## Summary

When using the `/collections:update` API endpoint to add a new column with `unique: true`, the operation fails on SQLite and PostgreSQL databases. Both return errors because the code attempts to add a UNIQUE column in a way not supported by those backends.

## Severity

- **High:** Prevents schema evolution via API on affected databases; disrupts e-commerce/data extensibility.

## Steps to Reproduce

1. Set up a Moon backend using SQLite or PostgreSQL.
2. Create a collection called `products`:
   ```json
   {
     "name": "products",
     "columns": [
       { "name": "title", "type": "string", "nullable": false }
     ]
   }
   ```
3. Call `/collections:update` with a request to add a unique column:
   ```json
   {
     "name": "products",
     "add_columns": [
       { "name": "slug", "type": "string", "nullable": false, "unique": true }
     ]
   }
   ```
4. Observe the response:
   - SQLite:  
     ```json
     { "code":500,"error":"failed to add column 'slug': SQL logic error: Cannot add a UNIQUE column (1)" }
     ```
   - PostgreSQL:  
     Error indicating syntax or unsupported request.

## Actual Behavior

- Server attempts to run a SQL statement:
  ```
  ALTER TABLE products ADD COLUMN slug TEXT NOT NULL UNIQUE
  ```
- This fails on:
  - **SQLite:** Does not support adding a UNIQUE column directly via ALTER TABLE.
  - **PostgreSQL:** Does not allow adding unique constraints in ADD COLUMN statement (requires separate constraint).

## Expected Behavior

- The API should succeed on all supported database backends.
- After the call, the new column (`slug`) should exist and enforce uniqueness as expected, regardless of backend.

## Root Cause

- The DDL generation code (`generateAddColumnDDL` in `collections.go`) produces a statement like:
  ```
  ALTER TABLE ... ADD COLUMN ... UNIQUE
  ```
  This is not portable:
  - **SQLite:** Fails ("Cannot add a UNIQUE column").
  - **PostgreSQL:** Requires `ALTER TABLE ... ADD COLUMN`, followed by `ALTER TABLE ... ADD CONSTRAINT ... UNIQUE(...)`.
  - **MySQL/MariaDB:** May succeed, but is not portable.

## Affected Modules / Files

- `cmd/moon/internal/handlers/collections.go`  
  - `generateAddColumnDDL`
  - `/collections:update` handler

## Solution

**Portable implementation:**
1. Add the column WITHOUT any unique constraint:
   ```
   ALTER TABLE products ADD COLUMN slug TEXT NOT NULL
   ```
2. Add the unique constraint/index separately:
   - **PostgreSQL, MySQL, MariaDB:**
     ```
     ALTER TABLE products ADD CONSTRAINT products_slug_unique UNIQUE(slug)
     ```
   - **SQLite:**
     ```
     CREATE UNIQUE INDEX idx_products_slug ON products(slug)
     ```

**Key Steps:**
- Detect target dialect.
- After adding column, run a second DDL statement for uniqueness.
- Update registry only after both succeed.

## Additional Considerations

- If existing rows are present, adding a NOT NULL and UNIQUE column will fail unless you specify a default value or allow nullable.
- Rollback/cleanup: Clean up partial index/constraint on error.
- Update integration tests to verify correct DDL and schema evolution on all supported DBs.

## References

- [SQLite ALTER TABLE docs](https://sqlite.org/lang_altertable.html)
- [PostgreSQL ALTER TABLE docs](https://www.postgresql.org/docs/current/sql-altertable.html)
- [Moon code: generateAddColumnDDL](https://github.com/devnodesin/moon/blob/ea444edb2717262e780262caf319c51892e9b65f/cmd/moon/internal/handlers/collections.go#L1115-L1143)

## Suggested Patch

- Refactor `generateAddColumnDDL` and `/collections:update` logic:
  - Remove `UNIQUE` from add column statement.
  - Add dialect-specific code to create the unique constraint/index after column addition.

---

**This issue prevents universal database compatibility for Moon’s schema management API. Patching as described enables portable, standards-compliant, and future-proof schema evolution.**
