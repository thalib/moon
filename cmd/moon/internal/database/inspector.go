// Package database provides database abstraction and connection management.
package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// TableInfo contains information about a database table
type TableInfo struct {
	Name    string
	Columns []ColumnInfo
}

// ColumnInfo contains information about a table column
type ColumnInfo struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue *string
	IsPrimaryKey bool
	IsUnique     bool
}

// ListTables returns a list of all user tables in the database
// Excludes system tables and internal metadata tables
func (d *baseDriver) ListTables(ctx context.Context) ([]string, error) {
	var tables []string
	var query string

	switch d.dialect {
	case DialectSQLite:
		// SQLite: Query sqlite_master for user tables
		query = `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`

	case DialectMySQL:
		// MySQL: Query information_schema
		query = `SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE' ORDER BY table_name`

	case DialectPostgres:
		// PostgreSQL: Query information_schema
		query = `SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public' ORDER BY tablename`

	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", d.dialect)
	}

	rows, err := d.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}

// GetTableInfo retrieves detailed information about a table including its columns
func (d *baseDriver) GetTableInfo(ctx context.Context, tableName string) (*TableInfo, error) {
	// Validate table name to prevent SQL injection
	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	var columns []ColumnInfo
	var query string
	var args []any

	switch d.dialect {
	case DialectSQLite:
		// SQLite: Use PRAGMA table_info (safe after validation)
		query = fmt.Sprintf("PRAGMA table_info(%s)", tableName)

	case DialectMySQL:
		// MySQL: Query information_schema.columns
		query = `SELECT column_name, data_type, is_nullable, column_default, column_key 
		         FROM information_schema.columns 
		         WHERE table_schema = DATABASE() AND table_name = ? 
		         ORDER BY ordinal_position`
		args = []any{tableName}

	case DialectPostgres:
		// PostgreSQL: Query information_schema.columns
		query = `SELECT column_name, data_type, is_nullable, column_default,
		                CASE WHEN pk.constraint_type = 'PRIMARY KEY' THEN true ELSE false END as is_primary
		         FROM information_schema.columns c
		         LEFT JOIN (
		             SELECT kcu.column_name, tc.constraint_type
		             FROM information_schema.table_constraints tc
		             JOIN information_schema.key_column_usage kcu 
		                 ON tc.constraint_name = kcu.constraint_name
		             WHERE tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY'
		         ) pk ON c.column_name = pk.column_name
		         WHERE c.table_name = $1
		         ORDER BY c.ordinal_position`
		args = []any{tableName}

	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", d.dialect)
	}

	rows, err := d.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query table info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var err error

		switch d.dialect {
		case DialectSQLite:
			// SQLite PRAGMA returns: cid, name, type, notnull, dflt_value, pk
			var cid int
			var notNull int
			var pk int
			var dfltValue *string

			err = rows.Scan(&cid, &col.Name, &col.Type, &notNull, &dfltValue, &pk)
			if err == nil {
				col.Nullable = notNull == 0
				col.DefaultValue = dfltValue
				col.IsPrimaryKey = pk > 0
			}

		case DialectMySQL:
			var dataType string
			var isNullable string
			var columnDefault *string
			var columnKey string

			err = rows.Scan(&col.Name, &dataType, &isNullable, &columnDefault, &columnKey)
			if err == nil {
				col.Type = dataType
				col.Nullable = isNullable == "YES"
				col.DefaultValue = columnDefault
				col.IsPrimaryKey = columnKey == "PRI"
				col.IsUnique = columnKey == "UNI"
			}

		case DialectPostgres:
			var dataType string
			var isNullable string
			var columnDefault *string
			var isPrimary bool

			err = rows.Scan(&col.Name, &dataType, &isNullable, &columnDefault, &isPrimary)
			if err == nil {
				col.Type = dataType
				col.Nullable = isNullable == "YES"
				col.DefaultValue = columnDefault
				col.IsPrimaryKey = isPrimary
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	return &TableInfo{
		Name:    tableName,
		Columns: columns,
	}, nil
}

// TableExists checks if a table exists in the database
func (d *baseDriver) TableExists(ctx context.Context, tableName string) (bool, error) {
	tables, err := d.ListTables(ctx)
	if err != nil {
		return false, err
	}

	for _, table := range tables {
		if table == tableName {
			return true, nil
		}
	}

	return false, nil
}

// InferColumnType attempts to map a database column type to a registry.ColumnType
func InferColumnType(dbType string) registry.ColumnType {
	dbTypeLower := strings.ToLower(dbType)

	// Integer types
	if strings.Contains(dbTypeLower, "int") || strings.Contains(dbTypeLower, "serial") {
		return registry.TypeInteger
	}

	// Float types - map to integer (per PRD 041, float type removed)
	if strings.Contains(dbTypeLower, "float") || strings.Contains(dbTypeLower, "double") ||
		strings.Contains(dbTypeLower, "real") || strings.Contains(dbTypeLower, "decimal") ||
		strings.Contains(dbTypeLower, "numeric") {
		return registry.TypeInteger
	}

	// Boolean types
	if strings.Contains(dbTypeLower, "bool") {
		return registry.TypeBoolean
	}

	// Date/time types
	if strings.Contains(dbTypeLower, "date") || strings.Contains(dbTypeLower, "time") ||
		strings.Contains(dbTypeLower, "timestamp") {
		return registry.TypeDatetime
	}

	// JSON types
	if strings.Contains(dbTypeLower, "json") {
		return registry.TypeJSON
	}

	// All text types map to string
	// Default to string for varchar, char, text, clob, blob, and unknown types
	return registry.TypeString
}

// isValidIdentifier validates that an identifier (table/column name) contains only safe characters
func isValidIdentifier(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}

	for i, ch := range name {
		if i == 0 {
			// First character must be letter or underscore
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
				return false
			}
		} else {
			// Subsequent characters can be alphanumeric or underscore
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
				return false
			}
		}
	}

	return true
}
