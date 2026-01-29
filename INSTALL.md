# Moon Installation Guide

## Prerequisites

- **Docker** (for building)
- **Git**

## Quick Start

Clone, build, and install:

```bash
git clone https://github.com/thalib/moon.git
cd moon
sudo ./build.sh
sudo ./install.sh
```

Run the interactive demo:

```bash
./samples/api-demo.sh
```

## Building

### Automated Build (Recommended)

Use the provided build script with Docker:

```bash
sudo ./build.sh
```

This creates the `moon` binary using Docker, no Go installation required.

### Manual Docker Build

```bash
docker run --rm \
  -v "$(pwd):/app" \
  -v "$(pwd)/.gocache:/gocache" \
  -w /app -e GOCACHE=/gocache \
  golang:latest sh -c "go build -buildvcs=false -o moon ./cmd/moon"
```

### Go Build (Alternative)

If you have Go 1.24+ installed:

```bash
go build -o moon ./cmd/moon
```

## Configuration

Moon uses YAML configuration files exclusively.

**Location:** `/etc/moon.conf` (default)

**Sample configurations:**

- [samples/moon.conf](samples/moon.conf) - Quick-start configuration
- [samples/moon-full.conf](samples/moon-full.conf) - All options documented

**Generate JWT secret:**

```bash
openssl rand -base64 32
```

Edit `/etc/moon.conf` and set `jwt.secret` to the generated value.

## Development Mode

### Console Mode

Run in foreground for testing:

```bash
./moon
```

With custom config:

```bash
./moon --config /path/to/moon.conf
```

### Verify Health

```bash
curl http://localhost:6006/health
```

Expected: `{"status":"healthy","database":"sqlite","collections":0}`

## Testing

### Run All Tests

```bash
./samples/test-runner.sh
```

### Test Options

```bash
./samples/test-runner.sh unit      # Unit tests only
./samples/test-runner.sh coverage  # With coverage report
./samples/test-runner.sh race      # Race detection
./samples/test-runner.sh bench     # Benchmarks
```

### Manual Testing

```bash
go test ./... -v
go test ./... -coverprofile=coverage.txt -covermode=atomic
go test ./... -race -v
```

## Installation

### Automated Installation (Recommended)

Use the provided installation script:

```bash
sudo ./install.sh
```

This script:

- Creates moon system user
- Sets up directories (`/opt/moon`, `/var/log/moon`, `/var/lib/moon`)
- Installs binary to `/usr/local/bin/moon`
- Copies configuration to `/etc/moon.conf`
- Installs and enables systemd service
- Starts Moon service

## Docker Deployment

Moon can be run in a Docker container for consistent, portable deployments.

### Build Docker Image

From the repository root:

```bash
docker build -t moon:latest .
```

This creates a minimal Docker image using a multi-stage build:
- Builder stage compiles the Go binary
- Runtime stage uses `scratch` base (minimal footprint)
- Final image is ~20MB

### Run with Docker

#### Basic Usage

```bash
docker run -d \
  --name moon \
  -p 6006:6006 \
  -v /path/to/moon.conf:/etc/moon.conf \
  -v /path/to/data:/opt/moon \
  moon:latest
```

#### Example with Local Directories

```bash
# Create local directories
mkdir -p ./moon-data ./moon-config

# Create a config file
cat > ./moon-config/moon.conf << EOF
server:
  host: "0.0.0.0"
  port: 6006

database:
  connection: "sqlite"
  database: "/opt/moon/sqlite.db"

logging:
  path: "/var/log/moon"

jwt:
  secret: "your-secret-key-change-in-production"
  expiry: 3600

apikey:
  enabled: false
  header: "X-API-KEY"
EOF

# Run Moon container
docker run -d \
  --name moon \
  -p 6006:6006 \
  -v $(pwd)/moon-config/moon.conf:/etc/moon.conf \
  -v $(pwd)/moon-data:/opt/moon \
  moon:latest

# Check status
curl http://localhost:6006/health
```

#### Docker Compose (Optional)

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  moon:
    build: .
    ports:
      - "6006:6006"
    volumes:
      - ./moon.conf:/etc/moon.conf
      - ./data:/opt/moon
    restart: unless-stopped
```

Run with:

```bash
docker-compose up -d
```

### Volume Mounts

- **Config:** Mount your YAML config at `/etc/moon.conf`
- **Data:** Mount a persistent volume at `/opt/moon/` for SQLite database
- **Logs:** Internal logs are written to `/var/log/moon` (optional mount)

### Environment

The Docker container:
- Runs Moon in console mode (foreground)
- Uses YAML-only configuration (no environment variables)
- Exposes port 6006 by default
- Requires a valid config file at `/etc/moon.conf`
- Persists SQLite data via volume mount to `/opt/moon/`

### Verification

Test the running container:

```bash
# Check container logs
docker logs moon

# Test health endpoint
curl http://localhost:6006/health

# Check collections
curl http://localhost:6006/api/v1/collections:list
```
