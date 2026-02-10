### Get Schema

```bash
curl -s -X GET "http://localhost:6006/products:schema" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "collection": "products",
  "fields": [
    {
      "name": "id",
      "type": "string",
      "nullable": false
    },
    {
      "name": "title",
      "type": "string",
      "nullable": false
    },
    {
      "name": "price",
      "type": "decimal",
      "nullable": false
    },
    {
      "name": "details",
      "type": "string",
      "nullable": true
    },
    {
      "name": "quantity",
      "type": "integer",
      "nullable": true
    },
    {
      "name": "brand",
      "type": "string",
      "nullable": true
    }
  ],
  "total": 0
}
```

### Create a New Record

```bash
curl -s -X POST "http://localhost:6006/products:create" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "data": {
          "title": "Wireless Mouse",
          "price": "29.99",
          "details": "Ergonomic wireless mouse",
          "quantity": 10,
          "brand": "Wow"
        }
      }
    ' | jq .
```

**Response (201 Created):**

```json
{
  "data": {
    "brand": "Wow",
    "details": "Ergonomic wireless mouse",
    "id": "01KH2TQPMZS2VY5V0V25AKBGYS",
    "price": "29.99",
    "quantity": 10,
    "title": "Wireless Mouse"
  },
  "message": "Record created successfully with id 01KH2TQPMZS2VY5V0V25AKBGYS"
}
```

### Get All Records

```bash
curl -s -X GET "http://localhost:6006/products:list" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH2TQPMZS2VY5V0V25AKBGYS",
      "price": "29.99",
      "quantity": 10,
      "title": "Wireless Mouse"
    }
  ],
  "total": 1,
  "next_cursor": null,
  "limit": 15
}
```

### Get a Single Record

```bash
curl -s -X GET "http://localhost:6006/products:get?id=$ULID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": {
    "brand": "Wow",
    "details": "Ergonomic wireless mouse",
    "id": "01KH2TQPMZS2VY5V0V25AKBGYS",
    "price": "29.99",
    "quantity": 10,
    "title": "Wireless Mouse"
  }
}
```

### Update an Existing Record

```bash
curl -s -X POST "http://localhost:6006/products:update" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "id": "$ULID",
        "data": {
          "price": "6000.00"
        }
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "data": {
    "id": "01KH2TQPMZS2VY5V0V25AKBGYS",
    "price": "6000.00"
  },
  "message": "Record 01KH2TQPMZS2VY5V0V25AKBGYS updated successfully"
}
```

### Delete a Record

```bash
curl -s -X POST "http://localhost:6006/products:destroy" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "id": "$ULID"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "message": "Record 01KH2TQPMZS2VY5V0V25AKBGYS deleted successfully"
}
```
