# Swagger / OpenAPI

itemservicecentral can expose per-table Swagger UI and dynamically generated OpenAPI YAML allowing for interactive documentation for testing and development.

This feature is disabled by default and can be enabled by setting `server.swagger.enabled: true` in YAML.

## HTTP Endpoints

When enabled, each table gets two public (no-auth) endpoints:

- `GET /v1/{table}/_swagger` serves the embedded `swagger.html` UI.
- `GET /v1/{table}/_openapi` serves the table-specific OpenAPI YAML.

These two endpoints are intentionally unauthenticated so documentation can be viewed without a token. Do not enable this in production if you do not want your API contract to be public.

## JWT-Aware OpenAPI

When `server.jwt.enabled: true`, generated OpenAPI includes HTTP bearer auth (`bearerAuth`) so Swagger UI prompts for a bearer token.

No OAuth flow is configured. It is expected that users will obtain a valid JWT from their auth provider of choice and paste it into the Swagger UI auth dialog to try out authenticated endpoints.

## Dynamic Generation and Caching

OpenAPI YAML is generated lazily on first request per table and then cached in memory. The generated OpenAPI spec reflects the current configuration for the table, including key field names and index definitions. If you change the config and restart the API, the generated OpenAPI will reflect those changes. This allows you to utilize the API as it is defined including the JSON Schema validation rules for each object and the optional query parameters for each provided API endpoint.

## CLI OpenAPI Spec Generation

You can generate OpenAPI YAML without starting the API or connecting to PostgreSQL:

```bash
go run . swagger -config config.yaml -table users
```

By default YAML is written to stdout.

To write to a file:

```bash
go run . swagger -config config.yaml -table users -output users-openapi.yaml
```
