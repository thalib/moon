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
- **💾 Resource Efficient** - Memory footprint under 50MB, optimized for cloud and edge deployments
- **🔒 Built-in Security** - JWT authentication and optional API key support

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/thalib/moon.git
cd moon

# Set up configuration
cp samples/.env.example .env
# Edit .env and set MOON_JWT_SECRET

# Build and run
go build -o moon ./cmd/moon
./moon

# Test the API
curl http://localhost:8080/health
```

**See the full [Installation Guide](docs/INSTALL.md) for detailed setup instructions.**

## 📖 Documentation

- **[Installation Guide](docs/INSTALL.md)** - Build, install, and deployment instructions
- **[Usage Guide](docs/USAGE.md)** - Complete API reference and examples
- **[Architecture Spec](SPEC.md)** - System design and technical specifications
- **[Sample Scripts](samples/README.md)** - Ready-to-use configuration and demo scripts
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
curl -X POST http://localhost:8080/api/v1/collections:create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "products",
    "columns": [
      {"name": "name", "type": "text", "required": true},
      {"name": "price", "type": "real", "required": true},
      {"name": "stock", "type": "integer", "required": true}
    ]
  }'

# Insert a product
curl -X POST http://localhost:8080/api/v1/products:create \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "name": "Laptop",
      "price": 1299.99,
      "stock": 50
    }
  }'

# List all products
curl http://localhost:8080/api/v1/products:list

# Update a product
curl -X POST http://localhost:8080/api/v1/products:update \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "data": {"price": 1199.99}
  }'
```

**For more examples, see the [Usage Guide](docs/USAGE.md) or run the [API demo script](samples/api-demo.sh).**

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

Moon supports flexible configuration via environment variables or YAML files:

### Minimum Required Configuration

```bash
# Required: Set JWT secret for authentication
export MOON_JWT_SECRET=your-super-secret-key
```

### Database Options

```bash
# SQLite (default)
MOON_DATABASE_CONNECTION_STRING=sqlite://moon.db

# PostgreSQL
MOON_DATABASE_CONNECTION_STRING=postgres://user:pass@localhost:5432/dbname

# MySQL
MOON_DATABASE_CONNECTION_STRING=mysql://user:pass@localhost:3306/dbname
```

**See [Installation Guide](docs/INSTALL.md) for complete configuration options and [samples/](samples/) for example configurations.**

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

# Run with Docker
docker run -p 8080:8080 -e MOON_JWT_SECRET=secret moon:latest
```

**See [Installation Guide](docs/INSTALL.md#docker-build) for Docker deployment options.**

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

- **Documentation:** [docs/](docs/)
- **Issues:** [GitHub Issues](https://github.com/thalib/moon/issues)
- **Discussions:** [GitHub Discussions](https://github.com/thalib/moon/discussions)

---

**Made with ❤️ by [Mohamed Thalib](https://github.com/thalib)**
