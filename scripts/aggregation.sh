#!/bin/bash
# Aggregation API test script for Moon

# Supports PREFIX environment variable for custom URL prefixes
# Usage: PREFIX=/api/v1 ./scripts/aggregation.sh
# Usage: PREFIX="" ./scripts/aggregation.sh (for no prefix)

# Default to empty prefix if not set
PREFIX=${PREFIX:-}

# Base URL
BASE_URL="http://localhost:6006${PREFIX}"

echo "Testing Moon Aggregation API"
echo "Using prefix: ${PREFIX:-<empty>}"
echo "Base URL: ${BASE_URL}"
echo ""

echo "=== Moon Aggregation API Test ==="
echo

# Create orders collection
echo "[1] Creating 'orders' collection..."
curl -s -X POST "${BASE_URL}/collections:create" \
	-H "Content-Type: application/json" \
	-d '{
		"name": "orders",
		"columns": [
			{"name": "order_id", "type": "string", "required": true},
			{"name": "customer_name", "type": "string", "required": true},
			{"name": "total", "type": "float", "required": true},
			{"name": "subtotal", "type": "float", "required": true},
			{"name": "tax", "type": "float", "required": true},
			{"name": "products", "type": "json", "required": false}
		]
	}' | jq .
echo

echo "[2] Inserting 5 sample orders..."
curl -s -X POST "${BASE_URL}/orders:create" \
	-H "Content-Type: application/json" \
	-d '{
		"data": {
			"order_id": "ORD-0001",
			"customer_name": "Customer 1",
			"total": 125.00,
			"subtotal": 112.50,
			"tax": 12.50,
			"products": "{\"items\": 1}"
		}
	}' > /dev/null
curl -s -X POST "${BASE_URL}/orders:create" \
	-H "Content-Type: application/json" \
	-d '{
		"data": {
			"order_id": "ORD-0002",
			"customer_name": "Customer 2",
			"total": 150.00,
			"subtotal": 135.00,
			"tax": 15.00,
			"products": "{\"items\": 2}"
		}
	}' > /dev/null
curl -s -X POST "${BASE_URL}/orders:create" \
	-H "Content-Type: application/json" \
	-d '{
		"data": {
			"order_id": "ORD-0003",
			"customer_name": "Customer 3",
			"total": 175.00,
			"subtotal": 157.50,
			"tax": 17.50,
			"products": "{\"items\": 3}"
		}
	}' > /dev/null
curl -s -X POST "${BASE_URL}/orders:create" \
	-H "Content-Type: application/json" \
	-d '{
		"data": {
			"order_id": "ORD-0004",
			"customer_name": "Customer 4",
			"total": 200.00,
			"subtotal": 180.00,
			"tax": 20.00,
			"products": "{\"items\": 4}"
		}
	}' > /dev/null
curl -s -X POST "${BASE_URL}/orders:create" \
	-H "Content-Type: application/json" \
	-d '{
		"data": {
			"order_id": "ORD-0005",
			"customer_name": "Customer 5",
			"total": 225.00,
			"subtotal": 202.50,
			"tax": 22.50,
			"products": "{\"items\": 5}"
		}
	}' > /dev/null
echo "✓ Created 5 orders"
echo

# List all orders
echo "[3] Listing all orders:"
curl -s -X GET "${BASE_URL}/orders:list" | jq .
echo

# Aggregation: Count
echo "[4] Aggregation - Count all orders:"
curl -s -X GET "${BASE_URL}/orders:count" | jq .
echo

# Aggregation: Sum of total
echo "[5] Aggregation - Sum of 'total' field:"
curl -s -X GET "${BASE_URL}/orders:sum?field=total" | jq .
echo

# Aggregation: Average of total
echo "[6] Aggregation - Average of 'total' field:"
curl -s -X GET "${BASE_URL}/orders:avg?field=total" | jq .
echo

# Aggregation: Min of total
echo "[7] Aggregation - Minimum 'total' field:"
curl -s -X GET "${BASE_URL}/orders:min?field=total" | jq .
echo

# Aggregation: Max of total
echo "[8] Aggregation - Maximum 'total' field:"
curl -s -X GET "${BASE_URL}/orders:max?field=total" | jq .
echo

# Aggregation on other fields
echo "[9] Aggregation - Sum of 'tax' field:"
curl -s -X GET "${BASE_URL}/orders:sum?field=tax" | jq .
echo

echo "[10] Aggregation - Average of 'subtotal' field:"
curl -s -X GET "${BASE_URL}/orders:avg?field=subtotal" | jq .
echo

# Aggregation with filters
echo "[11] Aggregation with filters - Count orders with total > 200:"
curl --globoff -s -X GET "${BASE_URL}/orders:count?total[gt]=200" | jq .
echo

echo "[12] Aggregation with filters - Sum of orders with total >= 200:"
curl --globoff -s -X GET "${BASE_URL}/orders:sum?field=total&total[gte]=200" | jq .
echo

echo "=== Test Complete ==="
