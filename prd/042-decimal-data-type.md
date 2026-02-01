# PRD-042: Decimal Data Type

## Overview

The `Decimal` data type provides **exact, deterministic numeric handling** for precision-critical values such as price, amount, weight, tax, and quantity. This addresses the inherent precision errors in floating-point arithmetic while maintaining full SQL aggregation support across all database backends.

### Problem Statement

Floating-point types (`float`, `double`) are unsuitable for financial and measurement values due to rounding errors and non-deterministic behavior. Applications require exact arithmetic for monetary calculations, quantities, and measurements where precision loss is unacceptable.

### Solution

Introduce a `Decimal` type backed by Go's `math/big.Rat` that:

- Exposes values as **strings** via API (preserving API primitive simplicity)
- Stores values using native SQL numeric types
- Supports SQL aggregation functions (`SUM`, `AVG`, `MIN`, `MAX`)
- Provides exact, deterministic arithmetic
- Works consistently across SQLite, MySQL, PostgreSQL, and MariaDB

### Scope

**In Scope:**
- Exact numeric storage and retrieval
- SQL aggregation support
- String-based API representation
- Database-agnostic implementation
- Validation with configurable precision
- JSON serialization/deserialization

**Out of Scope:**
- Scientific notation support
- Locale-specific formatting
- Currency handling (separate concern)
- High-frequency scientific computation (use integer scaling instead)

---

## Requirements

### FR-1: Type Definition

**FR-1.1: Internal Representation**
- Type name: `Decimal`
- Implementation: Go `math/big.Rat` (standard library)
- No external dependencies required
- Provides arbitrary precision arithmetic

**FR-1.2: API Type Mapping**
- API input type: `string`
- API output type: `string`
- API primitive type list remains unchanged
- No new public API types introduced

### FR-2: Data Format and Validation

**FR-2.1: String Format**
- Valid formats:
  - `"10"`
  - `"10.50"`
  - `"1299.99"`
  - `"-42.75"`
  - `"0.01"`
- No scientific notation allowed
- No locale-specific separators

**FR-2.2: Precision Configuration**
- Default scale: **2 decimal places**
- Maximum scale: configurable via `constants` package
- Configuration constant: `MaxDecimalScale`
- Validation enforces configured scale limits

**FR-2.3: Validation Rules**
- Validate before conversion to `Decimal`
- Reject invalid inputs:
  - Non-numeric characters: `"abc"`
  - Scientific notation: `"1e10"`
  - Excess precision: `"10.999"` when scale = 2
  - Empty strings
  - Malformed decimals: `"10."`
- Validation occurs at:
  - API boundary (request unmarshaling)
  - Database write boundary

**FR-2.4: Validation Implementation**
- Use regex pattern OR precision/scale rule check
- Fail fast on invalid input
- Return clear validation error messages

### FR-3: Database Storage

**FR-3.1: Storage Types**

| Database        | Storage Type   | Notes                    |
|-----------------|----------------|--------------------------|
| SQLite          | `NUMERIC`      | Arbitrary precision      |
| MySQL           | `DECIMAL(p,s)` | p = precision, s = scale |
| MariaDB         | `DECIMAL(p,s)` | p = precision, s = scale |
| PostgreSQL      | `NUMERIC(p,s)` | p = precision, s = scale |

**FR-3.2: Storage Requirements**
- Never store as `FLOAT`, `REAL`, or `DOUBLE`
- Precision (p) must be explicit
- Scale (s) must be explicit
- Default: `DECIMAL(19,2)` for MySQL/MariaDB/PostgreSQL
- Use `NUMERIC` for SQLite (type affinity handles precision)

**FR-3.3: Schema Definition**
- Columns defined with explicit precision/scale
- Index decimal columns used in filtering or aggregation
- Support `NOT NULL` and `DEFAULT` constraints

### FR-4: SQL Aggregation Support

**FR-4.1: Supported Functions**
- `SUM(decimal_column)`
- `AVG(decimal_column)`
- `MIN(decimal_column)`
- `MAX(decimal_column)`
- `COUNT(decimal_column)`

**FR-4.2: Aggregation Requirements**
- Use native SQL numeric operations
- Preserve precision in aggregation results
- Return results convertible back to `Decimal`
- Support aggregation in existing `/aggregate` endpoint
- Maintain consistency across all database backends

**FR-4.3: Aggregation Endpoint Integration**
- Extend existing aggregation logic to handle `Decimal` fields
- Return aggregated values as strings in API responses
- Apply same validation rules to aggregated results

### FR-5: Go Integration

**FR-5.1: JSON Serialization**
- Implement `json.Marshaler`:
  ```go
  func (d Decimal) MarshalJSON() ([]byte, error)
  ```
- Implement `json.Unmarshaler`:
  ```go
  func (d *Decimal) UnmarshalJSON(data []byte) error
  ```
- Serialize as JSON string: `"123.45"`
- Deserialize from JSON string with validation

**FR-5.2: Database Integration**
- Implement `database/sql.Scanner`:
  ```go
  func (d *Decimal) Scan(value interface{}) error
  ```
- Implement `driver.Valuer`:
  ```go
  func (d Decimal) Value() (driver.Value, error)
  ```
- Support scanning from:
  - String values
  - Numeric values (int64, float64 for compatibility)
  - NULL values (return error or zero value)

**FR-5.3: Arithmetic Operations**
- All operations use `big.Rat` methods
- No implicit float conversion
- Support operations:
  - Addition: `Add(a, b Decimal) Decimal`
  - Subtraction: `Sub(a, b Decimal) Decimal`
  - Multiplication: `Mul(a, b Decimal) Decimal`
  - Division: `Div(a, b Decimal) (Decimal, error)`
- Division by zero returns explicit error

**FR-5.4: Comparison Operations**
- Equality: `Equal(a, b Decimal) bool`
- Less than: `Less(a, b Decimal) bool`
- Greater than: `Greater(a, b Decimal) bool`
- Compare: `Compare(a, b Decimal) int` (returns -1, 0, 1)

**FR-5.5: String Conversion**
- `String() string` method for canonical representation
- Fixed scale formatting (e.g., `"123.45"` for scale=2)
- No trailing zeros for whole numbers configurable
- Parse from string: `ParseDecimal(s string) (Decimal, error)`

### FR-6: API Contract

**FR-6.1: Input Validation**
- Validate decimal strings at API boundary
- Return HTTP 400 with clear error message on invalid input
- Example error: `"Invalid decimal value: '10.999' exceeds maximum scale of 2"`

**FR-6.2: Output Format**
- Always return decimals as strings in JSON responses
- Use canonical formatting (fixed scale)
- Example response:
  ```json
  {
    "id": "01HQZC8K6TXYZ9QRSTUVWXY012",
    "price": "199.99",
    "quantity": "5.00",
    "total": "999.95"
  }
  ```

**FR-6.3: Collection Schema Definition**
- Support `decimal` as a field type in collection schemas
- Schema definition:
  ```json
  {
    "name": "products",
    "fields": [
      {
        "name": "price",
        "type": "decimal",
        "required": true
      }
    ]
  }
  ```

### FR-7: Configuration

**FR-7.1: Constants Definition**
- Add to `internal/constants` package:
  ```go
  const (
      DefaultDecimalScale = 2
      MaxDecimalScale     = 10
  )
  ```

**FR-7.2: Configuration Scope**
- `DefaultDecimalScale`: used when scale not explicitly defined
- `MaxDecimalScale`: hard limit enforced during validation
- Both constants are compile-time values (not runtime config)

### FR-8: Error Handling

**FR-8.1: Error Types**
- **Validation Error**: Invalid decimal format or precision
- **Parse Error**: Cannot parse string to decimal
- **Division by Zero**: Explicit division by zero attempt
- **Conversion Error**: Cannot convert from database type

**FR-8.2: Error Messages**
- Clear and actionable error messages
- Include field name and invalid value
- Examples:
  - `"Invalid decimal value for field 'price': 'abc' is not a valid number"`
  - `"Decimal precision exceeded for field 'amount': maximum scale is 2, got 3"`
  - `"Division by zero in calculation"`

**FR-8.3: Error Handling Strategy**
- Fail fast on validation errors
- Return errors, do not panic
- Use standard error wrapping patterns
- Log validation failures at DEBUG level

### FR-9: Backward Compatibility

**FR-9.1: No Breaking Changes**
- No changes to existing API primitive types
- No changes to existing field types
- `Decimal` is a new optional field type

**FR-9.2: Migration Path**
- Existing `float` fields remain unchanged
- New schemas can use `decimal` type
- Documentation guides migration from `float` to `decimal`

### FR-10: Performance

**FR-10.1: Acceptable Use Cases**
- Business logic calculations
- Financial transactions
- Monetary value storage
- Measurement data
- Quantity calculations

**FR-10.2: Performance Characteristics**
- Slower than native float64 operations
- Acceptable for typical CRUD and aggregation workloads
- Not optimized for high-frequency scientific computation

**FR-10.3: Performance Constraints**
- Single record operations: < 1ms overhead
- Aggregation queries: comparable to native NUMERIC aggregation
- Bulk operations: acceptable for typical API usage patterns

---

## Acceptance Criteria

### AC-1: Type Implementation

**Verification:**
- [ ] `Decimal` type defined in appropriate package
- [ ] Backed by `math/big.Rat`
- [ ] Implements `json.Marshaler` and `json.Unmarshaler`
- [ ] Implements `sql.Scanner` and `driver.Valuer`
- [ ] No external dependencies added

**Test Cases:**
```go
// Test decimal creation
d, err := ParseDecimal("123.45")
assert.NoError(err)
assert.Equal("123.45", d.String())

// Test precision validation
_, err = ParseDecimal("123.456") // scale > 2
assert.Error(err)

// Test arithmetic
a := MustParseDecimal("10.50")
b := MustParseDecimal("5.25")
sum := a.Add(b)
assert.Equal("15.75", sum.String())
```

### AC-2: Database Storage

**Verification:**
- [ ] SQLite stores as `NUMERIC`
- [ ] MySQL/MariaDB stores as `DECIMAL(19,2)`
- [ ] PostgreSQL stores as `NUMERIC(19,2)`
- [ ] Values round-trip without precision loss

**Test Scenarios:**
1. **Insert and retrieve exact values:**
   ```
   POST /data/products
   {"price": "99.99"}
   
   GET /data/products/{id}
   Response: {"price": "99.99"}
   ```

2. **Verify database storage:**
   ```sql
   SELECT typeof(price), price FROM products WHERE id = ?
   -- SQLite: numeric, 99.99
   
   SELECT COLUMN_TYPE FROM information_schema.COLUMNS 
   WHERE TABLE_NAME='products' AND COLUMN_NAME='price'
   -- MySQL: decimal(19,2)
   ```

### AC-3: Aggregation Functions

**Verification:**
- [ ] `SUM` aggregation works correctly
- [ ] `AVG` aggregation preserves precision
- [ ] `MIN` and `MAX` work with decimal values
- [ ] Results returned as strings via API

**Test Cases:**
```bash
# Setup test data
POST /data/orders
[
  {"amount": "100.50", "quantity": "2.00"},
  {"amount": "75.25", "quantity": "1.50"},
  {"amount": "50.00", "quantity": "3.00"}
]

# Test aggregations
GET /aggregate/orders?field=amount&function=sum
Response: {"result": "225.75"}

GET /aggregate/orders?field=amount&function=avg
Response: {"result": "75.25"}

GET /aggregate/orders?field=amount&function=min
Response: {"result": "50.00"}

GET /aggregate/orders?field=amount&function=max
Response: {"result": "100.50"}
```

### AC-4: Validation

**Verification:**
- [ ] Invalid formats rejected at API boundary
- [ ] Excess precision rejected
- [ ] Empty strings rejected
- [ ] Scientific notation rejected
- [ ] Error messages are clear and actionable

**Test Cases:**
```bash
# Invalid decimal format
POST /data/products
{"price": "abc"}
Response: 400 {"error": "Invalid decimal value for field 'price': 'abc' is not a valid number"}

# Excess precision
POST /data/products
{"price": "99.999"}
Response: 400 {"error": "Decimal precision exceeded for field 'price': maximum scale is 2, got 3"}

# Scientific notation
POST /data/products
{"price": "1e10"}
Response: 400 {"error": "Invalid decimal value for field 'price': scientific notation not supported"}
```

### AC-5: JSON Serialization

**Verification:**
- [ ] Decimals serialize as JSON strings
- [ ] Decimals deserialize from JSON strings
- [ ] Invalid JSON strings rejected during unmarshal

**Test Cases:**
```go
// Marshal test
d := MustParseDecimal("123.45")
json, err := json.Marshal(d)
assert.NoError(err)
assert.Equal(`"123.45"`, string(json))

// Unmarshal test
var d Decimal
err := json.Unmarshal([]byte(`"123.45"`), &d)
assert.NoError(err)
assert.Equal("123.45", d.String())

// Invalid unmarshal
err = json.Unmarshal([]byte(`"invalid"`), &d)
assert.Error(err)
```

### AC-6: Arithmetic Operations

**Verification:**
- [ ] Addition works correctly
- [ ] Subtraction works correctly
- [ ] Multiplication works correctly
- [ ] Division works correctly
- [ ] Division by zero returns error

**Test Cases:**
```go
// Addition
a := MustParseDecimal("10.50")
b := MustParseDecimal("5.25")
assert.Equal("15.75", a.Add(b).String())

// Subtraction
assert.Equal("5.25", a.Sub(b).String())

// Multiplication
assert.Equal("55.13", a.Mul(b).String()) // Rounded to scale

// Division
result, err := a.Div(b)
assert.NoError(err)
assert.Equal("2.00", result.String())

// Division by zero
zero := MustParseDecimal("0")
_, err = a.Div(zero)
assert.Error(err)
assert.Contains(err.Error(), "division by zero")
```

### AC-7: Configuration Constants

**Verification:**
- [ ] `DefaultDecimalScale` constant exists and equals 2
- [ ] `MaxDecimalScale` constant exists and is configurable
- [ ] Constants used in validation logic

**Test Cases:**
```go
assert.Equal(2, constants.DefaultDecimalScale)
assert.Equal(10, constants.MaxDecimalScale)

// Test max scale enforcement
longDecimal := "1." + strings.Repeat("9", 11) // 11 digits
_, err := ParseDecimal(longDecimal)
assert.Error(err)
```

### AC-8: Cross-Database Consistency

**Verification:**
- [ ] Same values stored across all DB backends
- [ ] Aggregations produce identical results
- [ ] No precision loss on any backend

**Test Scenarios:**
Run identical test suite against:
1. SQLite
2. MySQL
3. MariaDB
4. PostgreSQL

**Verify:**
- All CRUD operations produce identical JSON responses
- All aggregation results match exactly
- Schema creation succeeds on all backends

### AC-9: Error Handling

**Verification:**
- [ ] All errors return clear messages
- [ ] No panics on invalid input
- [ ] Errors include field names and invalid values
- [ ] Validation errors logged at DEBUG level

**Test Cases:**
```bash
# Test various error conditions
POST /data/products {"price": null}
POST /data/products {"price": ""}
POST /data/products {"price": "10."}
POST /data/products {"price": ".50"}
POST /data/products {"price": "1,234.56"}

# All should return 400 with clear error message
# All should log at DEBUG level
```

### AC-10: Documentation and Code Quality

**Verification:**
- [ ] `Decimal` type fully documented with godoc comments
- [ ] Public functions have usage examples
- [ ] Unit tests achieve ≥90% coverage
- [ ] Integration tests verify cross-DB behavior
- [ ] Benchmark tests document performance characteristics

**Test Coverage:**
- Unit tests for all arithmetic operations
- Unit tests for JSON marshaling/unmarshaling
- Unit tests for database scanning/valuing
- Unit tests for validation logic
- Integration tests for CRUD operations
- Integration tests for aggregation queries
- Benchmark tests for critical operations

### AC-11: Integration with Existing System

**Verification:**
- [ ] Schema registry accepts `decimal` field type
- [ ] Query builder handles decimal filtering
- [ ] Aggregation endpoint supports decimal fields
- [ ] API validation framework validates decimals
- [ ] OpenAPI generation includes decimal type

**Test Cases:**
```bash
# Schema creation with decimal field
POST /schema
{
  "name": "invoices",
  "fields": [
    {"name": "amount", "type": "decimal", "required": true},
    {"name": "tax", "type": "decimal", "required": false}
  ]
}

# Query with decimal filtering
GET /data/invoices?amount_gt=100.00

# Sort by decimal field
GET /data/invoices?sort=amount

# Aggregate decimal field
GET /aggregate/invoices?field=amount&function=sum
```

### AC-12: No Regression

**Verification:**
- [ ] All existing tests pass
- [ ] No new compilation warnings
- [ ] No changes to existing API responses
- [ ] No performance degradation for non-decimal operations

**Test:**
- Run full test suite before and after implementation
- Compare API response samples for existing collections
- Run performance benchmarks for existing endpoints

---

## Implementation Checklist

- [x] Ensure all documentation, scripts, and samples (`SPEC.md`, `INSTALL.md`, `README.md`, `USAGE.md`, `install.sh`, and all files in `samples/*`) are updated and remain consistent with the implemented code changes.
- [x] Update API doc template at `cmd/moon/internal/handlers/templates/doc.md.tmpl` to reflect these changes.
- [x] Run all tests and ensure 100% pass rate.
- [x] If any test failure is unrelated to your feature, investigate and fix it before marking the task as complete.
- [x] Ensure all test scripts in `scripts/*.sh` are working properly and up to date with the latest code and API changes.

---

## Related PRDs

- [PRD-004: In-Memory Schema Registry](004-inmemory-schema-registry.md) - Schema field type definitions
- [PRD-008: Dynamic Query Builder](008-dynamic-query-builder.md) - Query filtering support
- [PRD-011: Request Validation](011-request-validation.md) - Validation framework
- [PRD-025: Aggregation Endpoints](025-aggregation-endpoints.md) - Aggregation function support
- [PRD-032: Config Version Constants](032-config-version-constants.md) - Constants package pattern
- [PRD-041: Data Type Refactor](041-data-type-refactor-sql-native-mapping.md) - Type system architecture
