# Moon Usage Guide

This guide provides comprehensive information on using Moon's API and features.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [API Endpoints](#api-endpoints)
- [Example Commands](#example-commands)
- [Using Sample Data](#using-sample-data)
- [Configuration Details](#configuration-details)
- [Authentication](#authentication)
- [Troubleshooting](#troubleshooting)

## Overview

Moon is a dynamic headless engine that provides a migration-less database management system through REST APIs. It uses a custom action pattern (`:action`) to provide predictable, AI-friendly interfaces for managing collections (tables) and their data.

### Key Features

- **Migration-Less Schema Management:** Create, modify, and delete database tables via API calls
- **Multi-Database Support:** SQLite, PostgreSQL, and MySQL
- **In-Memory Schema Caching:** Fast validation with zero-latency lookups
- **RESTful API:** Follows AIP-136 custom action pattern
- **Dynamic OpenAPI Documentation:** Auto-generated from schema cache
- **Lightweight:** Memory footprint under 50MB

### API Pattern

Moon uses a colon (`:`) separator to distinguish between resources and actions:

- `/collections:list` - List all collections
- `/collections:create` - Create a new collection
- `/products:list` - List all products
- `/products:create` - Create a new product

This pattern makes the API predictable and easy to use.

## Quick Start

### 1. Start the Server

```bash
# Console mode (foreground)
./moon --config /etc/moon.conf

# Or daemon mode (background)
./moon --daemon --config /etc/moon.conf
```

### 2. Check Health

```bash
curl http://localhost:6006/health
```

Expected response:

```json
{"status":"healthy"}
```

### 3. Create Your First Collection

```bash
curl -X POST http://localhost:6006/api/v1/collections:create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "users",
    "columns": [
      {"name": "email", "type": "string", "required": true},
      {"name": "name", "type": "string", "required": true},
      {"name": "age", "type": "integer", "required": false}
    ]
  }'
```

### 4. Insert Data

```bash
curl -X POST http://localhost:6006/api/v1/users:create \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "email": "john@example.com",
      "name": "John Doe",
      "age": 30
    }
  }'
```

### 5. Query Data

```bash
curl http://localhost:6006/api/v1/users:list
```

## API Endpoints

### Base URL

All API endpoints are prefixed with `/api/v1`.

### Collections Management

Manage database schemas (tables and columns).

#### List Collections

```
GET /api/v1/collections:list
```

Lists all managed collections.

**Response:**

```json
{
  "collections": ["users", "products", "orders"],
  "count": 3
}
```

#### Get Collection Schema

```
GET /api/v1/collections:get?name={collectionName}
```

Retrieves the schema for a specific collection.

**Parameters:**

- `name` (query, required): Collection name

**Response:**

```json
{
  "collection": {
    "name": "users",
    "columns": [
      {"name": "id", "type": "integer", "required": true},
      {"name": "email", "type": "text", "required": true},
      {"name": "name", "type": "text", "required": true},
      {"name": "age", "type": "integer", "required": false}
    ]
  }
}
```

#### Create Collection

```
POST /api/v1/collections:create
```

Creates a new collection (table) in the database.

**Request Body:**

```json
{
  "name": "products",
  "columns": [
    {"name": "name", "type": "string", "required": true},
    {"name": "price", "type": "float", "required": true},
    {"name": "description", "type": "text", "required": false}
  ]
}
```

**Column Types:**

- `string`: Short string values
- `text`: Long text values
- `integer`: Whole numbers
- `float`: Floating-point numbers
- `boolean`: True/false values
- `datetime`: Date and time values
- `json`: JSON data

**Response:**

```json
{
  "collection": { /* collection schema */ },
  "message": "Collection created successfully"
}
```

#### Update Collection

```
POST /api/v1/collections:update
```

Modifies a collection schema (adds columns).

**Request Body:**

```json
{
  "name": "products",
  "add_columns": [
    {"name": "category", "type": "text", "required": false}
  ]
}
```

**Response:**

```json
{
  "collection": { /* updated schema */ },
  "message": "Collection updated successfully"
}
```

#### Destroy Collection

```
POST /api/v1/collections:destroy
```

Drops a collection and all its data.

**Request Body:**

```json
{
  "name": "products"
}
```

**Response:**

```json
{
  "message": "Collection destroyed successfully"
}
```

### Data Operations

Perform CRUD operations on collection data.

#### List Records

```
GET /api/v1/{collectionName}:list
```

Retrieves all records from a collection.

**Query Parameters:**

- `limit` (optional): Number of records to return (default: no limit)
- `offset` (optional): Number of records to skip (default: 0)

**Example:**

```bash
# List all products
curl http://localhost:8080/api/v1/products:list

# List with pagination
curl http://localhost:8080/api/v1/products:list?limit=10&offset=0
```

**Response:**

```json
{
  "data": [
    {"id": 1, "name": "Laptop", "price": 1299.99},
    {"id": 2, "name": "Mouse", "price": 29.99}
  ],
  "count": 2,
  "limit": 10,
  "offset": 0
}
```

#### Get Record

```
GET /api/v1/{collectionName}:get?id={id}
```

Retrieves a single record by ID.

**Parameters:**

- `id` (query, required): Record ID

**Example:**

```bash
curl http://localhost:8080/api/v1/products:get?id=1
```

**Response:**

```json
{
  "data": {
    "id": 1,
    "name": "Laptop",
    "price": 1299.99,
    "description": "High-performance laptop"
  }
}
```

#### Create Record

```
POST /api/v1/{collectionName}:create
```

Inserts a new record into the collection.

**Request Body:**

```json
{
  "data": {
    "name": "Keyboard",
    "price": 79.99,
    "description": "Mechanical keyboard"
  }
}
```

**Response:**

```json
{
  "data": {
    "id": 3,
    "name": "Keyboard",
    "price": 79.99,
    "description": "Mechanical keyboard"
  },
  "message": "Record created successfully"
}
```

#### Update Record

```
POST /api/v1/{collectionName}:update
```

Updates an existing record.

**Request Body:**

```json
{
  "id": 1,
  "data": {
    "price": 1199.99,
    "description": "Discounted laptop"
  }
}
```

Only the fields provided in `data` will be updated; other fields remain unchanged.

**Response:**

```json
{
  "data": {
    "id": 1,
    "name": "Laptop",
    "price": 1199.99,
    "description": "Discounted laptop"
  },
  "message": "Record updated successfully"
}
```

#### Delete Record

```
POST /api/v1/{collectionName}:destroy
```

Deletes a record from the collection.

**Request Body:**

```json
{
  "id": 3
}
```

**Response:**

```json
{
  "message": "Record deleted successfully"
}
```

## Example Commands

### Complete Workflow Example

Here's a complete workflow demonstrating Moon's capabilities:

```bash
# 1. Create a blog collection
curl -X POST http://localhost:8080/api/v1/collections:create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "posts",
    "columns": [
      {"name": "title", "type": "string", "required": true},
      {"name": "content", "type": "text", "required": true},
      {"name": "author", "type": "string", "required": true},
      {"name": "published", "type": "boolean", "required": false}
    ]
  }'

# 2. Create some posts
curl -X POST http://localhost:8080/api/v1/posts:create \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "title": "Getting Started with Moon",
      "content": "Moon is a powerful headless CMS...",
      "author": "Jane Doe",
      "published": true
    }
  }'

curl -X POST http://localhost:8080/api/v1/posts:create \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "title": "Advanced Moon Features",
      "content": "Learn about dynamic schemas...",
      "author": "John Smith",
      "published": false
    }
  }'

# 3. List all posts
curl http://localhost:8080/api/v1/posts:list

# 4. Get a specific post
curl http://localhost:8080/api/v1/posts:get?id=1

# 5. Update a post
curl -X POST http://localhost:8080/api/v1/posts:update \
  -H "Content-Type: application/json" \
  -d '{
    "id": 2,
    "data": {
      "published": true
    }
  }'

# 6. Add a new column to the schema
curl -X POST http://localhost:8080/api/v1/collections:update \
  -H "Content-Type: application/json" \
  -d '{
    "name": "posts",
    "add_columns": [
      {"name": "views", "type": "integer", "required": false}
    ]
  }'

# 7. Delete a post
curl -X POST http://localhost:8080/api/v1/posts:destroy \
  -H "Content-Type: application/json" \
  -d '{"id": 1}'

# 8. Clean up - destroy the collection
curl -X POST http://localhost:8080/api/v1/collections:destroy \
  -H "Content-Type: application/json" \
  -d '{"name": "posts"}'
```

### Using jq for Pretty Output

Install `jq` for formatted JSON output:

```bash
# List collections with formatted output
curl -s http://localhost:8080/api/v1/collections:list | jq '.'

# Get and extract specific fields
curl -s http://localhost:8080/api/v1/products:list | jq '.data[] | {id, name, price}'
```

## Using Sample Data

The `samples/` directory contains helpful scripts and examples.

### Running the API Demo

The API demo script demonstrates all Moon operations:

```bash
# Start Moon server
./moon &

# Run comprehensive demo
./samples/api-demo.sh
```

The script will:

1. Create a products collection
2. Insert sample data
3. Demonstrate pagination
4. Update records
5. Modify schema
6. Clean up

See [`samples/README.md`](../samples/README.md) for more details.

### Using Sample Configuration

Copy and customize the sample configuration:

```bash
# Use sample environment variables
cp samples/.env.example .env
nano .env  # Edit as needed

# Or use sample YAML config
cp samples/config.example.yaml config.yaml
nano config.yaml  # Edit as needed
```

## Configuration Details

### Configuration Sources

Moon loads configuration from multiple sources in this priority order:

1. **Environment Variables** (highest priority)
2. **Configuration File** (`config.yaml` or `config.toml`)
3. **Default Values** (lowest priority)

### Environment Variables

All configuration can be set via environment variables using the `MOON_` prefix:

```bash
# Server configuration
export MOON_SERVER_HOST=0.0.0.0
export MOON_SERVER_PORT=8080

# Database configuration
export MOON_DATABASE_CONNECTION_STRING=sqlite://moon.db
export MOON_DATABASE_MAX_OPEN_CONNS=25
export MOON_DATABASE_MAX_IDLE_CONNS=5
export MOON_DATABASE_CONN_MAX_LIFETIME=300

# Authentication configuration
export MOON_JWT_SECRET=your-production-secret
export MOON_JWT_EXPIRATION=3600

# API Key configuration (optional)
export MOON_APIKEY_ENABLED=false
export MOON_APIKEY_HEADER=X-API-Key
```

### Configuration File

Create a `config.yaml` file in the project root:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  connection_string: "sqlite://moon.db"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300

jwt:
  secret: "your-secret-key"
  expiration: 3600

apikey:
  enabled: false
  header: "X-API-Key"
```

### Database Connection Strings

#### SQLite (Default)

```bash
MOON_DATABASE_CONNECTION_STRING=sqlite://moon.db
# Or just the file path
MOON_DATABASE_CONNECTION_STRING=moon.db
```

#### PostgreSQL

```bash
MOON_DATABASE_CONNECTION_STRING=postgres://username:password@localhost:5432/dbname
```

#### MySQL

```bash
MOON_DATABASE_CONNECTION_STRING=mysql://username:password@localhost:3306/dbname
```

### Server Configuration

#### Host and Port

```bash
# Listen on all interfaces (default)
MOON_SERVER_HOST=0.0.0.0
MOON_SERVER_PORT=8080

# Listen only on localhost
MOON_SERVER_HOST=127.0.0.1
MOON_SERVER_PORT=3000
```

### Performance Tuning

#### Database Connection Pool

Adjust based on your workload:

```bash
# Maximum open connections
MOON_DATABASE_MAX_OPEN_CONNS=25

# Maximum idle connections
MOON_DATABASE_MAX_IDLE_CONNS=5

# Connection lifetime (seconds)
MOON_DATABASE_CONN_MAX_LIFETIME=300
```

## Authentication

### JWT Authentication

Moon uses JWT (JSON Web Tokens) for authentication.

#### Setting JWT Secret

The JWT secret is **required** for the server to start:

```bash
# Set via environment variable (recommended)
export MOON_JWT_SECRET=your-super-secret-key-at-least-32-chars

# Or in config.yaml (less secure)
jwt:
  secret: "your-super-secret-key"
```

**Security Note:** Use a strong, random secret in production:

```bash
# Generate a secure random secret
openssl rand -base64 32
```

#### Using JWT Tokens

Future versions will include token generation endpoints. For now, Moon validates the JWT secret configuration at startup.

### API Key Authentication (Optional)

Enable API key authentication for additional security:

```bash
MOON_APIKEY_ENABLED=true
MOON_APIKEY_HEADER=X-API-Key
```

Then pass the API key in requests:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/collections:list
```

## Troubleshooting

### Common Issues

#### Server Won't Start

**Problem:** `Failed to load configuration: JWT secret is required`

**Solution:**

```bash
export MOON_JWT_SECRET=your-secret-key
./moon
```

#### Database Connection Errors

**Problem:** `Failed to connect to database`

**Solutions:**

1. Check connection string format
2. Ensure database server is running
3. Verify credentials and permissions
4. For SQLite, ensure write permissions on directory

#### API Returns 404

**Problem:** Endpoint not found

**Solutions:**

1. Check URL pattern: `/api/v1/{collection}:{action}`
2. Verify collection exists: `GET /api/v1/collections:list`
3. Ensure proper HTTP method (GET vs POST)

#### Schema Validation Errors

**Problem:** `Invalid column type` or `Required field missing`

**Solutions:**

1. Check column types: `string`, `text`, `integer`, `float`, `boolean`, `datetime`, `json`
2. Ensure required fields are provided
3. Get collection schema: `GET /api/v1/collections:get?name={collection}`

### Debug Mode

Run with verbose logging:

```bash
# Set log level (if implemented)
MOON_LOG_LEVEL=debug ./moon
```

### Health Check

Always start troubleshooting with a health check:

```bash
curl http://localhost:8080/health

# Expected response
{"status":"healthy"}
```

### Checking Logs

Moon logs to stdout by default. Redirect to a file for analysis:

```bash
./moon > moon.log 2>&1 &
tail -f moon.log
```

### Testing Connection

Use the API demo script to verify everything works:

```bash
./samples/api-demo.sh
```

If the demo completes successfully, your Moon installation is working correctly.

## Next Steps

- Review the [Installation Guide](INSTALL.md) for deployment options
- Check [SPEC.md](../SPEC.md) for architecture details
- Explore [sample scripts](../samples/README.md) for more examples
- Read the [README](../README.md) for project overview

## Additional Resources

- **Sample Scripts:** See `samples/` directory for ready-to-use examples
- **Configuration Examples:** `samples/config.example.yaml` and `samples/.env.example`
- **API Demo:** Run `samples/api-demo.sh` for a complete walkthrough
- **GitHub Repository:** [https://github.com/thalib/moon](https://github.com/thalib/moon)

## Getting Help

If you encounter issues:

1. Check this troubleshooting section
2. Review the [INSTALL.md](INSTALL.md) guide
3. Verify your configuration in `samples/` directory
4. Run the API demo script to isolate issues
5. Open an issue on GitHub with detailed error messages
