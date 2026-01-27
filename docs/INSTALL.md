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

# Set up configuration
cp samples/.env.example .env
# Edit .env and set MOON_JWT_SECRET to a secure value

# Build and run
go build -o moon ./cmd/moon
./moon
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

EXPOSE 8080
CMD ["./moon"]
```

Build and run:

```bash
docker build -t moon:latest .
docker run -p 8080:8080 -e MOON_JWT_SECRET=your-secret moon:latest
```

## Configuration

Moon uses a flexible configuration system that supports both file-based and environment variable configuration.

### Configuration Priority

1. Environment variables (highest priority)
2. Configuration file (`config.yaml`)
3. Default values (lowest priority)

### Setup Configuration

#### Option 1: Using Environment Variables (Recommended for Production)

```bash
# Copy the example .env file
cp samples/.env.example .env

# Edit .env and set required variables
nano .env  # or use your preferred editor
```

**Required Environment Variables:**

- `MOON_JWT_SECRET`: JWT secret key for authentication (REQUIRED)

**Optional Environment Variables:**

- `MOON_SERVER_HOST`: Server host address (default: `0.0.0.0`)
- `MOON_SERVER_PORT`: Server port (default: `8080`)
- `MOON_DATABASE_CONNECTION_STRING`: Database connection string (default: `sqlite://moon.db`)

#### Option 2: Using Configuration File

```bash
# Copy the example config file
cp samples/config.example.yaml config.yaml

# Edit config.yaml
nano config.yaml
```

**Important:** If using `config.yaml`, you still need to set `MOON_JWT_SECRET` via environment variable in production for security.

### Database Configuration

Moon supports three database backends:

#### SQLite (Default)

```bash
# Environment variable
MOON_DATABASE_CONNECTION_STRING=sqlite://moon.db

# Or in config.yaml
database:
  connection_string: "sqlite://moon.db"
```

SQLite is used by default if no database is configured. It's perfect for:
- Development and testing
- Single-server deployments
- Embedded applications

#### PostgreSQL

```bash
# Environment variable
MOON_DATABASE_CONNECTION_STRING=postgres://user:password@localhost:5432/moon

# Or in config.yaml
database:
  connection_string: "postgres://user:password@localhost:5432/moon"
```

#### MySQL

```bash
# Environment variable
MOON_DATABASE_CONNECTION_STRING=mysql://user:password@localhost:3306/moon

# Or in config.yaml
database:
  connection_string: "mysql://user:password@localhost:3306/moon"
```

## Running Moon

### Standard Run

```bash
./moon
```

### Run with Custom Configuration File

```bash
MOON_CONFIG=/path/to/config.yaml ./moon
```

### Run in Background

```bash
# Using nohup
nohup ./moon > moon.log 2>&1 &

# Or using systemd (see Deployment section below)
```

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
# Check health endpoint
curl http://localhost:8080/health

# Expected response: {"status":"healthy"}
```

### Run API Demo

```bash
# Start Moon in one terminal
./moon

# Run demo in another terminal
./samples/api-demo.sh
```

## Deployment

### systemd Service (Linux)

Create a systemd service file:

```bash
sudo nano /etc/systemd/system/moon.service
```

```ini
[Unit]
Description=Moon Dynamic Headless Engine
After=network.target

[Service]
Type=simple
User=moon
WorkingDirectory=/opt/moon
Environment="MOON_JWT_SECRET=your-production-secret"
Environment="MOON_DATABASE_CONNECTION_STRING=sqlite:///opt/moon/data/moon.db"
ExecStart=/opt/moon/moon
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
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
      - "8080:8080"
    environment:
      - MOON_JWT_SECRET=${MOON_JWT_SECRET}
      - MOON_DATABASE_CONNECTION_STRING=sqlite://data/moon.db
    volumes:
      - moon-data:/app/data
    restart: unless-stopped

volumes:
  moon-data:
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

**Solution:** Set `MOON_JWT_SECRET` environment variable:
```bash
export MOON_JWT_SECRET=your-secret-key
./moon
```

**Problem:** `Failed to connect to database`

**Solution:** Check your database connection string format and ensure the database server is running.

**Problem:** Permission denied when accessing SQLite database

**Solution:** Ensure the Moon process has write permissions to the database file and its directory:
```bash
chmod 755 /path/to/database/directory
chmod 644 /path/to/database/moon.db
```

### Port Already in Use

**Problem:** `bind: address already in use`

**Solution:** Change the port or stop the conflicting service:
```bash
# Change port
export MOON_SERVER_PORT=9090
./moon

# Or find and stop the conflicting process
lsof -i :8080
kill <PID>
```

## Next Steps

- Read the [Usage Guide](USAGE.md) for API documentation and examples
- Check out [sample scripts](../samples/README.md) for common workflows
- Review [SPEC.md](../SPEC.md) for architecture and design details
- See the [README](../README.md) for project overview

## Additional Resources

- **Configuration Examples:** See `samples/config.example.yaml` and `samples/.env.example`
- **API Demo Script:** Run `samples/api-demo.sh` for a comprehensive API walkthrough
- **Test Scripts:** Use `samples/test-runner.sh` for various testing scenarios
- **Project Repository:** [https://github.com/thalib/moon](https://github.com/thalib/moon)
