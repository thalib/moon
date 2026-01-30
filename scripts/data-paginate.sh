#!/bin/bash
# Data pagination test script for Moon
# Supports PREFIX environment variable for custom URL prefixes
# Usage: PREFIX=/api/v1 ./scripts/data-paginate.sh
# Usage: PREFIX="" ./scripts/data-paginate.sh (for no prefix)

# Default to empty prefix if not set
PREFIX=${PREFIX:-}

# Base URL
BASE_URL="http://localhost:6006${PREFIX}"

echo "Testing Moon Data Pagination API"
echo "Using prefix: ${PREFIX:-<empty>}"
echo "Base URL: ${BASE_URL}"
echo ""

echo "[1] List products with limit=1:"
curl -s -X GET "${BASE_URL}/products:list?limit=1" | jq . || curl -s -X GET "${BASE_URL}/products:list?limit=1"
echo

echo "[2] List products with limit=1 and after=<CURSOR>:"
curl -s -X GET "${BASE_URL}/products:list?limit=1&after=<CURSOR>" | jq . || curl -s -X GET "${BASE_URL}/products:list?limit=1&after=<CURSOR>"
echo

