#!/bin/bash
# Sample script demonstrating Moon API usage
# Requires: curl, jq (optional for JSON formatting)

# Configuration
BASE_URL="${MOON_URL:-http://localhost:6006}"
API_BASE="$BASE_URL/api/v1"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Temporary files for demo
DEMO_DIR="/tmp/moon-api-demo-$$"
DEMO_CONFIG="$DEMO_DIR/moon.conf"
DEMO_DB="$DEMO_DIR/moon.db"
DEMO_LOG_DIR="$DEMO_DIR/logs"
MOON_PID=""
MOON_BINARY=""
CLEANUP_ON_EXIT=true

# Cleanup function
cleanup() {
    if [ "$CLEANUP_ON_EXIT" = true ]; then
        echo -e "\n${YELLOW}Cleaning up...${NC}"
        
        # Stop Moon server if we started it
        if [ -n "$MOON_PID" ]; then
            echo "Stopping Moon server (PID: $MOON_PID)..."
            kill "$MOON_PID" 2>/dev/null || true
            
            # Wait for graceful shutdown (max 5 seconds)
            WAIT_COUNT=0
            while [ $WAIT_COUNT -lt 5 ]; do
                if ! kill -0 "$MOON_PID" 2>/dev/null; then
                    break
                fi
                sleep 1
                WAIT_COUNT=$((WAIT_COUNT + 1))
            done
            
            # Force kill if still running
            if kill -0 "$MOON_PID" 2>/dev/null; then
                echo "Force stopping server..."
                kill -9 "$MOON_PID" 2>/dev/null || true
            fi
            
            wait "$MOON_PID" 2>/dev/null || true
        fi
        
        # Remove temporary files
        if [ -d "$DEMO_DIR" ]; then
            echo "Removing temporary files: $DEMO_DIR"
            rm -rf "$DEMO_DIR"
        fi
        
        echo -e "${GREEN}✓ Cleanup complete${NC}"
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Helper function for printing section headers
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

# Setup demo environment
setup_demo_environment() {
    print_header "Setup"
    
    # Create temporary directory structure
    echo "Creating temporary demo environment..."
    mkdir -p "$DEMO_DIR" "$DEMO_LOG_DIR"
    
    # Create demo configuration
    cat > "$DEMO_CONFIG" << EOF
server:
  host: "127.0.0.1"  # Bind to localhost only for security
  port: 6006

database:
  connection: "sqlite"
  database: "$DEMO_DB"

logging:
  path: "$DEMO_LOG_DIR"

jwt:
  # WARNING: This secret is for demo purposes only!
  # NEVER use this in production - generate a secure secret with: openssl rand -base64 32
  secret: "demo-secret-key-for-testing-only"
  expiry: 3600

apikey:
  enabled: false
  header: "X-API-KEY"
EOF
    
    echo -e "${GREEN}✓ Demo environment created${NC}"
    echo "  Config: $DEMO_CONFIG"
    echo "  Database: $DEMO_DB"
    echo "  Logs: $DEMO_LOG_DIR"
}

# Find or start Moon server
start_moon_server() {
    # Look for moon binary
    if [ -f "./moon" ]; then
        MOON_BINARY="./moon"
    elif [ -f "../moon" ]; then
        MOON_BINARY="../moon"
    elif command -v moon &> /dev/null; then
        MOON_BINARY="moon"
    else
        echo -e "${RED}Error: Moon binary not found${NC}"
        echo "Please build Moon first:"
        echo "  go build -o moon ./cmd/moon"
        exit 1
    fi
    
    echo "Starting Moon server..."
    echo "Command: $MOON_BINARY --config $DEMO_CONFIG"
    
    # Start server in background
    "$MOON_BINARY" --config "$DEMO_CONFIG" > "$DEMO_LOG_DIR/server.log" 2>&1 &
    MOON_PID=$!
    
    # Wait for server to be ready
    echo "Waiting for server to start (PID: $MOON_PID)..."
    MAX_WAIT=10
    WAIT_COUNT=0
    while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
        if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Server is running${NC}"
            return 0
        fi
        sleep 1
        WAIT_COUNT=$((WAIT_COUNT + 1))
        echo -n "."
    done
    
    echo -e "\n${RED}Error: Server failed to start${NC}"
    echo "Check logs at: $DEMO_LOG_DIR/server.log"
    cat "$DEMO_LOG_DIR/server.log"
    exit 1
}

# Helper function for API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "${GREEN}$description${NC}"
    echo "Request: $method $API_BASE$endpoint"
    
    if [ -n "$data" ]; then
        echo "Data: $data"
        echo -n "Command: "
        echo "curl -X \"$method\" -H \"Content-Type: application/json\" -d '$data' \"$API_BASE$endpoint\" -w \"\nHTTP Status: %{http_code}\n\""
        curl -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_BASE$endpoint" \
            -w "\nHTTP Status: %{http_code}\n" \
            2>/dev/null
    else
        echo -n "Command: "
        echo "curl -X \"$method\" \"$API_BASE$endpoint\" -w \"\nHTTP Status: %{http_code}\n\""
        curl -X "$method" \
            "$API_BASE$endpoint" \
            -w "\nHTTP Status: %{http_code}\n" \
            2>/dev/null
    fi
    echo -e "\n"
}

# Check if server is running
print_header "Health Check"
echo "Checking if Moon server is running..."
if ! curl -s "$BASE_URL/health" > /dev/null; then
    echo -e "${YELLOW}Server not running, will start one for the demo${NC}"
    
    # Setup and start server
    setup_demo_environment
    start_moon_server
else
    echo -e "${GREEN}✓ Server is already running${NC}"
    echo -e "${YELLOW}Note: Using existing server. Demo will NOT clean up automatically.${NC}"
    CLEANUP_ON_EXIT=false
fi

# 1. List all collections (should be empty initially)
print_header "1. List Collections"
api_call "GET" "/collections:list" "" "Listing all collections"

# 2. Create a new collection
print_header "2. Create a Collection"
api_call "POST" "/collections:create" \
    '{
        "name": "products",
        "columns": [
            {"name": "name", "type": "string", "required": true},
            {"name": "description", "type": "text", "required": false},
            {"name": "price", "type": "float", "required": true},
            {"name": "stock", "type": "integer", "required": true}
        ]
    }' \
    "Creating 'products' collection"

# 3. Get collection schema
print_header "3. Get Collection Schema"
api_call "GET" "/collections:get?name=products" "" "Retrieving 'products' collection schema"

# 4. Create a record
print_header "4. Create a Record"
api_call "POST" "/products:create" \
    '{
        "data": {
            "name": "Laptop",
            "description": "High-performance laptop",
            "price": 1299.99,
            "stock": 50
        }
    }' \
    "Creating a product record"

# 5. Create another record
print_header "5. Create Another Record"
api_call "POST" "/products:create" \
    '{
        "data": {
            "name": "Mouse",
            "description": "Wireless mouse",
            "price": 29.99,
            "stock": 200
        }
    }' \
    "Creating another product record"

# 6. List all records
print_header "6. List All Records"
api_call "GET" "/products:list" "" "Listing all products"

# 7. List records with pagination
print_header "7. List Records with Pagination"
api_call "GET" "/products:list?limit=1&offset=0" "" "Listing products (limit=1, offset=0)"

# 8. Get a specific record
print_header "8. Get Specific Record"
api_call "GET" "/products:get?id=1" "" "Getting product with ID=1"

# 9. Update a record
print_header "9. Update a Record"
api_call "POST" "/products:update" \
    '{
        "id": 1,
        "data": {
            "price": 1199.99,
            "stock": 45
        }
    }' \
    "Updating product ID=1 (new price and stock)"

# 10. Verify update
print_header "10. Verify Update"
api_call "GET" "/products:get?id=1" "" "Getting updated product with ID=1"

# 11. Delete a record
print_header "11. Delete a Record"
api_call "POST" "/products:destroy" \
    '{
        "id": 2
    }' \
    "Deleting product ID=2"

# 12. Verify deletion
print_header "12. Verify Deletion"
api_call "GET" "/products:list" "" "Listing all products after deletion"

# 13. Update collection schema (add column)
print_header "13. Update Collection Schema"
api_call "POST" "/collections:update" \
    '{
        "name": "products",
        "add_columns": [
            {"name": "category", "type": "text", "required": false}
        ]
    }' \
    "Adding 'category' column to products"

# 14. Verify schema update
print_header "14. Verify Schema Update"
api_call "GET" "/collections:get?name=products" "" "Retrieving updated 'products' schema"

# 15. Clean up - destroy collection
print_header "15. Clean Up"

# Only prompt if we're using an existing server
if [ "$CLEANUP_ON_EXIT" = false ]; then
    read -p "Do you want to delete the 'products' collection? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        api_call "POST" "/collections:destroy" \
            '{"name": "products"}' \
            "Destroying 'products' collection"
    fi
else
    # If we started our own server, always clean up the collection
    api_call "POST" "/collections:destroy" \
        '{"name": "products"}' \
        "Destroying 'products' collection"
fi

print_header "Demo Complete"
echo "All operations completed successfully!"
echo "For more information, see USAGE.md"

# Note: cleanup() will be called automatically on exit due to trap
