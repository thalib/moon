**Filtering**

```bash
curl -s -X GET "http://localhost:6006/products:list?quantity[gt]=5&brand[eq]=Wow" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH1KB39V63TKQ95H7P77CQWF",
      "price": "29.99",
      "quantity": 10,
      "title": "Wireless Mouse"
    },
    {
      "brand": "Wow",
      "details": "Full HD monitor",
      "id": "01KH1KB46FVZS4083DCNQG6H1S",
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

**Sorting**

```bash
curl -s -X GET "http://localhost:6006/products:list?sort=-quantity,title" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [
    {
      "brand": "Orange",
      "details": "Gaming keyboard",
      "id": "01KH1KB3PXTFQ1KQ0016SD3HKE",
      "price": "19.99",
      "quantity": 55,
      "title": "USB Keyboard"
    },
    {
      "brand": "Wow",
      "details": "Full HD monitor",
      "id": "01KH1KB46FVZS4083DCNQG6H1S",
      "price": "199.99",
      "quantity": 20,
      "title": "Monitor 21 inch"
    },
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH1KB39V63TKQ95H7P77CQWF",
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

**Full-Text Search**

```bash
curl -s -X GET "http://localhost:6006/products:list?q=mouse" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH1KB39V63TKQ95H7P77CQWF",
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

**Field Selection**

```bash
curl -s -X GET "http://localhost:6006/products:list?fields=quantity,title" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [
    {
      "id": "01KH1KB39V63TKQ95H7P77CQWF",
      "quantity": 10,
      "title": "Wireless Mouse"
    },
    {
      "id": "01KH1KB3PXTFQ1KQ0016SD3HKE",
      "quantity": 55,
      "title": "USB Keyboard"
    },
    {
      "id": "01KH1KB46FVZS4083DCNQG6H1S",
      "quantity": 20,
      "title": "Monitor 21 inch"
    }
  ],
  "total": 3,
  "next_cursor": null,
  "limit": 15
}
```

**Limit**

```bash
curl -s -X GET "http://localhost:6006/products:list?limit=2" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Ergonomic wireless mouse",
      "id": "01KH1KB39V63TKQ95H7P77CQWF",
      "price": "29.99",
      "quantity": 10,
      "title": "Wireless Mouse"
    },
    {
      "brand": "Orange",
      "details": "Gaming keyboard",
      "id": "01KH1KB3PXTFQ1KQ0016SD3HKE",
      "price": "19.99",
      "quantity": 55,
      "title": "USB Keyboard"
    }
  ],
  "total": 3,
  "next_cursor": "01KH1KB3PXTFQ1KQ0016SD3HKE",
  "limit": 2
}
```

**Pagination**

```bash
curl -s -X GET "http://localhost:6006/products:list?after=$NEXT_CURSOR&limit=1" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [
    {
      "brand": "Orange",
      "details": "Gaming keyboard",
      "id": "01KH1KB3PXTFQ1KQ0016SD3HKE",
      "price": "19.99",
      "quantity": 55,
      "title": "USB Keyboard"
    }
  ],
  "total": 3,
  "next_cursor": "01KH1KB3PXTFQ1KQ0016SD3HKE",
  "limit": 1
}
```

**Combined Query - Filter, sort, and limit**

```bash
curl -s -X GET "http://localhost:6006/products:list?quantity[gte]=10&price[lt]=100&sort=-price&limit=5" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [],
  "total": 0,
  "next_cursor": null,
  "limit": 5
}
```

**Combined Query - Search with category filter and field selection**

```bash
curl -s -X GET "http://localhost:6006/products:list?q=laptop&brand[eq]=Wow&fields=title,price,quantity" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [],
  "total": 0,
  "next_cursor": null,
  "limit": 15
}
```

**Combined Query - Multiple filters with pagination**

```bash
curl -s -X GET "http://localhost:6006/products:list?price[gte]=100&quantity[gt]=0&sort=-price&limit=1&after=$NEXT_CURSOR" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

***Response (200 OK):***

```json
{
  "data": [
    {
      "brand": "Wow",
      "details": "Full HD monitor",
      "id": "01KH1KB46FVZS4083DCNQG6H1S",
      "price": "199.99",
      "quantity": 20,
      "title": "Monitor 21 inch"
    }
  ],
  "total": 3,
  "next_cursor": "01KH1KB46FVZS4083DCNQG6H1S",
  "limit": 1
}
```
