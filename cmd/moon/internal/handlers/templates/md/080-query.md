### Filtering

**Query Option:** `?column[operator]=value`

**Operators:** eq, ne, gt, lt, gte, lte, like, in

```bash
curl -s -X GET "http://localhost:6006/products:list?quantity[gt]=5&brand[eq]=Wow" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH2TQT61EJSB8ZK9HVZRPWV6",
      "price": "29.99",
      "quantity": 10,
      "title": "Wireless Mouse"
    },
    {
      "brand": "Wow",
      "details": "Full HD monitor",
      "id": "01KH2TQTX9WZGE82WJP96K77KT",
      "price": "199.99",
      "quantity": 20,
      "title": "Monitor 21 inch"
    }
  ],
  "total": 2,
  "next_cursor": null,
  "limit": 15
}
```

### Sorting

**Query Option:** `?sort={-field1,field2}`

Sort by `field` (ascending) or `-field` (descending).

```bash
curl -s -X GET "http://localhost:6006/products:list?sort=-quantity,title" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "brand": "Orange",
      "details": "Gaming keyboard",
      "id": "01KH2TQTH15052GWSD6AQZ2FFB",
      "price": "19.99",
      "quantity": 55,
      "title": "USB Keyboard"
    },
    {
      "brand": "Wow",
      "details": "Full HD monitor",
      "id": "01KH2TQTX9WZGE82WJP96K77KT",
      "price": "199.99",
      "quantity": 20,
      "title": "Monitor 21 inch"
    },
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH2TQT61EJSB8ZK9HVZRPWV6",
      "price": "29.99",
      "quantity": 10,
      "title": "Wireless Mouse"
    }
  ],
  "total": 3,
  "next_cursor": null,
  "limit": 15
}
```

### Full-Text Search

**Query Option:** `?q={search_term}` (across all text columns)

Searches across all string/text fields in the collection.

```bash
curl -s -X GET "http://localhost:6006/products:list?q=mouse" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH2TQT61EJSB8ZK9HVZRPWV6",
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

### Field Selection

**Query Option:** `?fields={field1,field2}`

Returns only the specified fields (plus `id` which is always included).

```bash
curl -s -X GET "http://localhost:6006/products:list?fields=quantity,title" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "id": "01KH2TQT61EJSB8ZK9HVZRPWV6",
      "quantity": 10,
      "title": "Wireless Mouse"
    },
    {
      "id": "01KH2TQTH15052GWSD6AQZ2FFB",
      "quantity": 55,
      "title": "USB Keyboard"
    },
    {
      "id": "01KH2TQTX9WZGE82WJP96K77KT",
      "quantity": 20,
      "title": "Monitor 21 inch"
    }
  ],
  "total": 3,
  "next_cursor": null,
  "limit": 15
}
```

### Limit

**Query Option:** `?limit={limit}`

```bash
curl -s -X GET "http://localhost:6006/products:list?limit=2" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH2TQT61EJSB8ZK9HVZRPWV6",
      "price": "29.99",
      "quantity": 10,
      "title": "Wireless Mouse"
    },
    {
      "brand": "Orange",
      "details": "Gaming keyboard",
      "id": "01KH2TQTH15052GWSD6AQZ2FFB",
      "price": "19.99",
      "quantity": 55,
      "title": "USB Keyboard"
    }
  ],
  "total": 3,
  "next_cursor": "01KH2TQTH15052GWSD6AQZ2FFB",
  "limit": 2
}
```

### Pagination

**Query Option:** `?after={cursor}`

 (Response includes `next_cursor` when more results are available.)

```bash
curl -s -X GET "http://localhost:6006/products:list?after=$NEXT_CURSOR&limit=1" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "brand": "Orange",
      "details": "Gaming keyboard",
      "id": "01KH2TQTH15052GWSD6AQZ2FFB",
      "price": "19.99",
      "quantity": 55,
      "title": "USB Keyboard"
    }
  ],
  "total": 3,
  "next_cursor": "01KH2TQTH15052GWSD6AQZ2FFB",
  "limit": 1
}
```
