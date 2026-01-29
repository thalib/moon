package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	if builder == nil {
		t.Fatal("expected builder to be created")
	}
	if builder.Dialect() != database.DialectPostgres {
		t.Errorf("expected dialect %s, got %s", database.DialectPostgres, builder.Dialect())
	}
}

func TestCreateTable_PostgreSQL(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	columns := []registry.Column{
		{Name: "name", Type: registry.TypeString, Nullable: false},
		{Name: "price", Type: registry.TypeFloat, Nullable: false},
	}

	sql := builder.CreateTable("products", columns)

	if !strings.Contains(sql, "CREATE TABLE") {
		t.Error("expected CREATE TABLE clause")
	}
	if !strings.Contains(sql, "id SERIAL PRIMARY KEY") {
		t.Error("expected SERIAL primary key for PostgreSQL")
	}
	if !strings.Contains(sql, "\"name\"") {
		t.Error("expected quoted column name")
	}
	if !strings.Contains(sql, "VARCHAR(255)") {
		t.Error("expected VARCHAR type")
	}
	if !strings.Contains(sql, "NOT NULL") {
		t.Error("expected NOT NULL constraint")
	}
}

func TestCreateTable_MySQL(t *testing.T) {
	builder := NewBuilder(database.DialectMySQL)
	columns := []registry.Column{
		{Name: "name", Type: registry.TypeString},
	}

	sql := builder.CreateTable("products", columns)

	if !strings.Contains(sql, "AUTO_INCREMENT PRIMARY KEY") {
		t.Error("expected AUTO_INCREMENT for MySQL")
	}
	if !strings.Contains(sql, "`name`") {
		t.Error("expected backtick-quoted column name for MySQL")
	}
}

func TestCreateTable_SQLite(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	columns := []registry.Column{
		{Name: "count", Type: registry.TypeInteger},
	}

	sql := builder.CreateTable("stats", columns)

	if !strings.Contains(sql, "AUTOINCREMENT") {
		t.Error("expected AUTOINCREMENT for SQLite")
	}
	if !strings.Contains(sql, "INTEGER") {
		t.Error("expected INTEGER type")
	}
}

func TestAlterTableAddColumn(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	column := registry.Column{
		Name:     "email",
		Type:     registry.TypeString,
		Nullable: false,
		Unique:   true,
	}

	sql := builder.AlterTableAddColumn("users", column)

	if !strings.Contains(sql, "ALTER TABLE") {
		t.Error("expected ALTER TABLE clause")
	}
	if !strings.Contains(sql, "ADD COLUMN") {
		t.Error("expected ADD COLUMN clause")
	}
	if !strings.Contains(sql, "UNIQUE") {
		t.Error("expected UNIQUE constraint")
	}
}

func TestDropTable(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	sql := builder.DropTable("old_table")

	if sql != "DROP TABLE old_table" {
		t.Errorf("expected 'DROP TABLE old_table', got '%s'", sql)
	}
}

func TestSelect_Simple(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	sql, args := builder.Select("products", nil, nil, "", 0, 0)

	expected := "SELECT * FROM products"
	if sql != expected {
		t.Errorf("expected '%s', got '%s'", expected, sql)
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

func TestSelect_WithColumns(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	sql, _ := builder.Select("users", []string{"id", "name", "email"}, nil, "", 0, 0)

	if !strings.Contains(sql, `"id"`) || !strings.Contains(sql, `"name"`) || !strings.Contains(sql, `"email"`) {
		t.Error("expected column names in SELECT")
	}
}

func TestSelect_WithWhere(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	where := []Condition{
		{Column: "status", Operator: "=", Value: "active"},
		{Column: "age", Operator: ">", Value: 18},
	}

	sql, args := builder.Select("users", nil, where, "", 0, 0)

	if !strings.Contains(sql, "WHERE") {
		t.Error("expected WHERE clause")
	}
	if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") {
		t.Error("expected PostgreSQL placeholders")
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
	if args[0] != "active" || args[1] != 18 {
		t.Error("unexpected argument values")
	}
}

func TestSelect_WithOrderByLimitOffset(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	sql, args := builder.Select("products", nil, nil, "price DESC", 10, 20)

	if !strings.Contains(sql, "ORDER BY price DESC") {
		t.Error("expected ORDER BY clause")
	}
	if !strings.Contains(sql, "LIMIT") {
		t.Error("expected LIMIT clause")
	}
	if !strings.Contains(sql, "OFFSET") {
		t.Error("expected OFFSET clause")
	}
	if len(args) != 2 || args[0] != 10 || args[1] != 20 {
		t.Error("unexpected limit/offset values")
	}
}

func TestInsert(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	columns := []string{"name", "price"}
	values := []any{"Product", 19.99}

	sql, args := builder.Insert("products", columns, values)

	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("expected INSERT INTO clause")
	}
	if !strings.Contains(sql, "VALUES") {
		t.Error("expected VALUES clause")
	}
	if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") {
		t.Error("expected PostgreSQL placeholders")
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

func TestUpdate(t *testing.T) {
	builder := NewBuilder(database.DialectMySQL)
	updates := map[string]any{
		"name":  "Updated Name",
		"price": 29.99,
	}
	where := []Condition{
		{Column: "id", Operator: "=", Value: 42},
	}

	sql, args := builder.Update("products", updates, where)

	if !strings.Contains(sql, "UPDATE") {
		t.Error("expected UPDATE clause")
	}
	if !strings.Contains(sql, "SET") {
		t.Error("expected SET clause")
	}
	if !strings.Contains(sql, "WHERE") {
		t.Error("expected WHERE clause")
	}
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
}

func TestDelete(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	where := []Condition{
		{Column: "id", Operator: "=", Value: 42},
	}

	sql, args := builder.Delete("products", where)

	expected := "DELETE FROM products WHERE id = ?"
	if sql != expected {
		t.Errorf("expected '%s', got '%s'", expected, sql)
	}
	if len(args) != 1 || args[0] != 42 {
		t.Error("unexpected arguments")
	}
}

func TestPlaceholder_Postgres(t *testing.T) {
	b := &builder{dialect: database.DialectPostgres}
	if p := b.placeholder(1); p != "$1" {
		t.Errorf("expected $1, got %s", p)
	}
	if p := b.placeholder(10); p != "$10" {
		t.Errorf("expected $10, got %s", p)
	}
}

func TestPlaceholder_MySQL(t *testing.T) {
	b := &builder{dialect: database.DialectMySQL}
	if p := b.placeholder(1); p != "?" {
		t.Errorf("expected ?, got %s", p)
	}
}

func TestEscapeIdentifier(t *testing.T) {
	tests := []struct {
		dialect  database.DialectType
		input    string
		expected string
	}{
		{database.DialectPostgres, "user_name", `"user_name"`},
		{database.DialectMySQL, "user_name", "`user_name`"},
		{database.DialectSQLite, "user_name", "user_name"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dialect), func(t *testing.T) {
			b := &builder{dialect: tt.dialect}
			result := b.escapeIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestMapColumnTypes(t *testing.T) {
	tests := []struct {
		dialect      database.DialectType
		colType      registry.ColumnType
		expectedType string
	}{
		{database.DialectPostgres, registry.TypeString, "VARCHAR(255)"},
		{database.DialectPostgres, registry.TypeInteger, "INTEGER"},
		{database.DialectPostgres, registry.TypeFloat, "DOUBLE PRECISION"},
		{database.DialectPostgres, registry.TypeBoolean, "BOOLEAN"},
		{database.DialectPostgres, registry.TypeDatetime, "TIMESTAMP"},
		{database.DialectPostgres, registry.TypeJSON, "JSONB"},
		{database.DialectMySQL, registry.TypeString, "VARCHAR(255)"},
		{database.DialectMySQL, registry.TypeInteger, "INT"},
		{database.DialectMySQL, registry.TypeFloat, "DOUBLE"},
		{database.DialectMySQL, registry.TypeJSON, "JSON"},
		{database.DialectSQLite, registry.TypeString, "TEXT"},
		{database.DialectSQLite, registry.TypeInteger, "INTEGER"},
		{database.DialectSQLite, registry.TypeFloat, "REAL"},
		{database.DialectSQLite, registry.TypeBoolean, "INTEGER"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dialect)+"_"+string(tt.colType), func(t *testing.T) {
			b := &builder{dialect: tt.dialect}
			result := b.mapColumnTypeToSQL(tt.colType)
			if result != tt.expectedType {
				t.Errorf("expected '%s', got '%s'", tt.expectedType, result)
			}
		})
	}
}

func TestColumnConstraints(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	defaultVal := "test"
	columns := []registry.Column{
		{
			Name:         "status",
			Type:         registry.TypeString,
			Nullable:     false,
			Unique:       true,
			DefaultValue: &defaultVal,
		},
	}

	sql := builder.CreateTable("test", columns)

	if !strings.Contains(sql, "NOT NULL") {
		t.Error("expected NOT NULL constraint")
	}
	if !strings.Contains(sql, "UNIQUE") {
		t.Error("expected UNIQUE constraint")
	}
	if !strings.Contains(sql, "DEFAULT test") {
		t.Error("expected DEFAULT value")
	}
}

// Test SQL injection prevention via parameterized queries
func TestSQLInjectionPrevention(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)

	// Malicious input that would cause SQL injection if not parameterized
	maliciousValue := "'; DROP TABLE users; --"

	where := []Condition{
		{Column: "username", Operator: "=", Value: maliciousValue},
	}

	sql, args := builder.Select("users", nil, where, "", 0, 0)

	// The malicious string should be in args, not in the SQL string
	if strings.Contains(sql, "DROP TABLE") {
		t.Error("SQL injection vulnerability: malicious code in query string")
	}

	if len(args) == 0 || args[0] != maliciousValue {
		t.Error("expected malicious value to be safely parameterized")
	}

	// Verify we're using placeholders
	if !strings.Contains(sql, "$1") {
		t.Error("expected parameterized placeholder")
	}
}

// PRD 020: Query Builder Enhancements Tests

func TestValidateOperator(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		wantErr  bool
	}{
		{"Equal operator", OpEqual, false},
		{"Not equal operator", OpNotEqual, false},
		{"Greater than", OpGreaterThan, false},
		{"Less than", OpLessThan, false},
		{"Greater than or equal", OpGreaterThanOrEqual, false},
		{"Less than or equal", OpLessThanOrEqual, false},
		{"LIKE operator", OpLike, false},
		{"IN operator", OpIn, false},
		{"Invalid operator", "INVALID", true},
		{"SQL injection attempt", "; DROP TABLE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOperator(tt.operator)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOperator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSelect_ComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		value    any
		dialect  database.DialectType
	}{
		{"Equal", OpEqual, "value", database.DialectSQLite},
		{"Not equal", OpNotEqual, "value", database.DialectSQLite},
		{"Greater than", OpGreaterThan, 100, database.DialectSQLite},
		{"Less than", OpLessThan, 50, database.DialectSQLite},
		{"Greater than or equal", OpGreaterThanOrEqual, 75, database.DialectPostgres},
		{"Less than or equal", OpLessThanOrEqual, 25, database.DialectMySQL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.dialect)
			where := []Condition{
				{Column: "field", Operator: tt.operator, Value: tt.value},
			}

			sql, args := builder.Select("table", nil, where, "", 0, 0)

			if !strings.Contains(sql, "WHERE") {
				t.Error("expected WHERE clause")
			}
			if !strings.Contains(sql, tt.operator) {
				t.Errorf("expected operator %s in SQL", tt.operator)
			}
			if len(args) != 1 || args[0] != tt.value {
				t.Errorf("expected args [%v], got %v", tt.value, args)
			}
		})
	}
}

func TestSelect_LikeOperator(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		expectedValue string
		dialect       database.DialectType
	}{
		{
			name:          "Normal string",
			value:         "moon",
			expectedValue: "moon",
			dialect:       database.DialectSQLite,
		},
		{
			name:          "String with percent wildcard",
			value:         "test%value",
			expectedValue: `test\%value`,
			dialect:       database.DialectSQLite,
		},
		{
			name:          "String with underscore wildcard",
			value:         "test_value",
			expectedValue: `test\_value`,
			dialect:       database.DialectPostgres,
		},
		{
			name:          "String with both wildcards",
			value:         "test%_value",
			expectedValue: `test\%\_value`,
			dialect:       database.DialectMySQL,
		},
		{
			name:          "String with backslash",
			value:         `test\value`,
			expectedValue: `test\\value`,
			dialect:       database.DialectSQLite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.dialect)
			where := []Condition{
				{Column: "name", Operator: OpLike, Value: tt.value},
			}

			sql, args := builder.Select("table", nil, where, "", 0, 0)

			if !strings.Contains(sql, "WHERE") {
				t.Error("expected WHERE clause")
			}
			if !strings.Contains(sql, "LIKE") {
				t.Error("expected LIKE operator")
			}
			if len(args) != 1 {
				t.Errorf("expected 1 arg, got %d", len(args))
			}
			if args[0] != tt.expectedValue {
				t.Errorf("expected escaped value %q, got %q", tt.expectedValue, args[0])
			}
		})
	}
}

func TestSelect_InOperator(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		dialect database.DialectType
	}{
		{
			name:    "Multiple values",
			value:   []any{"a", "b", "c"},
			dialect: database.DialectSQLite,
		},
		{
			name:    "Single value as slice",
			value:   []any{"single"},
			dialect: database.DialectPostgres,
		},
		{
			name:    "Integer values",
			value:   []any{1, 2, 3, 4, 5},
			dialect: database.DialectMySQL,
		},
		{
			name:    "Mixed types",
			value:   []any{"string", 123, 45.67},
			dialect: database.DialectSQLite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.dialect)
			where := []Condition{
				{Column: "status", Operator: OpIn, Value: tt.value},
			}

			sql, args := builder.Select("table", nil, where, "", 0, 0)

			if !strings.Contains(sql, "WHERE") {
				t.Error("expected WHERE clause")
			}
			if !strings.Contains(sql, "IN") {
				t.Error("expected IN operator")
			}
			if !strings.Contains(sql, "(") || !strings.Contains(sql, ")") {
				t.Error("expected parentheses for IN clause")
			}

			values := tt.value.([]any)
			if len(args) != len(values) {
				t.Errorf("expected %d args, got %d", len(values), len(args))
			}

			// Verify correct number of placeholders
			switch tt.dialect {
			case database.DialectPostgres:
				for i := 1; i <= len(values); i++ {
					placeholder := fmt.Sprintf("$%d", i)
					if !strings.Contains(sql, placeholder) {
						t.Errorf("expected placeholder %s", placeholder)
					}
				}
			case database.DialectMySQL, database.DialectSQLite:
				expectedPlaceholders := strings.Repeat("?, ", len(values)-1) + "?"
				if !strings.Contains(sql, expectedPlaceholders) {
					t.Errorf("expected placeholders %s", expectedPlaceholders)
				}
			}
		})
	}
}

func TestSelect_InOperator_NonSliceValue(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	where := []Condition{
		{Column: "id", Operator: OpIn, Value: "single_value"},
	}

	sql, args := builder.Select("table", nil, where, "", 0, 0)

	if !strings.Contains(sql, "IN") {
		t.Error("expected IN operator")
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg for single value, got %d", len(args))
	}
	if args[0] != "single_value" {
		t.Errorf("expected 'single_value', got %v", args[0])
	}
}

func TestSelect_MultipleOperators(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	where := []Condition{
		{Column: "price", Operator: OpGreaterThan, Value: 100},
		{Column: "name", Operator: OpLike, Value: "product%"},
		{Column: "category", Operator: OpIn, Value: []any{"electronics", "gadgets"}},
		{Column: "status", Operator: OpEqual, Value: "active"},
	}

	sql, args := builder.Select("products", nil, where, "", 0, 0)

	if !strings.Contains(sql, "WHERE") {
		t.Error("expected WHERE clause")
	}
	if !strings.Contains(sql, ">") {
		t.Error("expected > operator")
	}
	if !strings.Contains(sql, "LIKE") {
		t.Error("expected LIKE operator")
	}
	if !strings.Contains(sql, "IN") {
		t.Error("expected IN operator")
	}
	if !strings.Contains(sql, "=") {
		t.Error("expected = operator")
	}

	// Should have 5 args: price, escaped LIKE value, 2 IN values, status
	if len(args) != 5 {
		t.Errorf("expected 5 args, got %d", len(args))
	}
}

func TestUpdate_WithSpecialOperators(t *testing.T) {
	builder := NewBuilder(database.DialectSQLite)
	updates := map[string]any{
		"name": "New Name",
	}
	where := []Condition{
		{Column: "status", Operator: OpIn, Value: []any{"pending", "active"}},
	}

	sql, args := builder.Update("table", updates, where)

	if !strings.Contains(sql, "UPDATE") {
		t.Error("expected UPDATE clause")
	}
	if !strings.Contains(sql, "WHERE") {
		t.Error("expected WHERE clause")
	}
	if !strings.Contains(sql, "IN") {
		t.Error("expected IN operator")
	}

	// 1 for SET clause, 2 for IN clause
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
}

func TestDelete_WithSpecialOperators(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	where := []Condition{
		{Column: "name", Operator: OpLike, Value: "test%"},
		{Column: "age", Operator: OpLessThan, Value: 18},
	}

	sql, args := builder.Delete("users", where)

	if !strings.Contains(sql, "DELETE FROM") {
		t.Error("expected DELETE FROM clause")
	}
	if !strings.Contains(sql, "WHERE") {
		t.Error("expected WHERE clause")
	}
	if !strings.Contains(sql, "LIKE") {
		t.Error("expected LIKE operator")
	}
	if !strings.Contains(sql, "<") {
		t.Error("expected < operator")
	}

	// Should have 2 args (escaped LIKE value and age)
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}

	// Verify LIKE value is escaped
	if !strings.Contains(args[0].(string), `\%`) {
		t.Error("expected escaped % in LIKE value")
	}
}

func TestDialect_SpecificBehavior(t *testing.T) {
	tests := []struct {
		name             string
		dialect          database.DialectType
		expectedFirstPH  string
		expectedSecondPH string
		expectedThirdPH  string
	}{
		{
			name:             "PostgreSQL placeholders",
			dialect:          database.DialectPostgres,
			expectedFirstPH:  "$1",
			expectedSecondPH: "$2",
			expectedThirdPH:  "$3",
		},
		{
			name:             "MySQL placeholders",
			dialect:          database.DialectMySQL,
			expectedFirstPH:  "?",
			expectedSecondPH: "?",
			expectedThirdPH:  "?",
		},
		{
			name:             "SQLite placeholders",
			dialect:          database.DialectSQLite,
			expectedFirstPH:  "?",
			expectedSecondPH: "?",
			expectedThirdPH:  "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.dialect)
			where := []Condition{
				{Column: "a", Operator: OpEqual, Value: 1},
				{Column: "b", Operator: OpGreaterThan, Value: 2},
				{Column: "c", Operator: OpLike, Value: "test"},
			}

			sql, args := builder.Select("table", nil, where, "", 0, 0)

			if !strings.Contains(sql, tt.expectedFirstPH) {
				t.Errorf("expected placeholder %s", tt.expectedFirstPH)
			}
			if len(args) != 3 {
				t.Errorf("expected 3 args, got %d", len(args))
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name       string
		dialect    database.DialectType
		table      string
		conditions []Condition
		wantSQL    string
		wantArgs   int
	}{
		{
			name:     "count all - postgres",
			dialect:  database.DialectPostgres,
			table:    "orders",
			wantSQL:  `SELECT COUNT(*) FROM "orders"`,
			wantArgs: 0,
		},
		{
			name:     "count all - sqlite",
			dialect:  database.DialectSQLite,
			table:    "orders",
			wantSQL:  "SELECT COUNT(*) FROM orders",
			wantArgs: 0,
		},
		{
			name:    "count with filter - postgres",
			dialect: database.DialectPostgres,
			table:   "orders",
			conditions: []Condition{
				{Column: "status", Operator: OpEqual, Value: "completed"},
			},
			wantSQL:  `SELECT COUNT(*) FROM "orders" WHERE "status" = $1`,
			wantArgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.dialect)
			sql, args := builder.Count(tt.table, tt.conditions)

			if sql != tt.wantSQL {
				t.Errorf("Count() sql = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != tt.wantArgs {
				t.Errorf("Count() args count = %d, want %d", len(args), tt.wantArgs)
			}
		})
	}
}

func TestSum(t *testing.T) {
	tests := []struct {
		name       string
		dialect    database.DialectType
		table      string
		field      string
		conditions []Condition
		wantSQL    string
		wantArgs   int
	}{
		{
			name:     "sum - postgres",
			dialect:  database.DialectPostgres,
			table:    "orders",
			field:    "total",
			wantSQL:  `SELECT SUM("total") FROM "orders"`,
			wantArgs: 0,
		},
		{
			name:     "sum - sqlite",
			dialect:  database.DialectSQLite,
			table:    "orders",
			field:    "total",
			wantSQL:  "SELECT SUM(total) FROM orders",
			wantArgs: 0,
		},
		{
			name:    "sum with filter - postgres",
			dialect: database.DialectPostgres,
			table:   "orders",
			field:   "total",
			conditions: []Condition{
				{Column: "status", Operator: OpEqual, Value: "completed"},
			},
			wantSQL:  `SELECT SUM("total") FROM "orders" WHERE "status" = $1`,
			wantArgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.dialect)
			sql, args := builder.Sum(tt.table, tt.field, tt.conditions)

			if sql != tt.wantSQL {
				t.Errorf("Sum() sql = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != tt.wantArgs {
				t.Errorf("Sum() args count = %d, want %d", len(args), tt.wantArgs)
			}
		})
	}
}

func TestAvg(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	sql, args := builder.Avg("orders", "total", nil)

	if !strings.Contains(sql, "SELECT AVG") {
		t.Error("expected AVG function")
	}
	if !strings.Contains(sql, `"total"`) {
		t.Error("expected quoted field name")
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

func TestMin(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	sql, args := builder.Min("orders", "total", nil)

	if !strings.Contains(sql, "SELECT MIN") {
		t.Error("expected MIN function")
	}
	if !strings.Contains(sql, `"total"`) {
		t.Error("expected quoted field name")
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

func TestMax(t *testing.T) {
	builder := NewBuilder(database.DialectPostgres)
	sql, args := builder.Max("orders", "total", nil)

	if !strings.Contains(sql, "SELECT MAX") {
		t.Error("expected MAX function")
	}
	if !strings.Contains(sql, `"total"`) {
		t.Error("expected quoted field name")
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}
