# Moon Samples

This directory contains example configuration files and scripts to help you get started with Moon.

## Configuration Files

### moon.conf

Minimal quick-start configuration file showing essential options. Copy this to `/etc/moon.conf` for production use:

```bash
sudo cp samples/moon.conf /etc/moon.conf
sudo nano /etc/moon.conf  # Edit and set jwt.secret
```

### moon-full.conf

Comprehensive configuration file with all available options documented. Use this as a reference for advanced configuration:

```bash
# For reference and advanced setup
cat samples/moon-full.conf
```

**Important:** Always change the `jwt.secret` to a secure random string in production!
Generate with: `openssl rand -base64 32`

## Test Scripts

Test scripts are now located in the `scripts/` directory at the repository root:

- `scripts/health.sh` - Health endpoint testing
- `scripts/collection.sh` - Collection management operations
- `scripts/data.sh` - Data CRUD operations
- `scripts/data-paginate.sh` - Pagination examples
- `scripts/aggregation.sh` - Aggregation endpoint examples

**Usage:**

```bash
# Start Moon server first
./moon --config samples/moon.conf &

# Run individual test scripts
./scripts/health.sh
./scripts/collection.sh
./scripts/data.sh
./scripts/data-paginate.sh
./scripts/aggregation.sh
```

### test-runner.sh (moved to scripts/)

A convenient test runner with multiple modes, now located in `scripts/`:

```bash
# Run all tests
./scripts/test-runner.sh

# Run only unit tests
./scripts/test-runner.sh unit

# Run tests with coverage report
./scripts/test-runner.sh coverage

# Run tests with race detector
./scripts/test-runner.sh race

# Run benchmarks
./scripts/test-runner.sh bench
```

## Quick Start

1. Build Moon:
   ```bash
   go build -o moon ./cmd/moon
   ```

2. Copy and configure:
   ```bash
   cp samples/moon.conf /etc/moon.conf
   # Edit /etc/moon.conf and set jwt.secret
   ```

3. Start the server:
   ```bash
   ./moon --config /etc/moon.conf
   ```

4. Run test scripts:
   ```bash
   ./scripts/health.sh
   ./scripts/collection.sh
   ./scripts/data.sh
   ```

For detailed documentation, see:
- [Installation Guide](../INSTALL.md)
- [API Documentation](http://localhost:6006/doc/) (available when server is running)
- [Project README](../README.md)
