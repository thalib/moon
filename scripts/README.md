# api-check.py

A compact automated API test runner for Moon API endpoints.

## Features

- Runs API tests from JSON files
- Outputs Markdown-formatted results
- Supports token auth, placeholder replacement, and health checks

## Usage

```sh
python api-check.py [-i TESTFILE.json] [-o OUTDIR] [-t TESTDIR]
```

- `-i`: Input test JSON file (default: all in `tests/`)
- `-o`: Output directory for Markdown (default: `./out`)
- `-t`: Directory with test JSONs (default: `./tests`)

PREREQUEST: make sure you run this tool against a fresh installation of Moon server.

```sh
cd 
python api-check.py -i ./tests/020-auth.json

# or run all 
python api-check.py
```

Results are saved as Markdown in the output directory. this file can be copied to `cmd\moon\internal\handlers\templates\md\` to update the documentations

```sh
cp .\out\*.md ..\cmd\moon\internal\handlers\templates\md\
```

## Requirements

- Python 3.x
- `requests` library

## Test Files

Test files are JSON files describing a sequence of API requests and expected behaviors.

### Structure

- `serverURL`: Base URL of the API server (e.g., <http://localhost:8080>)
- `docURL`: URL used for documentation output (optional)
- `prefix`: (optional) API prefix (e.g., /api)
- `username`/`password`: (optional) for login/auth tests
- `tests`: Array of test objects, each with:
  - `name`: Short description
  - `cmd`: HTTP method (GET, POST, etc.)
  - `endpoint`: API endpoint (e.g., /users:list)
  - `headers`: (optional) Dict of headers
  - `data`: (optional) Request body (dict or string)
  - `details`/`notes`: (optional) Markdown for docs

### Example

```json
{
  "serverURL": "http://localhost:8080",
  "docURL": "https://api.example.com",
  "prefix": "/api",
  "username": "admin",
  "password": "moonadmin12#",
  "tests": [
    {
      "name": "List users",
      "cmd": "GET",
      "endpoint": "/users:list"
    },
    {
      "name": "Create user",
      "cmd": "POST",
      "endpoint": "/users:create",
      "headers": {"Content-Type": "application/json"},
      "data": {"username": "bob", "password": "secret"}
    }
  ]
}
```

You can add placeholders like `$ACCESS_TOKEN`, `$ULID`, and `$NEXT_CURSOR` in endpoints, headers, or data. These will be replaced automatically during test execution.
