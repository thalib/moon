# Moon - Dynamic Headless Engine

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE) [![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev/)

Moon is an API-first, migration-less backend in Go. Manage database schemas and data via REST APIs—no migration files needed.

> ## ⚠️ Breaking Change: Mandatory Authentication
> 
> **All API endpoints now require authentication** (except `/health`).
> 
> Quick setup:
> 1. Generate JWT secret: `openssl rand -base64 32`
> 2. Configure `jwt.secret` in `moon.conf`
> 3. Set up bootstrap admin in config
> 4. Login via `POST /auth:login`
> 5. Use `Authorization: Bearer <token>` header
> 
> See [INSTALL.md](INSTALL.md#authentication-setup) for complete setup instructions.

## Features

- Migration-less schema management (create/modify tables via API)
- In-memory schema registry for fast validation
- Multi-database: SQLite, PostgreSQL, MySQL
- Predictable API pattern (AIP-136 custom actions)
- Built-in HTML & Markdown documentation (`/doc/`, `/doc/llms-full.txt`)
- Server-side aggregations (`:count`, `:sum`, etc.)
- Docker-ready, efficient (<50MB RAM)
- **JWT & API key authentication** (mandatory)
- **Role-based access control** (admin, user with can_write)
- **Rate limiting** (100 req/min JWT, 1000 req/min API key)
- ULID identifiers
- Headless Backend for API-first apps like CMS, E-Commerce, CRM, Blog, Datastores etc.

## Quick Start

```bash
git clone https://github.com/thalib/moon.git
cd moon
```

### With Authentication (Required)

```bash
# 1. Build and install
sudo ./build.sh && sudo ./install.sh

# 2. Edit /etc/moon.conf:
#    - Set jwt.secret to: $(openssl rand -base64 32)
#    - Configure auth.bootstrap_admin section

# 3. Start Moon
sudo systemctl start moon

# 4. Login and get token
TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "change-me-on-first-login"}' \
  | jq -r '.access_token')

# 5. Use authenticated requests
curl http://localhost:6006/collections:list \
  -H "Authorization: Bearer $TOKEN"
```

See [INSTALL.md](INSTALL.md) for complete setup including Docker deployment.

## Documentation

Moon provides comprehensive, auto-generated API documentation:

- **HTML Documentation**: Visit `http://localhost:6006/doc/` in your browser for a complete, interactive API reference
- **Markdown Documentation**: Access `http://localhost:6006/doc/llms-full.txt` for terminal-friendly or AI-agent documentation
- Configuration: See `samples/moon.conf` for comprehensive, spec-compliant configuration
- Testing: See `scripts/test-runner.sh`

### Authentication Test Suite

Run the authentication test scripts to verify your setup:

```bash
./scripts/auth-all.sh      # Run all auth tests
./scripts/auth-jwt.sh      # JWT authentication tests
./scripts/auth-apikey.sh   # API key tests
./scripts/auth-rbac.sh     # Role-based access control tests
./scripts/auth-ratelimit.sh # Rate limiting tests
```

### Additional Resources

- [INSTALL.md](INSTALL.md): Installation and deployment guide (includes authentication setup)
- [SPEC.md](SPEC.md): Architecture and technical specifications
- [SPEC_AUTH.md](SPEC_AUTH.md): Detailed authentication specification
- [samples/](samples/): Sample configuration files
- [scripts/](scripts/): Test and demo scripts
- [LICENSE](LICENSE): MIT License
- [GitHub Issues](https://github.com/thalib/moon/issues)
- [GitHub Discussions](https://github.com/thalib/moon/discussions)

## License & Credits

MIT License ([LICENSE](LICENSE))
Built with [Go](https://go.dev/), [Viper](https://github.com/spf13/viper), [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), [lib/pq](https://github.com/lib/pq), [modernc.org/sqlite](https://gitlab.com/cznic/sqlite)

---

Made by [Devnodes.in](https://github.com/devnodesin)
