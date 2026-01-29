#!/bin/bash
# Aggregation API test script for Moon

BASE_URL="http://localhost:6006/api/v1"

echo "=== Moon Aggregation API Test ==="
echo

# Create orders collection
echo "[1] Creating 'orders' collection..."
curl -s -X POST "$BASE_URL/collections:create" \
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

# Insert 10 sample orders
echo "[2] Inserting 10 sample orders..."
for i in {1..10}; do
	TOTAL=$((100 + i * 25))
	SUBTOTAL=$((TOTAL * 90 / 100))
	TAX=$((TOTAL - SUBTOTAL))
	
	curl -s -X POST "$BASE_URL/orders:create" \
		-H "Content-Type: application/json" \
		-d "{
			\"data\": {
				\"order_id\": \"ORD-$(printf '%04d' $i)\",
				\"customer_name\": \"Customer $i\",
				\"total\": $TOTAL.00,
				\"subtotal\": $SUBTOTAL.00,
				\"tax\": $TAX.00,
				\"products\": {\"items\": $i}
			}
		}" > /dev/null
done
echo "✓ Created 10 orders"
echo

# List all orders
echo "[3] Listing all orders:"
curl -s -X GET "$BASE_URL/orders:list" | jq '.data[] | {order_id, customer_name, total, subtotal, tax}'
echo

# Aggregation: Count
echo "[4] Aggregation - Count all orders:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:count")
echo "$RESULT" | jq .
COUNT=$(echo "$RESULT" | jq -r '.value')
echo "→ Total orders: $COUNT"
echo

# Aggregation: Sum of total
echo "[5] Aggregation - Sum of 'total' field:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:sum?field=total")
echo "$RESULT" | jq .
SUM=$(echo "$RESULT" | jq -r '.value')
echo "→ Sum of all order totals: \$$SUM"
echo

# Aggregation: Average of total
echo "[6] Aggregation - Average of 'total' field:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:avg?field=total")
echo "$RESULT" | jq .
AVG=$(echo "$RESULT" | jq -r '.value')
echo "→ Average order total: \$$AVG"
echo

# Aggregation: Min of total
echo "[7] Aggregation - Minimum 'total' field:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:min?field=total")
echo "$RESULT" | jq .
MIN=$(echo "$RESULT" | jq -r '.value')
echo "→ Minimum order total: \$$MIN"
echo

# Aggregation: Max of total
echo "[8] Aggregation - Maximum 'total' field:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:max?field=total")
echo "$RESULT" | jq .
MAX=$(echo "$RESULT" | jq -r '.value')
echo "→ Maximum order total: \$$MAX"
echo

# Aggregation on other fields
echo "[9] Aggregation - Sum of 'tax' field:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:sum?field=tax")
echo "$RESULT" | jq .
TAX_SUM=$(echo "$RESULT" | jq -r '.value')
echo "→ Total tax collected: \$$TAX_SUM"
echo

echo "[10] Aggregation - Average of 'subtotal' field:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:avg?field=subtotal")
echo "$RESULT" | jq .
SUBTOTAL_AVG=$(echo "$RESULT" | jq -r '.value')
echo "→ Average subtotal: \$$SUBTOTAL_AVG"
echo

# Aggregation with filters
echo "[11] Aggregation with filters - Count orders with total > 200:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:count?total[gt]=200")
echo "$RESULT" | jq .
COUNT_FILTERED=$(echo "$RESULT" | jq -r '.value')
echo "→ Orders with total > \$200: $COUNT_FILTERED"
echo

echo "[12] Aggregation with filters - Sum of orders with total >= 200:"
RESULT=$(curl -s -X GET "$BASE_URL/orders:sum?field=total&total[gte]=200")
echo "$RESULT" | jq .
SUM_FILTERED=$(echo "$RESULT" | jq -r '.value')
echo "→ Sum of orders with total >= \$200: \$$SUM_FILTERED"
echo

echo "=== Test Complete ==="
echo
echo "Summary:"
echo "  Total Orders: $COUNT"
echo "  Sum of Totals: \$$SUM"
echo "  Average Total: \$$AVG"
echo "  Min Total: \$$MIN"
echo "  Max Total: \$$MAX"
echo "  Total Tax: \$$TAX_SUM"
