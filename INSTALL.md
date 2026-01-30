# Moon Installation Guide

## Prerequisites

- **Docker** (for building)
- **Git**

## Quick Start

Clone, build, and install:

```bash
git clone https://github.com/thalib/moon.git
cd moon
```

## Docker Deployment (Recommended)

Moon can be run in a Docker container for consistent, portable deployments.

```bash
# Build Docker Image
# From the repository root:
sudo docker build -t moon:latest .
```

This creates a minimal Docker image using a multi-stage build:

```bash
# Prepare host directories
mkdir -pv ./temp/docker-data

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
## Stop / Remove any existing container (ignore errors if not present)
sudo docker stop moon && sudo docker rm -f moon
```

## Host Installation

Use the provided installation script:

```bash
sudo ./build.sh
sudo ./install.sh
```

The build script:
- Compiles the Moon binary using Docker
- Version information is defined in the codebase as constants

This script:

- Creates moon system user
- Sets up directories (`/opt/moon`, `/var/log/moon`, `/var/lib/moon`)
- Installs binary to `/usr/local/bin/moon`
- Copies configuration to `/etc/moon.conf`
- Installs and enables systemd service
- Starts Moon service

### Verification

Test the running service:

```bash
# Test health endpoint
curl http://localhost:6006/health

# Expected response:
# {
#   "status": "live",
#   "name": "moon",
#   "version": "1.99"
# }

# Check collections
curl http://localhost:6006/api/v1/collections:list
```

## Testing

```bash
# Run All Tests
./samples/test-runner.sh
```

Test Options

```bash
./samples/test-runner.sh unit      # Unit tests only
./samples/test-runner.sh coverage  # With coverage report
./samples/test-runner.sh race      # Race detection
./samples/test-runner.sh bench     # Benchmarks
```

Manual Testing

```bash
go test ./... -v
go test ./... -coverprofile=coverage.txt -covermode=atomic
go test ./... -race -v
```
