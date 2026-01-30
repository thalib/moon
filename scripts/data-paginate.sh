# Data pagination test script for Moon (hardcoded URLs)

echo "[1] List products with limit=1:"
curl -s -X GET "http://localhost:6006/api/v1/products:list?limit=1" | jq . || curl -s -X GET "http://localhost:6006/api/v1/products:list?limit=1"
echo

echo "[2] List products with limit=1 and after=<CURSOR>:"
curl -s -X GET "http://localhost:6006/api/v1/products:list?limit=1&after=<CURSOR>" | jq . || curl -s -X GET "http://localhost:6006/api/v1/products:list?limit=1&after=<CURSOR>"
echo
