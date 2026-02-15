# Swagger / OpenAPI

itemservicecentral can expose per-table Swagger UI and dynamically generated OpenAPI YAML.

This feature is disabled by default.

## Enablement

Swagger endpoints are enabled when `server.swagger.enabled: true` in YAML, or by runtime override:

- CLI flag: `-swagger-enabled`
- Environment variable: `ISC_SWAGGER_ENABLED=true`

If both are set, the CLI flag takes precedence.

## HTTP Endpoints

When enabled, each table gets two public (no-auth) endpoints:

- `GET /v1/{table}/_swagger`  
  Serves the embedded `swagger.html` UI.
- `GET /v1/{table}/_openapi`  
  Serves the table-specific OpenAPI YAML.

These two endpoints are intentionally unauthenticated so documentation can be viewed without a token.

## JWT-Aware OpenAPI

When `server.jwt.enabled: true`, generated OpenAPI includes HTTP bearer auth (`bearerAuth`) so Swagger UI prompts for a bearer token.

No OAuth flow is configured.

## Dynamic Generation and Caching

OpenAPI YAML is generated lazily on first request per table and then cached in memory.

- cache key: table name
- invalidation: process restart (configuration changes require restart)

This avoids regenerating the same document for repeated requests.

## Coverage

Generated OpenAPI includes table-specific routes based on table and index configuration, including:

- `GET/PUT/PATCH/DELETE` single-item routes (`_item`) for PK-only and PK+RK tables
- `GET` partition list route (`data/{pk}/_items`)
- optional table scan (`GET /v1/{table}/_items`) when `allowTableScan: true`
- index query routes (`GET /v1/{table}/_index/{index}/{indexPk}/_items`)
- optional index scan routes (`GET /v1/{table}/_index/{index}/_items`) when `allowIndexScan: true`
- index single-item route for index PK+RK (`GET /v1/{table}/_index/{index}/{indexPk}/{indexRk}/_item`)

Request/response schemas include the configured table JSON Schema and the list/error envelope structures.

Path parameters in generated OpenAPI use configured key field names, not shorthand placeholders.  
Example: `/v1/users/data/{userId}/_item` (not `{pk}`).

Operations are emitted in a stable logical order: item CRUD first, then partition/table list routes, then index routes.

Query parameters are emitted per endpoint, including:

- `fields`
- `limit`
- `pageToken`
- range-key filters when applicable (`rkBeginsWith`, `rkGt`, `rkGte`, `rkLt`, `rkLte`)

PATCH payload schema is generated from the table schema with table key fields required and other top-level fields optional (nullable) for merge-patch behavior.

## CLI Generation

You can generate OpenAPI YAML without starting the API or connecting to PostgreSQL:

```bash
go run . swagger -config config.yaml -table users
```

By default YAML is written to stdout.

To write to a file:

```bash
go run . swagger -config config.yaml -table users -output users-openapi.yaml
```
