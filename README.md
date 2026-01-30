# Moon - Dynamic Headless Engine

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE) [![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev/)

Moon is an API-first, migration-less backend in Go. Manage database schemas and data via REST APIs—no migration files needed.

## Features

- Migration-less schema management (create/modify tables via API)
- In-memory schema registry for fast validation
- Multi-database: SQLite, PostgreSQL, MySQL
- Predictable API pattern (AIP-136 custom actions)
- Built-in HTML & Markdown documentation (`/doc/`, `/doc/md`)
- Server-side aggregations (`:count`, `:sum`, etc.)
- Docker-ready, efficient (<50MB RAM)
- JWT & API key auth
- ULID identifiers

## Quick Start

```bash
git clone https://github.com/thalib/moon.git
cd moon
```

See [INSTALL.md](INSTALL.md) for setup.

## Documentation & Support

- [INSTALL.md](INSTALL.md): Install & deploy
- [USAGE.md](USAGE.md): API reference & examples
- [SPEC.md](SPEC.md): Architecture & technical details
- [samples/](samples/): Sample Configuration
- [scripts/](scripts/): Test Scripts
- [LICENSE](LICENSE): MIT License
- [GitHub Issues](https://github.com/thalib/moon/issues)
- [GitHub Discussions](https://github.com/thalib/moon/discussions)

## Use Cases

- Headless CMS
- API-first apps
- Edge computing
- Dynamic data platforms
- Multi-tenant systems

## Architecture

Request flow:
Client → Router → Schema Cache → Validator → SQL Builder → Database

Key components:

- In-memory schema registry (`sync.Map`)
- Dynamic SQL builder
- Custom action REST endpoints (e.g., `/products:create`)
- Multi-database driver abstraction

## API Pattern

Collections management (`/api/v1/collections`):

| Endpoint              | Method | Purpose                |
|-----------------------|--------|------------------------|
| `/collections:list`   | GET    | List all collections   |
| `/collections:get`    | GET    | Get collection schema  |
| `/collections:create` | POST   | Create new collection  |
| `/collections:update` | POST   | Modify collection      |
| `/collections:destroy`| POST   | Delete collection      |

Data operations (`/api/v1/{collectionName}`):

| Endpoint         | Method | Purpose         |
|------------------|--------|-----------------|
| `/{name}:list`   | GET    | List records    |
| `/{name}:get`    | GET    | Get record      |
| `/{name}:create` | POST   | Create record   |
| `/{name}:update` | POST   | Update record   |
| `/{name}:destroy`| POST   | Delete record   |

See [USAGE.md](USAGE.md) for full API docs and examples.

## Configuration

YAML-only config (no env vars). See `samples/moon.conf` and `samples/moon-full.conf`.

## Contributing

1. Fork & branch
2. Commit & push
3. Open a PR

Guidelines:

- Follow Go best practices
- Write tests for features
- Update docs for API changes
- Run `go fmt` and `go vet`

## License & Credits

MIT License ([LICENSE](LICENSE))
Built with [Go](https://go.dev/), [Viper](https://github.com/spf13/viper), [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), [lib/pq](https://github.com/lib/pq), [modernc.org/sqlite](https://gitlab.com/cznic/sqlite)

---
Made by [Mohamed Thalib](https://github.com/thalib)
