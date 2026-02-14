# itemservicecentral Documentation

## Overview

itemservicecentral is a configuration-driven REST API service that provides JSON Schema validated CRUD operations and index-style queries. Data is stored in PostgreSQL using JSONB columns, and access can be secured with JWT-based authentication.

Each table defined in the YAML configuration file gets its own set of REST endpoints for creating, reading, updating, and deleting items. Tables use a partition key (PK) and an optional range key (RK) to identify items, similar to DynamoDB's key model. Global Secondary Indexes (GSIs) can be defined on JSON fields within the data to enable efficient query patterns beyond the base table keys.

Key features:

- **Configuration-driven** — define tables, schemas, indexes, and validation in a single YAML file
- **JSON Schema validation** — every write is validated against a per-table JSON Schema
- **Flexible key model** — PK-only tables for simple lookups, PK+RK tables for hierarchical data
- **Global Secondary Indexes** — query data by alternate key fields stored in the JSONB payload
- **Field projection** — control which fields are returned per request or by default
- **Cursor-based pagination** — paginate through large result sets
- **JWT authentication** — optional RS256 JWT validation with JWKS support
- **Docker-ready** — ships as a single static binary in a distroless container

## CLI Commands

itemservicecentral provides four subcommands:

### `api`

Starts the HTTP API server. Connects to PostgreSQL, runs migrations, and begins serving requests.

```bash
go run . api -config config.yaml -db-url "postgres://user:pass@localhost:5432/dbname?sslmode=disable" -port 8080
```

Flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-config` | `ISC_CONFIG` | `config.yaml` | Path to YAML config file |
| `-port` | `ISC_PORT` | `8080` | Server port (overrides config file) |
| `-db-url` | `ISC_DATABASE_URL` | (required) | PostgreSQL connection string |

### `validate`

Validates the YAML configuration file and compiles all JSON schemas without starting the server. Useful for CI pipelines.

```bash
go run . validate -config config.yaml
```

### `migrate`

Runs database migrations (creates tables and indexes) without starting the server.

```bash
go run . migrate -config config.yaml -db-url "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```

### `version`

Prints the application version and exits.

```bash
go run . version
```

## Configuration

The service is configured via a YAML file. See [example-config.yaml](example-config.yaml) for a complete example.

### Server Section

```yaml
server:
  port: 8080
  jwt:
    enabled: true
    jwksUrl: "https://auth.example.com/.well-known/jwks.json"
    issuer: "https://auth.example.com"
    audience: "my-api"
```

- **port** — HTTP listen port (default: `8080`). Can be overridden with `-port` flag or `ISC_PORT`.
- **jwt.enabled** — when `true`, all requests require a valid Bearer token.
- **jwt.jwksUrl** — URL to fetch RS256 public keys (required when JWT is enabled).
- **jwt.issuer** — expected `iss` claim (optional).
- **jwt.audience** — expected `aud` claim (optional).

### Tables Section

Each entry in `tables` defines a REST resource with its own database table.

```yaml
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[A-Za-z_][A-Za-z0-9._-]*$"
    schema:
      type: object
      properties:
        userId:
          type: string
        name:
          type: string
        email:
          type: string
      required:
        - userId
        - name
    defaultFields:
      - userId
      - name
    allowTableScan: true
    indexes:
      - name: by_email
        pk:
          field: email
        projection:
          - userId
          - email
        allowIndexScan: false
```

#### Table Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Table name. Must match `^[a-z][a-z0-9_]*$`. |
| `pk.field` | Yes | JSON field name used as partition key. Must match `^[A-Za-z][A-Za-z0-9_-]*$`. |
| `pk.pattern` | Yes | Regex pattern that PK values in URLs must match. |
| `rk` | No | Range key configuration (same structure as `pk`). Enables PK+RK composite keys. |
| `rk.field` | When `rk` set | JSON field name used as range key. Must differ from `pk.field`. |
| `rk.pattern` | When `rk` set | Regex pattern that RK values in URLs must match. |
| `schema` | Yes | JSON Schema (Draft 2020-12 compatible) for validating request bodies. |
| `defaultFields` | No | Fields returned by default when no `?fields=` parameter is provided. PK and RK fields are always included. |
| `allowTableScan` | No | When `true`, enables the `GET /v1/{table}/_items` full table scan endpoint. Default: `false`. |
| `indexes` | No | List of Global Secondary Index definitions. |

#### Index Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Index name. Must match `^[a-z][a-z0-9_]*$`. Must be unique within the table. |
| `pk.field` | Yes | JSON field from the data payload to use as the index partition key. Must differ from the base table PK and RK fields. |
| `rk` | No | Optional range key for the index (same structure as `pk`). Must differ from base PK, base RK, and index PK fields. |
| `projection` | No | List of fields to include when querying this index. PK and RK fields are always included. If omitted, all fields are returned. |
| `allowIndexScan` | No | When `true`, enables the `GET /v1/{table}/_index/{name}/_items` full index scan endpoint. Default: `false`. |

## Data Model

### Storage

Each configured table maps to a PostgreSQL table with the following columns:

| Column | Type | Description |
|--------|------|-------------|
| `pk` | `TEXT NOT NULL` | Partition key value |
| `rk` | `TEXT` (NOT NULL for PK+RK tables) | Range key value |
| `data` | `JSONB NOT NULL` | JSON payload (PK/RK fields stripped before storage) |
| `created_at` | `TIMESTAMPTZ` | Row creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | Last update timestamp |

- **PK-only tables** have a `PRIMARY KEY (pk)` and `rk` is nullable.
- **PK+RK tables** have a `PRIMARY KEY (pk, rk)` and `rk` is NOT NULL.

### Key Handling

When an item is written (PUT/PATCH), the PK and RK field values are stripped from the JSON body before being stored in the `data` column. They are stored separately in the `pk` and `rk` columns.

When an item is read (GET), the PK and RK values are injected back into the JSON response using the configured field names.

If PK or RK fields are present in the request body, they must match the values in the URL path.

### Global Secondary Indexes

GSIs create PostgreSQL indexes on JSONB field expressions. They are sparse: only rows where the index PK field (and RK field, if configured) is present in the `data` column are indexed.

## API Reference

All endpoints use the `/v1/{table}` prefix. Responses are JSON with `Content-Type: application/json`.

Error responses have the format:

```json
{"error": "error message"}
```

### Item Endpoints

#### GET — Retrieve an item

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

#### PUT — Create or replace an item

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

#### PATCH — Partial update (JSON Merge Patch)

```
PATCH /v1/{table}/data/{pk}/_item
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

#### DELETE — Delete an item

```
DELETE /v1/{table}/data/{pk}/_item
DELETE /v1/{table}/data/{pk}/{rk}/_item
```

```bash
# Delete a user
curl -X DELETE http://localhost:8080/v1/users/data/user123/_item

# Delete an order line
curl -X DELETE http://localhost:8080/v1/orders/data/order1/line1/_item
```

**Response:** `204 No Content`.

### List Endpoints

List endpoints return paginated results in an envelope:

```json
{
  "items": [ ... ],
  "nextCursor": "..."
}
```

The `nextCursor` field is present only when more results are available.

#### List items in a partition

```
GET /v1/{table}/data/{pk}/_items
```

Lists all items that share the given PK value. For PK-only tables, this returns at most one item. For PK+RK tables, this returns all items with that PK.

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `limit` | Maximum number of items per page (default: 50) |
| `cursor` | Pagination cursor from a previous response |
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

#### Table scan

```
GET /v1/{table}/_items
```

Returns all items in the table. Only available when `allowTableScan: true` is set in the configuration.

```bash
# Scan all users (with pagination)
curl "http://localhost:8080/v1/users/_items?limit=20"

# Next page
curl "http://localhost:8080/v1/users/_items?limit=20&cursor=eyJwayI6InVzZXIxMjMifQ"
```

### Index Endpoints

#### Query an index

```
GET /v1/{table}/_index/{indexName}/{indexPk}/_items
```

Queries a GSI by its partition key value. Returns items that have the specified value in the index PK field.

Query parameters: same as list endpoints (`limit`, `cursor`, `fields`, and RK filters when the index has an RK).

```bash
# Find all orders for a customer
curl http://localhost:8080/v1/orders/_index/by_customer/cust1/_items

# Find items by status
curl http://localhost:8080/v1/items/_index/by_status/active/_items
```

#### Index scan

```
GET /v1/{table}/_index/{indexName}/_items
```

Returns all items that have the index PK field present. Only available when `allowIndexScan: true` is set on the index.

```bash
curl http://localhost:8080/v1/items/_index/by_status/_items
```

#### Get a single item by index (PK+RK indexes only)

```
GET /v1/{table}/_index/{indexName}/{indexPk}/{indexRk}/_item
```

Retrieves a single item by its index PK and RK values. Only available for indexes that have an RK configured.

```bash
curl http://localhost:8080/v1/orders/_index/by_status/active/2024-01-15/_item
```

## Validation

### JSON Schema

Each table requires a `schema` field containing a JSON Schema definition. All PUT and PATCH (after merge) request bodies are validated against this schema. Schemas are written inline in YAML and support JSON Schema Draft 2020-12 features.

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

## Pagination

List and scan endpoints use cursor-based pagination:

1. Make a request with an optional `limit` parameter (default: 50)
2. If more results exist, the response includes a `nextCursor` string
3. Pass `nextCursor` as the `cursor` query parameter in the next request
4. When `nextCursor` is absent from the response, there are no more pages

Cursors are opaque base64url-encoded strings. They encode the PK (and RK for composite key tables) of the last item in the current page.

## Projection and Filtering

### Field Projection (`fields` parameter)

Use the `?fields=` query parameter to request only specific fields:

```bash
curl "http://localhost:8080/v1/users/data/user123/_item?fields=name,email"
```

PK and RK fields are always included in the response regardless of the `fields` parameter.

### Default Fields (`defaultFields`)

If `defaultFields` is configured for a table and no `?fields=` parameter is provided, only the listed default fields (plus PK/RK) are returned.

### Index Projection

Indexes can define a `projection` list. When querying through an index, only the projected fields (plus PK/RK) are returned from the data. The `?fields=` parameter can further narrow the result within the projected fields.

## Docker

Build and run using Docker:

```bash
docker build -t itemservicecentral .
docker run -p 8080:8080 \
  -e ISC_CONFIG=/config.yaml \
  -e ISC_DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=disable" \
  -v ./config.yaml:/config.yaml:ro \
  itemservicecentral
```

The Docker image uses a distroless base for minimal attack surface.