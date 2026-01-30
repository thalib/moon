
#!/bin/bash
# Moon Build Script (Docker-based)
set -e

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root."
    echo ""
    echo "Please run this script with sudo:"
    echo "  sudo ./build.sh"
    echo ""
    exit 1
fi

# Check for Docker
if ! command -v docker >/dev/null 2>&1; then
    echo "[ERROR] Docker is not installed or not in PATH."
    exit 1
fi

# Read major version from VERSION file
MAJOR_VERSION="1"
if [ -f "VERSION" ]; then
    MAJOR_VERSION=$(cat VERSION)
fi

# Get short git commit hash
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "[INFO] Building Moon binary using Docker..."
echo "[INFO] Version: ${MAJOR_VERSION}-${GIT_COMMIT}"

docker run --rm \
  -v "$(pwd):/app" \
  -v "$(pwd)/.gocache:/gocache" \
  -w /app \
  -e GOCACHE=/gocache \
  golang:latest sh -c "go build -buildvcs=false -ldflags \"-X main.MajorVersion=${MAJOR_VERSION} -X main.GitCommit=${GIT_COMMIT}\" -o moon ./cmd/moon"

