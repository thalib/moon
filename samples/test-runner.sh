#!/bin/bash
# Test runner script for Moon project

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Header
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Moon - Test Runner${NC}"
echo -e "${BLUE}================================${NC}\n"

# Check if we're in the project root
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Please run this script from the project root."
    exit 1
fi

# Run tests based on arguments
case "${1:-all}" in
    unit)
        print_info "Running unit tests..."
        go test ./... -v -short
        print_success "Unit tests completed!"
        ;;
    
    coverage)
        print_info "Running tests with coverage..."
        go test ./... -coverprofile=coverage.txt -covermode=atomic
        print_success "Coverage report generated: coverage.txt"
        
        if command -v go > /dev/null 2>&1; then
            print_info "Generating HTML coverage report..."
            go tool cover -html=coverage.txt -o coverage.html
            print_success "HTML coverage report: coverage.html"
        fi
        ;;
    
    race)
        print_info "Running tests with race detector..."
        go test ./... -race -v
        print_success "Race condition tests completed!"
        ;;
    
    bench)
        print_info "Running benchmarks..."
        go test ./... -bench=. -benchmem
        print_success "Benchmarks completed!"
        ;;
    
    all|*)
        print_info "Running all tests..."
        go test ./... -v
        print_success "All tests completed!"
        ;;
esac

echo ""
print_success "Test run finished successfully!"
