
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

echo "[5] Update 'products' collection (rename 'description' to 'details'):"
curl -s -X POST "http://localhost:6006/api/v1/collections:update" \
	-H "Content-Type: application/json" \
	-d '{
		"name": "products",
		"rename_columns": [
			{"old_name": "description", "new_name": "details"}
		]
	}' | jq .

echo "[6] Update 'products' collection (modify 'details' type to string):"
curl -s -X POST "http://localhost:6006/api/v1/collections:update" \
	-H "Content-Type: application/json" \
	-d '{
		"name": "products",
		"modify_columns": [
			{"name": "details", "type": "string"}
		]
	}' | jq .

echo "[7] Update 'products' collection (remove 'category' column):"
curl -s -X POST "http://localhost:6006/api/v1/collections:update" \
	-H "Content-Type: application/json" \
	-d '{
		"name": "products",
		"remove_columns": ["category"]
	}' | jq .

echo "[8] Update 'products' collection (combined operations):"
curl -s -X POST "http://localhost:6006/api/v1/collections:update" \
	-H "Content-Type: application/json" \
	-d '{
		"name": "products",
		"add_columns": [
			{"name": "brand", "type": "string", "required": false}
		],
		"rename_columns": [
			{"old_name": "stock", "new_name": "quantity"}
		],
		"modify_columns": [
			{"name": "price", "type": "float", "nullable": false}
		]
	}' | jq .

echo "[9] Destroy 'products' collection:"
curl -s -X POST "http://localhost:6006/api/v1/collections:destroy" \
	-H "Content-Type: application/json" \
	-d '{"name": "products"}' | jq .