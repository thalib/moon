# Data API test script for Moon (hardcoded URLs)

echo "[1] Create a product record:"
curl -s -X POST "http://localhost:6006/api/v1/products:create" \
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
curl -s -X GET "http://localhost:6006/api/v1/products:list" | jq .


echo "[3] Get a product record (replace <ID>):"
curl -s -X GET "http://localhost:6006/api/v1/products:get?id=01KG4BC716AACQFN757DENC4BE" | jq . 


echo "[4] Update a product record (replace <ID>):"
curl -s -X POST "http://localhost:6006/api/v1/products:update" \
	-H "Content-Type: application/json" \
	-d '{
		"id": "01KG4BC716AACQFN757DENC4BE",
		"data": {
			"price": 1199.99,
			"stock": 45
		}
	}' | jq .


echo "[5] Delete a product record (replace <ID>):"
curl -s -X POST "http://localhost:6006/api/v1/products:destroy" \
	-H "Content-Type: application/json" \
	-d '{"id": "01KG4BC716AACQFN757DENC4BE"}' | jq . 