// Package query provides SQL query building functionality with dialect-aware
// query generation. It supports PostgreSQL, MySQL, and SQLite dialects with
// proper identifier escaping and parameterized queries.
package query

import (
	"fmt"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// Operator constants for safe SQL operations
const (
	OpEqual              = "="
	OpNotEqual           = "!="
	OpGreaterThan        = ">"
	OpLessThan           = "<"
	OpGreaterThanOrEqual = ">="
	OpLessThanOrEqual    = "<="
	OpLike               = "LIKE"
	OpIn                 = "IN"
)

// validOperators contains all supported SQL operators
var validOperators = map[string]bool{
	OpEqual:              true,
	OpNotEqual:           true,
	OpGreaterThan:        true,
	OpLessThan:           true,
	OpGreaterThanOrEqual: true,
	OpLessThanOrEqual:    true,
	OpLike:               true,
	OpIn:                 true,
}

// ValidateOperator checks if an operator is valid and safe to use
func ValidateOperator(op string) error {
	if !validOperators[op] {
		return fmt.Errorf("invalid operator: %s", op)
	}
	return nil
}

// Builder provides methods for building SQL queries
type Builder interface {
	// DDL operations
	CreateTable(name string, columns []registry.Column) string
	AlterTableAddColumn(tableName string, column registry.Column) string
	DropTable(name string) string

	// DML operations
	Select(tableName string, columns []string, where []Condition, orderBy string, limit, offset int) (string, []any)
	Insert(tableName string, columns []string, values []any) (string, []any)
	Update(tableName string, updates map[string]any, where []Condition) (string, []any)
	Delete(tableName string, where []Condition) (string, []any)

	// Dialect returns the database dialect
	Dialect() database.DialectType
}

// Condition represents a WHERE clause condition
type Condition struct {
	Column   string
	Operator string
	Value    any
}

// builder implements Builder interface
type builder struct {
	dialect database.DialectType
}

// NewBuilder creates a new query builder for the specified dialect
func NewBuilder(dialect database.DialectType) Builder {
	return &builder{
		dialect: dialect,
	}
}

// Dialect returns the database dialect
func (b *builder) Dialect() database.DialectType {
	return b.dialect
}

// CreateTable generates CREATE TABLE DDL
func (b *builder) CreateTable(tableName string, columns []registry.Column) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (", b.escapeIdentifier(tableName)))

	// Add id column (auto-increment primary key)
	switch b.dialect {
	case database.DialectPostgres:
		sb.WriteString("\n  id SERIAL PRIMARY KEY")
	case database.DialectMySQL:
		sb.WriteString("\n  id INT AUTO_INCREMENT PRIMARY KEY")
	case database.DialectSQLite:
		sb.WriteString("\n  id INTEGER PRIMARY KEY AUTOINCREMENT")
	}

	// Add user-defined columns
	for _, col := range columns {
		sb.WriteString(",\n  ")
		sb.WriteString(b.escapeIdentifier(col.Name))
		sb.WriteString(" ")
		sb.WriteString(b.mapColumnTypeToSQL(col.Type))

		if !col.Nullable {
			sb.WriteString(" NOT NULL")
		}

		if col.Unique {
			sb.WriteString(" UNIQUE")
		}

		if col.DefaultValue != nil {
			sb.WriteString(" DEFAULT ")
			sb.WriteString(*col.DefaultValue)
		}
	}

	sb.WriteString("\n)")

	return sb.String()
}

// AlterTableAddColumn generates ALTER TABLE ADD COLUMN DDL
func (b *builder) AlterTableAddColumn(tableName string, column registry.Column) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
		b.escapeIdentifier(tableName),
		b.escapeIdentifier(column.Name),
		b.mapColumnTypeToSQL(column.Type)))

	if !column.Nullable {
		sb.WriteString(" NOT NULL")
	}

	if column.Unique {
		sb.WriteString(" UNIQUE")
	}

	if column.DefaultValue != nil {
		sb.WriteString(" DEFAULT ")
		sb.WriteString(*column.DefaultValue)
	}

	return sb.String()
}

// DropTable generates DROP TABLE DDL
func (b *builder) DropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE %s", b.escapeIdentifier(tableName))
}

// Select generates SELECT query with optional WHERE, ORDER BY, LIMIT, OFFSET
func (b *builder) Select(tableName string, columns []string, where []Condition, orderBy string, limit, offset int) (string, []any) {
	var sb strings.Builder
	args := []any{}

	// SELECT clause
	sb.WriteString("SELECT ")
	if len(columns) == 0 {
		sb.WriteString("*")
	} else {
		for i, col := range columns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(b.escapeIdentifier(col))
		}
	}

	// FROM clause
	sb.WriteString(" FROM ")
	sb.WriteString(b.escapeIdentifier(tableName))

	// WHERE clause
	args = b.buildWhereClause(&sb, where, args)

	// ORDER BY clause
	if orderBy != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(orderBy)
	}

	// LIMIT and OFFSET
	if limit > 0 {
		sb.WriteString(" LIMIT ")
		sb.WriteString(b.placeholder(len(args) + 1))
		args = append(args, limit)

		if offset > 0 {
			sb.WriteString(" OFFSET ")
			sb.WriteString(b.placeholder(len(args) + 1))
			args = append(args, offset)
		}
	}

	return sb.String(), args
}

// Insert generates INSERT query
func (b *builder) Insert(tableName string, columns []string, values []any) (string, []any) {
	var sb strings.Builder

	sb.WriteString("INSERT INTO ")
	sb.WriteString(b.escapeIdentifier(tableName))
	sb.WriteString(" (")

	// Column names
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(b.escapeIdentifier(col))
	}

	sb.WriteString(") VALUES (")

	// Value placeholders
	for i := range values {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(b.placeholder(i + 1))
	}

	sb.WriteString(")")

	return sb.String(), values
}

// Update generates UPDATE query
func (b *builder) Update(tableName string, updates map[string]any, where []Condition) (string, []any) {
	var sb strings.Builder
	args := []any{}

	sb.WriteString("UPDATE ")
	sb.WriteString(b.escapeIdentifier(tableName))
	sb.WriteString(" SET ")

	// SET clause
	i := 0
	for col, val := range updates {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(b.escapeIdentifier(col))
		sb.WriteString(" = ")
		sb.WriteString(b.placeholder(len(args) + 1))
		args = append(args, val)
		i++
	}

	// WHERE clause
	args = b.buildWhereClause(&sb, where, args)

	return sb.String(), args
}

// Delete generates DELETE query
func (b *builder) Delete(tableName string, where []Condition) (string, []any) {
	var sb strings.Builder
	args := []any{}

	sb.WriteString("DELETE FROM ")
	sb.WriteString(b.escapeIdentifier(tableName))

	// WHERE clause
	args = b.buildWhereClause(&sb, where, args)

	return sb.String(), args
}

// escapeIdentifier escapes table/column names based on dialect
func (b *builder) escapeIdentifier(name string) string {
	switch b.dialect {
	case database.DialectPostgres:
		// PostgreSQL uses double quotes
		return fmt.Sprintf(`"%s"`, name)
	case database.DialectMySQL:
		// MySQL uses backticks
		return fmt.Sprintf("`%s`", name)
	case database.DialectSQLite:
		// SQLite supports double quotes but also works without
		return name
	default:
		return name
	}
}

// placeholder returns the appropriate placeholder for parameterized queries
func (b *builder) placeholder(position int) string {
	switch b.dialect {
	case database.DialectPostgres:
		return fmt.Sprintf("$%d", position)
	case database.DialectMySQL, database.DialectSQLite:
		return "?"
	default:
		return "?"
	}
}

// escapeLikeValue escapes special characters in LIKE patterns to prevent unintended wildcard matches
func (b *builder) escapeLikeValue(value any) any {
	str, ok := value.(string)
	if !ok {
		return value
	}

	// Escape LIKE wildcards: % and _
	// Use backslash as escape character (standard SQL)
	str = strings.ReplaceAll(str, `\`, `\\`)
	str = strings.ReplaceAll(str, `%`, `\%`)
	str = strings.ReplaceAll(str, `_`, `\_`)

	return str
}

// buildWhereClause builds a WHERE clause from conditions and returns the updated args slice
func (b *builder) buildWhereClause(sb *strings.Builder, where []Condition, args []any) []any {
	if len(where) == 0 {
		return args
	}

	sb.WriteString(" WHERE ")
	for i, cond := range where {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString(b.escapeIdentifier(cond.Column))
		sb.WriteString(" ")
		sb.WriteString(cond.Operator)
		sb.WriteString(" ")

		// Handle special operators
		if cond.Operator == OpIn {
			// IN operator expects a slice of values
			values, ok := cond.Value.([]any)
			if !ok {
				// If not a slice, treat as single value
				values = []any{cond.Value}
			}
			sb.WriteString("(")
			for j, v := range values {
				if j > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(b.placeholder(len(args) + 1))
				args = append(args, v)
			}
			sb.WriteString(")")
		} else if cond.Operator == OpLike {
			// LIKE operator - escape special characters in value
			sb.WriteString(b.placeholder(len(args) + 1))
			args = append(args, b.escapeLikeValue(cond.Value))
		} else {
			// Standard operators
			sb.WriteString(b.placeholder(len(args) + 1))
			args = append(args, cond.Value)
		}
	}

	return args
}

// mapColumnTypeToSQL maps ColumnType to SQL type for the dialect
func (b *builder) mapColumnTypeToSQL(colType registry.ColumnType) string {
	switch b.dialect {
	case database.DialectPostgres:
		return mapColumnTypeToPostgres(colType)
	case database.DialectMySQL:
		return mapColumnTypeToMySQL(colType)
	case database.DialectSQLite:
		return mapColumnTypeToSQLite(colType)
	default:
		return "TEXT"
	}
}

func mapColumnTypeToPostgres(colType registry.ColumnType) string {
	switch colType {
	case registry.TypeString:
		return "VARCHAR(255)"
	case registry.TypeInteger:
		return "INTEGER"
	case registry.TypeFloat:
		return "DOUBLE PRECISION"
	case registry.TypeBoolean:
		return "BOOLEAN"
	case registry.TypeDatetime:
		return "TIMESTAMP"
	case registry.TypeText:
		return "TEXT"
	case registry.TypeJSON:
		return "JSONB"
	default:
		return "TEXT"
	}
}

func mapColumnTypeToMySQL(colType registry.ColumnType) string {
	switch colType {
	case registry.TypeString:
		return "VARCHAR(255)"
	case registry.TypeInteger:
		return "INT"
	case registry.TypeFloat:
		return "DOUBLE"
	case registry.TypeBoolean:
		return "BOOLEAN"
	case registry.TypeDatetime:
		return "DATETIME"
	case registry.TypeText:
		return "TEXT"
	case registry.TypeJSON:
		return "JSON"
	default:
		return "TEXT"
	}
}

func mapColumnTypeToSQLite(colType registry.ColumnType) string {
	switch colType {
	case registry.TypeString:
		return "TEXT"
	case registry.TypeInteger:
		return "INTEGER"
	case registry.TypeFloat:
		return "REAL"
	case registry.TypeBoolean:
		return "INTEGER"
	case registry.TypeDatetime:
		return "TEXT"
	case registry.TypeText:
		return "TEXT"
	case registry.TypeJSON:
		return "TEXT"
	default:
		return "TEXT"
	}
}
