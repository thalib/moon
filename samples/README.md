# Moon Samples

This directory contains example configuration files and scripts to help you get started with Moon.

## Configuration Files

### config.example.yaml

Example YAML configuration file showing all available options. Copy this to `config.yaml` in the project root and customize as needed:

```bash
cp samples/config.example.yaml config.yaml
```

### .env.example

Example environment variables file. Environment variables take precedence over config file settings. Copy this to `.env` in the project root:

```bash
cp samples/.env.example .env
```

**Important:** Make sure to change the `MOON_JWT_SECRET` to a secure random string in production!

## Scripts

### api-demo.sh

A comprehensive demonstration script that shows all major Moon API operations:
- Creating collections (database tables)
- Managing collection schemas
- CRUD operations on data
- Pagination and filtering

**Usage:**

```bash
# Start the Moon server first
./moon &

# Run the demo
./samples/api-demo.sh
```

The script will walk through:
1. Health check
2. Collection management (create, list, get, update, destroy)
3. Data operations (create, read, update, delete)
4. Schema modifications

### test-runner.sh

A convenient test runner with multiple modes:

```bash
# Run all tests
./samples/test-runner.sh

# Run only unit tests
./samples/test-runner.sh unit

# Run tests with coverage report
./samples/test-runner.sh coverage

# Run tests with race detector
./samples/test-runner.sh race

# Run benchmarks
./samples/test-runner.sh bench
```

## Quick Start

1. Copy configuration files:
   ```bash
   cp samples/.env.example .env
   # Edit .env and set MOON_JWT_SECRET
   ```

2. Build and run Moon:
   ```bash
   go build -o moon ./cmd/moon
   ./moon
   ```

3. Try the API demo:
   ```bash
   ./samples/api-demo.sh
   ```

For detailed documentation, see:
- [Installation Guide](../docs/INSTALL.md)
- [Usage Guide](../docs/USAGE.md)
- [Project README](../README.md)
