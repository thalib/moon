### Collections Create

```bash
curl -s -X POST "http://localhost:6006/collections:create" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "products",
        "columns": [
          {
            "name": "title",
            "type": "string",
            "nullable": false
          },
          {
            "name": "price",
            "type": "integer",
            "nullable": false
          },
          {
            "name": "description",
            "type": "string",
            "nullable": true
          }
        ]
      }
    ' | jq .
```

**Response (201 Created):**

```json
{
  "collection": {
    "name": "products",
    "columns": [
      {
        "name": "title",
        "type": "string",
        "nullable": false,
        "unique": false
      },
      {
        "name": "price",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "description",
        "type": "string",
        "nullable": true,
        "unique": false
      }
    ]
  },
  "message": "Collection 'products' created successfully"
}
```

### Collections List

```bash
curl -s -X GET "http://localhost:6006/collections:list" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "collections": [
    "products"
  ],
  "count": 1
}
```

### Collections Get

```bash
curl -s -X GET "http://localhost:6006/collections:get?name=products" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "collection": {
    "name": "products",
    "columns": [
      {
        "name": "title",
        "type": "string",
        "nullable": false,
        "unique": false
      },
      {
        "name": "price",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "description",
        "type": "string",
        "nullable": true,
        "unique": false
      }
    ]
  }
}
```

### Collections Update - Add Columns

```bash
curl -s -X POST "http://localhost:6006/collections:update" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "products",
        "add_columns": [
          {
            "name": "stock",
            "type": "integer",
            "nullable": false
          },
          {
            "name": "category",
            "type": "string",
            "nullable": false
          }
        ]
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "collection": {
    "name": "products",
    "columns": [
      {
        "name": "title",
        "type": "string",
        "nullable": false,
        "unique": false
      },
      {
        "name": "price",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "description",
        "type": "string",
        "nullable": true,
        "unique": false
      },
      {
        "name": "stock",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "category",
        "type": "string",
        "nullable": false,
        "unique": false
      }
    ]
  },
  "message": "Collection 'products' updated successfully"
}
```

### Collections Update - Rename Columns

```bash
curl -s -X POST "http://localhost:6006/collections:update" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "products",
        "rename_columns": [
          {
            "old_name": "description",
            "new_name": "details"
          }
        ]
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "collection": {
    "name": "products",
    "columns": [
      {
        "name": "title",
        "type": "string",
        "nullable": false,
        "unique": false
      },
      {
        "name": "price",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "details",
        "type": "string",
        "nullable": true,
        "unique": false
      },
      {
        "name": "stock",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "category",
        "type": "string",
        "nullable": false,
        "unique": false
      }
    ]
  },
  "message": "Collection 'products' updated successfully"
}
```

### Collections Update - Modify Columns

```bash
curl -s -X POST "http://localhost:6006/collections:update" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "products",
        "modify_columns": [
          {
            "name": "category",
            "type": "integer",
            "nullable": true
          }
        ]
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "collection": {
    "name": "products",
    "columns": [
      {
        "name": "title",
        "type": "string",
        "nullable": false,
        "unique": false
      },
      {
        "name": "price",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "details",
        "type": "string",
        "nullable": true,
        "unique": false
      },
      {
        "name": "stock",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "category",
        "type": "integer",
        "nullable": true,
        "unique": false
      }
    ]
  },
  "message": "Collection 'products' updated successfully"
}
```

### Collections Update - Remove Columns

```bash
curl -s -X POST "http://localhost:6006/collections:update" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "products",
        "remove_columns": [
          "category"
        ]
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "collection": {
    "name": "products",
    "columns": [
      {
        "name": "title",
        "type": "string",
        "nullable": false,
        "unique": false
      },
      {
        "name": "price",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "details",
        "type": "string",
        "nullable": true,
        "unique": false
      },
      {
        "name": "stock",
        "type": "integer",
        "nullable": false,
        "unique": false
      }
    ]
  },
  "message": "Collection 'products' updated successfully"
}
```

### Collections Update - Combine Operations

```bash
curl -s -X POST "http://localhost:6006/collections:update" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "products",
        "add_columns": [
          {
            "name": "brand",
            "type": "string",
            "nullable": false
          }
        ],
        "rename_columns": [
          {
            "old_name": "stock",
            "new_name": "quantity"
          }
        ],
        "modify_columns": [
          {
            "name": "price",
            "type": "integer",
            "nullable": false
          }
        ]
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "collection": {
    "name": "products",
    "columns": [
      {
        "name": "title",
        "type": "string",
        "nullable": false,
        "unique": false
      },
      {
        "name": "price",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "details",
        "type": "string",
        "nullable": true,
        "unique": false
      },
      {
        "name": "quantity",
        "type": "integer",
        "nullable": false,
        "unique": false
      },
      {
        "name": "brand",
        "type": "string",
        "nullable": false,
        "unique": false
      }
    ]
  },
  "message": "Collection 'products' updated successfully"
}
```

### Collections Destroy

```bash
curl -s -X POST "http://localhost:6006/collections:destroy" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '
      {
        "name": "products"
      }
    ' | jq .
```

**Response (200 OK):**

```json
{
  "message": "Collection 'products' destroyed successfully"
}
```
