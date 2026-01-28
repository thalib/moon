# Moon Installation Guide

This guide covers building, installing, and running Moon in various environments.

## Prerequisites

- **Go 1.24** or later
- **Git** for cloning the repository
- **Docker** (optional, for containerized builds)
- **curl** (for testing API endpoints)

## Quick Start

For the impatient, here's the fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/thalib/moon.git
cd moon

# Build Moon
go build -o moon ./cmd/moon

# Run the API demo (no configuration needed!)
./samples/api-demo.sh
```

The demo script will automatically start a server and demonstrate all API features.

For a manual setup with custom configuration:

```bash
# Set up configuration
cp samples/moon.conf /etc/moon.conf
# Edit /etc/moon.conf and set jwt.secret to a secure value
# Generate with: openssl rand -base64 32

# Build and run
go build -o moon ./cmd/moon
./moon --config /etc/moon.conf
```

## Standard Build (Go)

### 1. Clone the Repository

```bash
git clone https://github.com/thalib/moon.git
cd moon
```

### 2. Install Dependencies

```bash
go mod download
go mod tidy
```

### 3. Build the Binary

```bash
go build -o moon ./cmd/moon
```

This creates a `moon` binary in the current directory.

### 4. Build with Specific Options

#### Optimized Production Build

```bash
go build -ldflags="-s -w" -o moon ./cmd/moon
```

The `-s -w` flags strip debug information and reduce binary size.

#### Build for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o moon-linux ./cmd/moon

# macOS
GOOS=darwin GOARCH=amd64 go build -o moon-darwin ./cmd/moon

# Windows
GOOS=windows GOARCH=amd64 go build -o moon.exe ./cmd/moon
```

## Docker Build

### Build Binary Using Docker

If you don't have Go installed locally, you can build using Docker:

```bash
# Download dependencies
docker run --rm -v "$(pwd):/app" -w /app golang:1.24 \
  sh -c "go mod download && go mod tidy"

# Build the binary
docker run --rm -v "$(pwd):/app" -w /app golang:1.24 \
  sh -c "go build -buildvcs=false -o moon ./cmd/moon"
```

The `-buildvcs=false` flag disables VCS stamping, which is useful when building in Docker without Git metadata.

### Create a Docker Container (Optional)

You can also create a Dockerfile for containerized deployment:

```dockerfile
# Dockerfile
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o moon ./cmd/moon

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/moon .
COPY samples/moon.conf /etc/moon.conf

EXPOSE 6006
CMD ["./moon", "--config", "/etc/moon.conf"]
```

Build and run:

```bash
docker build -t moon:latest .
docker run -p 6006:6006 -v /etc/moon.conf:/etc/moon.conf moon:latest
```

## Configuration

Moon uses **YAML-only configuration** (no environment variables). All configuration must be set in a YAML file.

### Configuration File Location

- **Default:** `/etc/moon.conf`
- **Custom:** Specify with `--config` flag: `./moon --config /path/to/config.yaml`

### Minimal Configuration

Create `/etc/moon.conf` with minimal settings:

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

### Configuration Examples

#### Development Configuration

For local development, use `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 6006

database:
  connection: "sqlite"
  database: "moon.db"  # Local file

logging:
  path: "./logs"

jwt:
  secret: "dev-secret-key-change-in-production"
  expiry: 3600

apikey:
  enabled: false
  header: "X-API-KEY"
```

#### Production Configuration

For production, use `/etc/moon.conf`:

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
  secret: "GENERATE_WITH_openssl_rand_base64_32"
  expiry: 3600

apikey:
  enabled: true
  header: "X-API-KEY"
```

### Database Configuration

Moon supports three database backends. Configure in the `database` section:

#### SQLite (Default)

```yaml
database:
  connection: "sqlite"
  database: "/opt/moon/sqlite.db"
```

SQLite is used by default if no database is configured. Perfect for:

- Development and testing
- Single-server deployments  
- Embedded applications

#### PostgreSQL

```yaml
database:
  connection: "postgres"
  database: "moon_db"
  user: "moon_user"
  password: "secure_password"
  host: "localhost"
```

#### MySQL

```yaml
database:
  connection: "mysql"
  database: "moon_db"
  user: "moon_user"
  password: "secure_password"
  host: "localhost"
```

### Sample Configuration Files

See `samples/` directory for example configurations:

- `samples/moon.conf` - Minimal quick-start configuration
- `samples/moon-full.conf` - Comprehensive configuration with all options documented
- `samples/config.example.yaml` - Development configuration template

## Running Moon

### Console Mode (Foreground)

Run in the foreground with logs to stdout/stderr:

```bash
# With default config (/etc/moon.conf)
./moon

# With custom config
./moon --config /path/to/config.yaml
```

### Daemon Mode (Background)

Run as a background daemon with file-based logging:

```bash
# With default config
./moon --daemon

# With custom config
./moon --daemon --config /path/to/config.yaml

# Shorthand
./moon -d --config /path/to/config.yaml
```

Daemon mode features:

- Detaches from terminal and runs in background
- Logs written to `/var/log/moon/main.log` (or path specified in config)
- PID file written to `/var/run/moon.pid`
- Process continues after terminal closes
- Graceful shutdown via SIGTERM/SIGINT

### Development Mode

For development with auto-reload, you can use tools like `air` or `reflex`:

```bash
# Install air
go install github.com/air-verse/air@latest

# Run with auto-reload
air
```

## Testing

### Run All Tests

```bash
go test ./... -v
```

### Run Tests with Coverage

```bash
go test ./... -coverprofile=coverage.txt -covermode=atomic

# Generate HTML coverage report
go tool cover -html=coverage.txt -o coverage.html
```

### Run Tests with Race Detector

```bash
go test ./... -race -v
```

### Using Test Runner Script

```bash
# Run all tests
./samples/test-runner.sh

# Run with coverage
./samples/test-runner.sh coverage

# Run with race detector
./samples/test-runner.sh race

# Run benchmarks
./samples/test-runner.sh bench
```

### Test with Docker

```bash
# Run tests
docker run --rm -v "$(pwd):/app" -w /app golang:1.24 \
  sh -c "go test ./... -v"

# Run tests with coverage
docker run --rm -v "$(pwd):/app" -w /app golang:1.24 \
  sh -c "go test ./... -coverprofile=coverage.txt -covermode=atomic"
```

## Verification

After installation, verify Moon is working:

```bash
# Console mode (for local development/testing)
./moon --config /path/to/moon.conf &

# Check health endpoint
curl http://localhost:6006/health

# Expected response: {"status":"healthy"}
```

### Run API Demo

The quickest way to verify Moon is working:

```bash
# The demo script will auto-start a server if needed
./samples/api-demo.sh
```

For manual verification with a running server:

```bash
# Start Moon in one terminal (console mode)
./moon --config /path/to/moon.conf

# Check health endpoint in another terminal
curl http://localhost:6006/health

# Expected response: {"status":"healthy","database":"sqlite","collections":0}
```

## Deployment

### systemd Service (Linux)

A systemd service file is provided at `samples/moon.service`.

#### Installation Steps

```bash
# 1. Create moon user and directories
sudo useradd -r -s /bin/false moon
sudo mkdir -p /opt/moon /var/log/moon /var/run
sudo chown moon:moon /opt/moon /var/log/moon

# 2. Copy binary
sudo cp moon /usr/local/bin/moon
sudo chmod +x /usr/local/bin/moon

# 3. Setup configuration
sudo cp samples/moon.conf /etc/moon.conf
sudo nano /etc/moon.conf  # Edit and set jwt.secret

# 4. Install systemd service
sudo cp samples/moon.service /etc/systemd/system/
sudo systemctl daemon-reload

# 5. Start service
sudo systemctl start moon

# 6. Enable on boot
sudo systemctl enable moon

# 7. Check status
sudo systemctl status moon

# View logs
sudo journalctl -u moon -f
```

#### Service Management

```bash
# Start service
sudo systemctl start moon

# Stop service
sudo systemctl stop moon

# Restart service
sudo systemctl restart moon

# Check status
sudo systemctl status moon

# View logs
sudo journalctl -u moon -f

# View recent logs
sudo journalctl -u moon -n 100
```

### Manual Daemon

For systems without systemd, run manually in daemon mode:

```bash
# Create required directories
sudo mkdir -p /opt/moon /var/log/moon /var/run
sudo chown $(whoami):$(whoami) /opt/moon /var/log/moon /var/run

# Copy binary
sudo cp moon /usr/local/bin/moon
sudo chmod +x /usr/local/bin/moon

# Setup configuration
sudo cp samples/moon.conf /etc/moon.conf
nano /etc/moon.conf  # Edit and set jwt.secret

# Start in daemon mode
moon --daemon --config /etc/moon.conf

# Check PID
cat /var/run/moon.pid

# Stop daemon
kill $(cat /var/run/moon.pid)
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable moon
sudo systemctl start moon
sudo systemctl status moon
```

### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  moon:
    build: .
    ports:
      - "6006:6006"
    volumes:
      - moon-data:/opt/moon
      - moon-logs:/var/log/moon
      - ./moon.conf:/etc/moon.conf:ro
    restart: unless-stopped

volumes:
  moon-data:
  moon-logs:
```

Run:

```bash
docker-compose up -d
```

## Troubleshooting

### Build Issues

**Problem:** `go: module not found`

**Solution:** Run `go mod download` and `go mod tidy`

**Problem:** CGO-related errors with SQLite

**Solution:** Ensure you have a C compiler installed:

```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS (install Xcode Command Line Tools)
xcode-select --install
```

### Runtime Issues

**Problem:** `Failed to load configuration: JWT secret is required`

**Solution:** Set `jwt.secret` in your configuration file:

```yaml
jwt:
  secret: "your-secure-secret-key"
```

**Problem:** `Failed to connect to database`

**Solution:** Check your database configuration in the YAML file and ensure the database server is running.

**Problem:** Permission denied when accessing SQLite database

**Solution:** Ensure the Moon process has write permissions to the database file and its directory:

```bash
chmod 755 /path/to/database/directory
chmod 644 /path/to/database/moon.db
```

**Problem:** Permission denied writing PID file in daemon mode

**Solution:** Ensure the moon user has write permissions to `/var/run`:

```bash
sudo chown moon:moon /var/run
# Or run as root (not recommended)
```

### Port Already in Use

**Problem:** `bind: address already in use`

**Solution:** Change the port in your configuration file:

```yaml
server:
  port: 9090
```

Or find and stop the conflicting process:

```bash
lsof -i :6006
kill <PID>
```

## Next Steps

- Read the [Usage Guide](USAGE.md) for API documentation and examples
- Check out [sample configurations](samples/) for configuration examples
- Review [SPEC.md](SPEC.md) for architecture and design details
- See the [README](README.md) for project overview

## Additional Resources

- **Configuration Examples:** See `samples/moon.conf` and `samples/moon-full.conf`
- **API Demo Script:** Run `samples/api-demo.sh` for a comprehensive API walkthrough
- **Test Scripts:** Use `samples/test-runner.sh` for various testing scenarios
- **Project Repository:** [https://github.com/thalib/moon](https://github.com/thalib/moon)
