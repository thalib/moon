#!/bin/bash
# Data API test script for Moon
# Supports PREFIX environment variable for custom URL prefixes
# Usage: PREFIX=/api/v1 ./scripts/data.sh
# Usage: PREFIX="" ./scripts/data.sh (for no prefix)

# Default to empty prefix if not set
PREFIX=${PREFIX:-}

# Base URL
BASE_URL="http://localhost:6006${PREFIX}"

echo "Testing Moon Data API"
echo "Using prefix: ${PREFIX:-<empty>}"
echo "Base URL: ${BASE_URL}"
echo ""

echo "[1] Create a product record:"
curl -s -X POST "${BASE_URL}/products:create" \
	-H "Content-Type: application/json" \
	-d '{
		"data": {
			"name": "Laptop",
			"description": "High-performance laptop",
			"price": 1299.99,
			"stock": 50
		}
	}' | jq .


echo "[2] List all product records:"
curl -s -X GET "${BASE_URL}/products:list" | jq .


echo "[3] Get a product record (replace <ID>):"
curl -s -X GET "${BASE_URL}/products:get?id=01KG4BC716AACQFN757DENC4BE" | jq . 


echo "[4] Update a product record (replace <ID>):"
curl -s -X POST "${BASE_URL}/products:update" \
	-H "Content-Type: application/json" \
	-d '{
		"id": "01KG4BC716AACQFN757DENC4BE",
		"data": {
			"price": 1199.99,
			"stock": 45
		}
	}' | jq .


echo "[5] Delete a product record (replace <ID>):"
curl -s -X POST "${BASE_URL}/products:destroy" \
	-H "Content-Type: application/json" \
	-d '{"id": "01KG4BC716AACQFN757DENC4BE"}' | jq .
 