
# Collection API test script for Moon (hardcoded URLs)

echo "[1] List collections:"
curl -s -X GET "http://localhost:6006/api/v1/collections:list" | jq . 

echo "[2] Create 'products' collection:"
curl -s -X POST "http://localhost:6006/api/v1/collections:create" \
	-H "Content-Type: application/json" \
	-d '{
		"name": "products",
		"columns": [
			{"name": "name", "type": "string", "required": true},
			{"name": "description", "type": "text", "required": false},
			{"name": "price", "type": "float", "required": true},
			{"name": "stock", "type": "integer", "required": true}
		]
	}' | jq .

echo "[3] Get 'products' collection schema:"
curl -s -X GET "http://localhost:6006/api/v1/collections:get?name=products" | jq . || curl -s -X GET "http://localhost:6006/api/v1/collections:get?name=products"
echo

echo "[4] Update 'products' collection (add 'category' column):"
curl -s -X POST "http://localhost:6006/api/v1/collections:update" \
	-H "Content-Type: application/json" \
	-d '{
		"name": "products",
		"add_columns": [
			{"name": "category", "type": "text", "required": false}
		]
	}' | jq .


echo "[5] Destroy 'products' collection:"
curl -s -X POST "http://localhost:6006/api/v1/collections:destroy" \
	-H "Content-Type: application/json" \
	-d '{"name": "products"}' | jq .