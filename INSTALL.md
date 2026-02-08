# Moon Installation Guide

> **⚠️ BREAKING CHANGE**: Authentication is now **mandatory** for all API endpoints.
> All requests (except `/health`) require valid JWT or API Key credentials.
> See the [Authentication Setup](#authentication-setup) section below.

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

The build script: `build.sh`

- Compiles the Moon binary using Docker
- Version information is defined in the codebase as constants

This script: `install.sh`

- Creates moon system user
- Sets up directories (`/opt/moon`, `/var/log/moon`, `/var/lib/moon`)
- Installs binary to `/usr/local/bin/moon`
- Copies configuration to `/etc/moon.conf`
- Installs and enables systemd service
- Starts Moon service

### Verification

Test the running service:

```bash
# Test health endpoint (no auth required)
curl http://localhost:6006/health

# Expected response:
# {
#   "status": "live",
#   "name": "moon",
#   "version": "1.99"
# }
```

## Authentication Setup

Moon requires authentication for all endpoints except `/health`. Follow these steps to set up authentication.

### Step 1: Generate a Secure JWT Secret

```bash
# Generate a secure 32-byte random secret
openssl rand -base64 32
```

### Step 2: Configure Moon

Edit `/etc/moon.conf` (or `samples/moon.conf` for Docker):

```yaml
# JWT Configuration (REQUIRED)
jwt:
  # Paste your generated secret here
  secret: "YOUR_GENERATED_SECRET_HERE"
  access_expiry: 3600      # 1 hour
  refresh_expiry: 604800   # 7 days

# API Key Configuration (Optional but recommended)
apikey:
  enabled: true

# Bootstrap Admin (First-time setup only)
# ⚠️ Remove this section after first login!
auth:
  bootstrap_admin:
    username: "admin"
    email: "admin@example.com"
    password: "change-me-on-first-login"
```

### Step 3: Start Moon and Verify Bootstrap

```bash
# Restart Moon to apply configuration
sudo systemctl restart moon

# Check logs for bootstrap confirmation
sudo journalctl -u moon -f

# You should see:
# "Bootstrap admin user created: admin"
```

### Step 4: Login and Change Password

```bash
# Login with bootstrap credentials
curl -X POST http://localhost:6006/auth:login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "change-me-on-first-login"}'

# Response includes access_token and refresh_token
# {
#   "access_token": "eyJhbGc...",
#   "refresh_token": "eyJhbGc...",
#   "expires_in": 3600,
#   "token_type": "Bearer",
#   "user": { "id": "...", "username": "admin", "role": "admin" }
# }

# Save the access token
export TOKEN="eyJhbGc..."

# Change password immediately
curl -X POST http://localhost:6006/auth:me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "current_password": "change-me-on-first-login",
    "new_password": "YourNewSecurePassword123!"
  }'
```

**Important**: After changing the password, remove the `auth.bootstrap_admin` section from your configuration file.

### Step 5: Creating Additional Users (Optional)

```bash
# Create a read-only user
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "reader",
    "email": "reader@example.com",
    "password": "SecurePass123!",
    "role": "user",
    "can_write": false
  }'

# Create a user with write access
curl -X POST http://localhost:6006/users:create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "writer",
    "email": "writer@example.com", 
    "password": "SecurePass123!",
    "role": "user",
    "can_write": true
  }'
```

### Step 6: Creating API Keys (Optional)

For machine-to-machine integrations:

```bash
# Create an API key for automation
curl -X POST http://localhost:6006/apikeys:create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD Integration",
    "description": "API key for automated deployments",
    "role": "user",
    "can_write": true
  }'

# Response includes the key (shown only once!)
# {
#   "id": "...",
#   "name": "CI/CD Integration",
#   "key": "moon_live_abc123...xyz789",
#   "warning": "Store this key securely. It will not be shown again."
# }

# Use the API key
curl http://localhost:6006/collections:list \
  -H "Authorization: Bearer moon_live_abc123...xyz789"
```

### Step 7: Running the Test Suite

Verify your authentication setup with the test scripts:

```bash
# Run all authentication tests
./scripts/auth-all.sh

# Or run individual test suites
./scripts/auth-jwt.sh       # JWT authentication
./scripts/auth-apikey.sh    # API key authentication
./scripts/auth-rbac.sh      # Role-based access control
./scripts/auth-ratelimit.sh # Rate limiting

# With custom prefix
PREFIX=/api/v1 ./scripts/auth-all.sh
```

## Using Authenticated Endpoints

### JWT Authentication

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:6006/auth:login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "YourPassword"}' | jq -r '.access_token')

# Use token for requests
curl http://localhost:6006/collections:list \
  -H "Authorization: Bearer $TOKEN"

# Refresh token before expiry
curl -X POST http://localhost:6006/auth:refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "YOUR_REFRESH_TOKEN"}'

# Logout
curl -X POST http://localhost:6006/auth:logout \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "YOUR_REFRESH_TOKEN"}'
```

### API Key Authentication

```bash
# Use API key for requests
curl http://localhost:6006/collections:list \
  -H "Authorization: Bearer moon_live_abc123...xyz789"
```

## Security Checklist

- [ ] Generated unique JWT secret (at least 32 characters)
- [ ] Changed bootstrap admin password
- [ ] Removed bootstrap_admin section from config
- [ ] Using HTTPS in production (via reverse proxy)
- [ ] API keys stored securely (not in code)
- [ ] Regular API key rotation scheduled
- [ ] Rate limits configured appropriately

