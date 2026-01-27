package query

import (
"strings"
"testing"

"github.com/thalib/moon/internal/database"
"github.com/thalib/moon/internal/registry"
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
