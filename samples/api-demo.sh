#!/bin/bash
# Sample script demonstrating Moon API usage
# Requires: curl, jq (optional for JSON formatting)

# Configuration
BASE_URL="${MOON_URL:-http://localhost:8080}"
API_VERSION="v1"
API_BASE="$BASE_URL/api/$API_VERSION"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper function for printing section headers
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
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
        curl -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_BASE$endpoint" \
            -w "\nHTTP Status: %{http_code}\n" \
            2>/dev/null
    else
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
    echo -e "${RED}Error: Moon server is not running at $BASE_URL${NC}"
    echo "Please start the server first with: ./moon"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}"

# 1. List all collections (should be empty initially)
print_header "1. List Collections"
api_call "GET" "/collections:list" "" "Listing all collections"

# 2. Create a new collection
print_header "2. Create a Collection"
api_call "POST" "/collections:create" \
    '{
        "name": "products",
        "columns": [
            {"name": "name", "type": "text", "required": true},
            {"name": "description", "type": "text", "required": false},
            {"name": "price", "type": "real", "required": true},
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
read -p "Do you want to delete the 'products' collection? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    api_call "POST" "/collections:destroy" \
        '{"name": "products"}' \
        "Destroying 'products' collection"
fi

print_header "Demo Complete"
echo "All operations completed successfully!"
echo "For more information, see docs/USAGE.md"
