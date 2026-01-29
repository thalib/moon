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
sudo docker build -t moon:latest .
```

This creates a minimal Docker image using a multi-stage build:

- Builder stage compiles the Go binary
- Runtime stage uses `scratch` base (minimal footprint)
- Final image is ~20MB

### Run with Docker

```bash
# Prepare host directories
mkdir -pv ./temp/docker-data

# Remove any existing container (ignore errors if not present)
sudo docker rm -f moon

# Run Moon container
sudo docker run -d \
  --name moon \
  -p 6006:6006 \
  -v $(pwd)/samples/moon.conf:/etc/moon.conf:ro \
  -v $(pwd)/temp/docker-data/data:/opt/moon \
  -v $(pwd)/temp/docker-data/log:/var/log/moon \
  moon:latest
```

```bash
sudo docker stop moon && sudo docker rm -f moon
```

### Verification

Test the running container:

```bash
# Check container logs
sudo docker logs moon

# Test health endpoint
curl http://localhost:6006/health

# Check collections
curl http://localhost:6006/api/v1/collections:list
```
