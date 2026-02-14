# API Reference

All endpoints use the `/v1/{table}` prefix. Responses are JSON with `Content-Type: application/json`.

Error responses have the format:

```json
{"error": "error message"}
```

## Item Endpoints

### GET — Retrieve an item

**PK-only table:**
```
GET /v1/{table}/data/{pk}/_item
```

**PK+RK table:**
```
GET /v1/{table}/data/{pk}/{rk}/_item
```

Optional query parameters:
- `fields` — comma-separated list of fields to return

```bash
# Get a user
curl http://localhost:8080/v1/users/data/user123/_item

# Get a user with field projection
curl "http://localhost:8080/v1/users/data/user123/_item?fields=name,email"

# Get an order line item
curl http://localhost:8080/v1/orders/data/order1/line1/_item
```

**Response:** `200 OK` with the item JSON, or `404 Not Found`.

```json
{
  "userId": "user123",
  "name": "Alice",
  "email": "alice@example.com"
}
```

### PUT — Create or replace an item

**PK-only table:**
```
PUT /v1/{table}/data/{pk}/_item
```

**PK+RK table:**
```
PUT /v1/{table}/data/{pk}/{rk}/_item
```

The request body is validated against the table's JSON Schema. If the PK/RK fields are present in the body, they must match the URL path values.

```bash
# Create a user
curl -X PUT http://localhost:8080/v1/users/data/user123/_item \
  -H "Content-Type: application/json" \
  -d '{"userId": "user123", "name": "Alice", "email": "alice@example.com"}'

# Create an order line item
curl -X PUT http://localhost:8080/v1/orders/data/order1/line1/_item \
  -H "Content-Type: application/json" \
  -d '{"orderId": "order1", "lineId": "line1", "customerId": "cust1", "amount": 42.50}'
```

**Response:** `200 OK` with the saved item (including PK/RK fields).

### PATCH — Partial update (JSON Merge Patch)

**PK-only table:**
```
PATCH /v1/{table}/data/{pk}/_item
```

**PK+RK table:**
```
PATCH /v1/{table}/data/{pk}/{rk}/_item
```

Applies an [RFC 7396 JSON Merge Patch](https://tools.ietf.org/html/rfc7396) to the existing item. The merged result is validated against the JSON Schema before saving.

- Set a field value by including it in the patch.
- Remove a field by setting its value to `null`.
- The item must already exist (returns `404` if not found).

```bash
# Update a user's email
curl -X PATCH http://localhost:8080/v1/users/data/user123/_item \
  -H "Content-Type: application/json" \
  -d '{"email": "newalice@example.com"}'

# Remove a field
curl -X PATCH http://localhost:8080/v1/users/data/user123/_item \
  -H "Content-Type: application/json" \
  -d '{"email": null}'
```

**Response:** `200 OK` with the full merged item, or `404 Not Found`.

### DELETE — Delete an item

**PK-only table:**
```
DELETE /v1/{table}/data/{pk}/_item
```

**PK+RK table:**
```
DELETE /v1/{table}/data/{pk}/{rk}/_item
```

```bash
# Delete a user
curl -X DELETE http://localhost:8080/v1/users/data/user123/_item

# Delete an order line
curl -X DELETE http://localhost:8080/v1/orders/data/order1/line1/_item
```

**Response:** `204 No Content`.

## List Endpoints

List endpoints return paginated results in an envelope:

```json
{
  "items": [...],
  "_meta": {
    "nextPageToken": "...",
    "previousPageToken": "..."
  }
}
```

The `_meta` object is present only when there are pagination tokens. The `nextPageToken` field indicates more results are available.

### Partition query

```
GET /v1/{table}/data/{pk}/_items
```

Lists all items that share the given PK value. For PK-only tables, this returns at most one item. For PK+RK tables, this returns all items with that PK.

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `limit` | Maximum number of items per page (default: 50) |
| `pageToken` | Pagination token from a previous response |
| `fields` | Comma-separated list of fields to return |
| `rkBeginsWith` | Filter: RK starts with this prefix (PK+RK tables only) |
| `rkGt` | Filter: RK greater than value |
| `rkGte` | Filter: RK greater than or equal to value |
| `rkLt` | Filter: RK less than value |
| `rkLte` | Filter: RK less than or equal to value |

```bash
# List all lines for an order
curl http://localhost:8080/v1/orders/data/order1/_items

# List lines with prefix filter
curl "http://localhost:8080/v1/orders/data/order1/_items?rkBeginsWith=line"

# Paginate with limit
curl "http://localhost:8080/v1/orders/data/order1/_items?limit=10"
```

### Table scan

```
GET /v1/{table}/_items
```

Returns all items in the table. Only available when `allowTableScan: true` is set in the configuration.

```bash
# Scan all users (with pagination)
curl "http://localhost:8080/v1/users/_items?limit=20"

# Next page
curl "http://localhost:8080/v1/users/_items?limit=20&pageToken=dXNlcjEyM3w"
```

## Index Endpoints

### Index query

```
GET /v1/{table}/_index/{indexName}/{indexPk}/_items
```

Queries a GSI by its partition key value. Returns items that have the specified value in the index PK field.

Query parameters: same as list endpoints (`limit`, `pageToken`, `fields`, and RK filters when the index has an RK).

```bash
# Find all orders for a customer
curl http://localhost:8080/v1/orders/_index/by_customer/cust1/_items

# Find items by status
curl http://localhost:8080/v1/items/_index/by_status/active/_items
```

### Index scan

```
GET /v1/{table}/_index/{indexName}/_items
```

Returns all items that have the index PK field present. Only available when `allowIndexScan: true` is set on the index.

```bash
curl http://localhost:8080/v1/items/_index/by_status/_items
```

### Index get (PK+RK indexes only)

```
GET /v1/{table}/_index/{indexName}/{indexPk}/{indexRk}/_item
```

Retrieves a single item by its index PK and RK values. Only available for indexes that have an RK configured.

```bash
curl http://localhost:8080/v1/orders/_index/by_status/active/2024-01-15/_item
```

## Pagination

List and scan endpoints use token-based pagination:

1. Make a request with an optional `limit` parameter (default: 50).
2. If more results exist, the response includes a `_meta` object with a `nextPageToken` string.
3. Pass `nextPageToken` as the `pageToken` query parameter in the next request.
4. When `_meta` is absent from the response, there are no more pages.

Page tokens are opaque base64url-encoded strings. They encode the PK (and RK for composite key tables) of the last item in the current page.

## Field Projection

Use the `?fields=` query parameter to request only specific fields:

```bash
curl "http://localhost:8080/v1/users/data/user123/_item?fields=name,email"
```

PK and RK fields are always included in the response regardless of the `fields` parameter.

### Index Projection

Indexes can define a `projection` with a `type` of `ALL`, `KEYS_ONLY`, or `INCLUDE`. When querying through an index:

- **`ALL`** (or no projection): all fields are returned.
- **`KEYS_ONLY`**: only the base PK/RK and index PK/RK fields are returned.
- **`INCLUDE`**: only the base PK/RK fields plus the listed `nonKeyAttributes` are returned.

The `?fields=` parameter can further narrow the result within the projected fields.

## Validation

### Key Value Rules

URL path values for PK, RK, and index keys must satisfy:

- Not empty
- Maximum 512 characters
- Match pattern: `^[A-Za-z_][A-Za-z0-9._-]*$`
- Additionally match the table's configured `pattern` regex

### JSON Key Constraints

All JSON object keys at any nesting depth in request bodies must match:

```
^[A-Za-z0-9][A-Za-z0-9_-]*$
```

Keys starting with an underscore or containing special characters are rejected.

## Authentication

When `server.jwt.enabled` is `true`, all requests must include a valid JWT in the `Authorization` header:

```
Authorization: Bearer <token>
```

Requirements:

- Signing algorithm: RS256
- Keys are fetched from the configured JWKS URL and cached for 5 minutes
- If `issuer` is configured, the `iss` claim must match
- If `audience` is configured, the `aud` claim must match

When JWT is disabled, all requests are allowed without authentication.
