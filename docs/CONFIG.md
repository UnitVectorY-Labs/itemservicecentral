# Configuration Reference

itemservicecentral has two configuration layers:

1. runtime options passed as command flags or environment variables,
2. a YAML file that defines server behavior, tables, validation, and indexes.

## Runtime Options (Flags and Environment Variables)

Every flag has a corresponding environment variable. If both are set, the flag value is used.

### Common Runtime Options

These options are shared across commands.

| Flag | Environment Variable | Commands | Default | Description |
|------|----------------------|----------|---------|-------------|
| `-config` | `ISC_CONFIG` | `api`, `validate`, `migrate` | `config.yaml` | Path to YAML configuration file |
| `-db-host` | `ISC_DB_HOST` | `api`, `migrate` | `localhost` | PostgreSQL host |
| `-db-port` | `ISC_DB_PORT` | `api`, `migrate` | `5432` | PostgreSQL port |
| `-db-name` | `ISC_DB_NAME` | `api`, `migrate` | required | PostgreSQL database name |
| `-db-user` | `ISC_DB_USER` | `api`, `migrate` | required | PostgreSQL username |
| `-db-password` | `ISC_DB_PASSWORD` | `api`, `migrate` | required | PostgreSQL password |
| `-db-sslmode` | `ISC_DB_SSLMODE` | `api`, `migrate` | `disable` | PostgreSQL SSL mode |

### Command-Specific Runtime Options

`api`:

| Flag | Environment Variable | Default | Description |
|------|----------------------|---------|-------------|
| `-port` | `ISC_PORT` | `8080` | HTTP listen port (overrides `server.port` in YAML) |

`validate`:

- no command-specific options.

`migrate`:

| Flag | Environment Variable | Default | Description |
|------|----------------------|---------|-------------|
| `-cleanup` | — | `false` | Remove tables and indexes not present in config |
| `-dry-run` | — | `false` | Print planned migration changes only |

## YAML Configuration File

The YAML file defines the service contract. See [example-config.yaml](./example-config.yaml).

### Top-Level Layout

```yaml
server:
  port: 8080
  jwt:
    enabled: false

tables:
  - name: users
    primaryKey:
      field: userId
      pattern: "^[A-Za-z_][A-Za-z0-9._-]*$"
    schema:
      type: object
      additionalProperties: false
      properties:
        userId:
          type: string
          pattern: "^[A-Za-z_][A-Za-z0-9._-]*$"
      required:
        - userId
```

### `server` Section

| Field | Required | Default | Description |
|------|----------|---------|-------------|
| `server.port` | No | `8080` | API listen port. Overridden by `-port` / `ISC_PORT` for `api`. |
| `server.jwt.enabled` | No | `false` | Enables JWT authentication when `true`. |
| `server.jwt.jwksUrl` | When JWT enabled | — | JWKS endpoint for RS256 public keys. |
| `server.jwt.issuer` | No | — | Expected `iss` claim value. |
| `server.jwt.audience` | No | — | Expected `aud` claim value. |

### `tables` Section

Each entry in `tables` defines one resource/table.

#### Table Attributes

| Field | Required | Description |
|------|----------|-------------|
| `name` | Yes | Logical table name. Must match `^[a-z][a-z0-9_]*$`. |
| `primaryKey.field` | Yes | JSON field used as the table Primary Key. Must match `^[A-Za-z][A-Za-z0-9_-]*$`. |
| `primaryKey.pattern` | Yes | Regex applied to URL and payload Primary Key values. |
| `rangeKey.field` | No | JSON field used as Range Key when composite keys are needed. |
| `rangeKey.pattern` | No | Regex applied to URL and payload Range Key values. |
| `schema` | Yes | Restricted JSON Schema used for request validation. |
| `allowTableScan` | No | Enables `GET /v1/{table}/_items` when `true`. Default `false`. |
| `indexes` | No | List of secondary index definitions. |

#### Cross-Field Table Rules

- `primaryKey.field` and `rangeKey.field` must be different.
- If a Range Key is configured, both its `field` and `pattern` are required.
- The schema must define key fields with `type: string` and a non-empty `pattern`.

### `schema` (Restricted JSON Schema Subset)

Schemas are intentionally restricted to keep behavior deterministic.

Required rules:

- top-level schema must declare `type: object`,
- every object schema level must set `additionalProperties: false` (the `allowAdditionalProperties` policy is always false),
- unsupported extension keywords are rejected (for example `$ref`, `$defs`, `definitions`, `allOf`, `anyOf`, `oneOf`, and conditional schema keywords).

Supported keyword set is limited to common validation primitives such as:

- `type`, `properties`, `required`, `additionalProperties`,
- `items`, `pattern`, `enum`, `const`,
- common string/number/array/object bounds (`minLength`, `maximum`, `maxItems`, `minProperties`, etc.).

Example with nested object levels:

```yaml
schema:
  type: object
  additionalProperties: false
  properties:
    userId:
      type: string
      pattern: "^[A-Za-z_][A-Za-z0-9._-]*$"
    profile:
      type: object
      additionalProperties: false
      properties:
        displayName:
          type: string
  required:
    - userId
```

### `indexes` Section

Each index creates an alternate query path.

| Field | Required | Description |
|------|----------|-------------|
| `name` | Yes | Index name. Must match `^[a-z][a-z0-9_]*$`. Unique per table. |
| `primaryKey.field` | Yes | JSON field used as index Primary Key. |
| `rangeKey.field` | No | JSON field used as index Range Key. |
| `projection.type` | No | One of `ALL`, `KEYS_ONLY`, `INCLUDE`. |
| `projection.nonKeyAttributes` | For `INCLUDE` | Must be non-empty for `INCLUDE`; must be empty for other projection types. |
| `allowIndexScan` | No | Enables `GET /v1/{table}/_index/{index}/_items` when `true`. Default `false`. |

Index field constraints:

- index key fields must be different from base table key fields,
- index Range Key (if set) must be different from index Primary Key.

## Related

- [Database Model and Migrations](./DATABASE.md)
- [API Reference](./API.md)
- [CLI Usage](./USAGE.md)
