#!/bin/bash
#
# Moon Installation Script
# Interactive installation script for Moon - Dynamic Headless Engine
#

set -e  # Exit on error

# Check for root privileges
check_root() {
    echo "Checking for root privileges..."
    if [ "$EUID" -ne 0 ]; then
        echo "This script must be run as root."
        echo ""
        echo "Please run this script with sudo:"
        echo "  sudo ./build.sh"
        echo ""
        exit 1
    fi
    echo "Running with root privileges"
}

check_root

echo 'sudo docker run --rm -v "$(pwd):/app" -v "$(pwd)/.gocache:/gocache" -w /app -e GOCACHE=/gocache golang:latest sh -c "go build -buildvcs=false -o moon ./cmd/moon"'

docker run --rm -v "$(pwd):/app" -v "$(pwd)/.gocache:/gocache" -w /app -e GOCACHE=/gocache golang:latest sh -c "go build -buildvcs=false -o moon ./cmd/moon"
