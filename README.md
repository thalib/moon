# Moon - Dynamic Headless Engine

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev/)

Moon is a high-performance, API-first headless backend built in Go that enables **migration-less database management** through REST APIs. With Moon, you can create, modify, and manage database schemas dynamically via API calls, eliminating the need for traditional migration files.

## ✨ Key Features

- **🚀 Migration-Less Schema Management** - Create and modify database tables via API calls in real-time
- **⚡ Zero-Latency Validation** - In-memory schema registry for nanosecond-level validation
- **🔌 Multi-Database Support** - SQLite, PostgreSQL, and MySQL with automatic dialect detection
- **🎯 Predictable API Pattern** - AIP-136 custom actions (`:create`, `:update`, `:list`, etc.)
- **📊 Dynamic OpenAPI Documentation** - Auto-generated from the current database schema
- **📈 Server-Side Aggregations** - Built-in `:count`, `:sum`, `:avg`, `:min`, `:max` endpoints with filtering
- **🐳 Docker Ready** - Multi-stage builds for minimal production containers
- **💾 Resource Efficient** - Memory footprint under 50MB, optimized for cloud and edge deployments
- **🔒 Built-in Security** - JWT authentication and optional API key support
- **🆔 ULID Identifiers** - Lexicographically sortable, 26-character unique identifiers for all records

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/thalib/moon.git
cd moon

# Set up configuration
cp samples/moon.conf /etc/moon.conf
# Edit /etc/moon.conf and set JWT secret

# Build
go build -o moon ./cmd/moon

# Run in console mode (foreground)
./moon --config /etc/moon.conf

# Or run in daemon mode (background)
./moon --daemon --config /etc/moon.conf
# or shorthand: ./moon -d --config /etc/moon.conf

# Test the API
curl http://localhost:6006/health
```

**See the full [Installation Guide](INSTALL.md) for detailed setup instructions.**

## 📖 Documentation

- **[Installation Guide](INSTALL.md)** - Build, install, and deployment instructions
- **[Usage Guide](USAGE.md)** - Complete API reference and examples
- **[Architecture Spec](SPEC.md)** - System design and technical specifications
- **[Sample Configurations](samples/)** - Ready-to-use configuration files and scripts
- **[License](LICENSE)** - MIT License

## 🎯 Use Cases

Moon is perfect for:

- **Headless CMS** - Build content management systems with dynamic schemas
- **API-First Applications** - Rapid prototyping without database migrations
- **Edge Computing** - Lightweight backend for resource-constrained environments
- **Dynamic Data Platforms** - Applications requiring runtime schema modifications
- **Multi-Tenant Systems** - Flexible data models for diverse clients

## 🔥 Example Usage

Create a collection and manage data with simple API calls:

```bash
# Create a products collection
curl -X POST http://localhost:6006/api/v1/collections:create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "products",
    "columns": [
      {"name": "name", "type": "string", "required": true},
      {"name": "price", "type": "float", "required": true},
      {"name": "stock", "type": "integer", "required": true}
    ]
  }'

# Insert a product
curl -X POST http://localhost:6006/api/v1/products:create \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "name": "Laptop",
      "price": 1299.99,
      "stock": 50
    }
  }'

# List all products with pagination
curl "http://localhost:6006/api/v1/products:list?limit=20"

# List products with cursor-based pagination
curl "http://localhost:6006/api/v1/products:list?limit=20&after=01ARZ3NDEKTSV4RRFFQ69G5FAV"

# Get a specific product by ULID
curl "http://localhost:6006/api/v1/products:get?id=01ARZ3NDEKTSV4RRFFQ69G5FAV"

# Update a product (using ULID)
curl -X POST http://localhost:6006/api/v1/products:update \
  -H "Content-Type: application/json" \
  -d '{
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "data": {"price": 1199.99}
  }'
```

**Note:** All records are identified by ULIDs (Universally Unique Lexicographically Sortable Identifiers), which are 26-character, URL-safe strings. ULIDs are automatically generated when creating records and are used for all operations (get, update, delete).

**For more examples, see the [Usage Guide](USAGE.md) or run the [API demo script](samples/api-demo.sh).**

## 🏗️ Architecture

Moon implements a **smart bridge** between your application and the database:

```
Client Request → Router → In-Memory Schema Cache → Validator → SQL Builder → Database
                             ↑                                       ↓
                             └───────── Schema Updates ─────────────┘
```

### Key Components

- **In-Memory Schema Registry** - Stores database structure using `sync.Map` for concurrent access
- **Dynamic SQL Builder** - Generates safe, parameterized queries based on schema
- **Custom Action Pattern** - RESTful endpoints using `:action` suffix (e.g., `/products:create`)
- **Multi-Database Driver** - Abstracts SQLite, PostgreSQL, and MySQL differences

See [SPEC.md](SPEC.md) for detailed architecture documentation.

## 🛠️ API Pattern

Moon uses the AIP-136 custom action pattern for predictable, AI-friendly APIs:

### Collections Management (`/api/v1/collections`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/collections:list` | GET | List all collections |
| `/collections:get` | GET | Get collection schema |
| `/collections:create` | POST | Create new collection |
| `/collections:update` | POST | Modify collection schema |
| `/collections:destroy` | POST | Delete collection |

### Data Operations (`/api/v1/{collectionName}`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/{name}:list` | GET | List all records |
| `/{name}:get` | GET | Get single record |
| `/{name}:create` | POST | Create new record |
| `/{name}:update` | POST | Update existing record |
| `/{name}:destroy` | POST | Delete record |

**For complete API documentation, see the [Usage Guide](docs/USAGE.md).**

## ⚙️ Configuration

Moon uses YAML-only configuration (no environment variables):

### Minimum Required Configuration (`/etc/moon.conf`)

```yaml
server:
  host: "0.0.0.0"
  port: 6006

database:
  connection: "sqlite"
  database: "/opt/moon/sqlite.db"

logging:
  path: "/var/log/moon"

jwt:
  secret: "your-super-secret-key"  # REQUIRED - generate with: openssl rand -base64 32
  expiry: 3600

apikey:
  enabled: false
  header: "X-API-KEY"
```

### Database Options

```yaml
# SQLite (default)
database:
  connection: "sqlite"
  database: "/opt/moon/sqlite.db"

# PostgreSQL
database:
  connection: "postgres"
  database: "moon_db"
  user: "moon_user"
  password: "secure_password"
  host: "localhost"

# MySQL
database:
  connection: "mysql"
  database: "moon_db"
  user: "moon_user"
  password: "secure_password"
  host: "localhost"
```

### Running Modes

**Console Mode (Default)** - Foreground with dual logging (stdout + file):

```bash
./moon --config /etc/moon.conf
# Logs to terminal (console format) AND /var/log/moon/main.log (simple format)
```

**Daemon Mode** - Background with file-only logging:

```bash
./moon --daemon --config /etc/moon.conf
# or shorthand
./moon -d --config /etc/moon.conf
# Logs only to /var/log/moon/main.log
```

**Systemd Service** - Production deployment:

```bash
sudo systemctl start moon
sudo systemctl enable moon
```

**See [Installation Guide](INSTALL.md) for complete configuration options and [samples/](samples/) for example configurations.**

## 🧪 Testing

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.txt

# Use the test runner script
./samples/test-runner.sh

# Run the API demo
./moon &
./samples/api-demo.sh
```

## 🐳 Docker Support

```bash
# Build with Docker
docker run --rm -v "$(pwd):/app" -w /app golang:1.24 \
  sh -c "go build -buildvcs=false -o moon ./cmd/moon"

# Run with Docker (mount config file)
docker run -p 6006:6006 -v /etc/moon.conf:/etc/moon.conf moon:latest
```

**See [Installation Guide](INSTALL.md#docker-build) for Docker deployment options.**

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write tests for new features
- Update documentation for API changes
- Run `go fmt` and `go vet` before committing

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🌟 Acknowledgments

- Built with [Go](https://go.dev/)
- Uses [Viper](https://github.com/spf13/viper) for configuration
- Database drivers: [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), [lib/pq](https://github.com/lib/pq), [modernc.org/sqlite](https://gitlab.com/cznic/sqlite)

## 📞 Support

- **Documentation:** [INSTALL.md](INSTALL.md), [USAGE.md](USAGE.md), [SPEC.md](SPEC.md)
- **Issues:** [GitHub Issues](https://github.com/thalib/moon/issues)
- **Discussions:** [GitHub Discussions](https://github.com/thalib/moon/discussions)

---

**Made with ❤️ by [Mohamed Thalib](https://github.com/thalib)**
