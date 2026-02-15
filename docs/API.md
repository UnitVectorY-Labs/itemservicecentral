# API Reference

All endpoints are rooted at `/v1/{table}` and return JSON (`Content-Type: application/json`).

Error response format:

```json
{"error": "error message"}
```

## Swagger / OpenAPI Endpoints (Optional)

When Swagger support is enabled, each table exposes:

- `GET /v1/{table}/_swagger` (embedded Swagger UI HTML)
- `GET /v1/{table}/_openapi` (generated OpenAPI YAML)

These endpoints are public (no JWT required), even when JWT is enabled for API operations.

## Item Endpoints

### GET - Retrieve an item

Primary Key-only tables:

```
GET /v1/{table}/data/{primaryKey}/_item
```

Composite key tables (Primary Key + Range Key):

```
GET /v1/{table}/data/{primaryKey}/{rangeKey}/_item
```

Optional query parameters:

- `fields`: comma-separated fields to return

### PUT - Create or replace an item

Primary Key-only tables:

```
PUT /v1/{table}/data/{primaryKey}/_item
```

Composite key tables:

```
PUT /v1/{table}/data/{primaryKey}/{rangeKey}/_item
```

The request body is validated against the table schema. If Primary Key or Range Key fields are present in the body, they must match URL values.

### PATCH - Partial update (JSON Merge Patch)

Primary Key-only tables:

```
PATCH /v1/{table}/data/{primaryKey}/_item
```

Composite key tables:

```
PATCH /v1/{table}/data/{primaryKey}/{rangeKey}/_item
```

Applies [RFC 7396 JSON Merge Patch](https://tools.ietf.org/html/rfc7396) to the existing item and validates the merged result.

- Set a field by including it.
- Remove a field by setting it to `null`.
- Item must already exist (`404` if not found).

### DELETE - Delete an item

Primary Key-only tables:

```
DELETE /v1/{table}/data/{primaryKey}/_item
```

Composite key tables:

```
DELETE /v1/{table}/data/{primaryKey}/{rangeKey}/_item
```

Returns `204 No Content`.

## List Endpoints

List responses are paginated:

```json
{
  "items": [...],
  "_meta": {
    "nextPageToken": "...",
    "previousPageToken": "..."
  }
}
```

`_meta` appears only when pagination tokens exist.

### Partition query

```
GET /v1/{table}/data/{primaryKey}/_items
```

Returns all items that share a Primary Key value.

- Primary Key-only tables: at most one item.
- Composite key tables: all matching Range Key rows for that Primary Key.

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `limit` | Max items per page (default `50`) |
| `pageToken` | Pagination token |
| `fields` | Comma-separated fields to return |
| `rkBeginsWith` | Range Key starts with prefix (composite tables only) |
| `rkGt` | Range Key greater than value |
| `rkGte` | Range Key greater than or equal to value |
| `rkLt` | Range Key less than value |
| `rkLte` | Range Key less than or equal to value |

### Table scan

```
GET /v1/{table}/_items
```

Returns all items in a table. Available only when `allowTableScan: true`.

## Index Endpoints

### Index query

```
GET /v1/{table}/_index/{indexName}/{indexPrimaryKey}/_items
```

Queries an index by index Primary Key.

Query parameters are the same as list endpoints (`limit`, `pageToken`, `fields`, and `rk*` filters if the index has a Range Key).

### Index scan

```
GET /v1/{table}/_index/{indexName}/_items
```

Returns all items with the index Primary Key present. Available only when `allowIndexScan: true`.

### Index get (indexes with Range Key only)

```
GET /v1/{table}/_index/{indexName}/{indexPrimaryKey}/{indexRangeKey}/_item
```

Returns a single item by index Primary Key and index Range Key.

## Pagination

Token-based pagination flow:

1. Request with optional `limit`.
2. Read `_meta.nextPageToken` when present.
3. Pass token as `pageToken` on the next request.
4. When `_meta` is absent, paging is complete.

Page tokens are opaque base64url values encoding the last returned key position.

## Field Projection

`?fields=` limits returned fields.

Primary Key and Range Key fields are always returned, even when omitted from `fields`.

### Index Projection

Index projection types:

- `ALL`: return all fields.
- `KEYS_ONLY`: return table and index key fields.
- `INCLUDE`: return table keys plus `nonKeyAttributes`.

`fields` can further narrow projected output.

## Validation

### Key Value Rules

URL key values must:

- be non-empty,
- be at most 512 characters,
- match `^[A-Za-z_][A-Za-z0-9._-]*$`,
- match configured key `pattern` expressions.

### JSON Key Rules

JSON object keys at any nesting depth must match:

```
^[A-Za-z0-9][A-Za-z0-9_-]*$
```

### Schema Rules

Table schemas must use the supported restricted subset:

- top-level schema must be `type: object`,
- every object level must set `additionalProperties: false` (equivalent to `allowAdditionalProperties: false`),
- extension features such as `$ref` are rejected.

## Authentication

When `server.jwt.enabled: true`, requests must include:

```
Authorization: Bearer <token>
```

Requirements:

- RS256 signing
- keys loaded from configured JWKS URL (cached 5 minutes)
- optional `issuer` and `audience` checks when configured
