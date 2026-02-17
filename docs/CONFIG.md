# Configuration Reference

itemservicecentral has two configuration layers:

1. Runtime options passed as command flags or environment variables defined in [Usage](./USAGE.md)
2. a YAML file that defines server behavior, tables, validation, and indexes.

## YAML Configuration File

The YAML file defines the service contract. This includes the server configuration (port, JWT settings, and Swagger settings), the tables to be exposed by the API, their JSON schema for validation, and any secondary indexes for those tables.

See [example-config.yaml](./example-config.yaml).

### Top-Level Layout

```yaml
server:
  port: 8080
  jwt:
    enabled: false
  swagger:
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
| `server.port` | No | `8080` | API listen port. Overridden by `-port` / `PORT` for `api`. |
| `server.jwt.enabled` | No | `false` | Enables JWT authentication when `true`. |
| `server.jwt.jwksUrl` | When JWT enabled | — | JWKS endpoint for RS256 public keys. |
| `server.jwt.issuer` | No | — | Expected `iss` claim value. |
| `server.jwt.audience` | No | — | Expected `aud` claim value. |
| `server.swagger.enabled` | No | `false` | Enables public per-table `/_swagger` and `/_openapi` endpoints. |

### `tables` Section

Each entry in `tables` defines one resource/table.

Tables can be defined with only a Primary Key or with a composite key using both a Primary Key and Range Key. The API endpoints and database schema are designed accordingly based on the presence of a Range Key.

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

- top-level schema must declare `type: object`
- every object schema level must set `additionalProperties: false` for every level of nesting
- unsupported extension keywords are rejected (for example `$ref`, `$defs`, `definitions`, `allOf`, `anyOf`, `oneOf`, and conditional schema keywords)
- all attribute names must match `^[A-Za-z][A-Za-z0-9_-]*$` to ensure they can be used as URL parameters without encoding

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

Secondary indexes provide an alternative method for querying items by a non-key field. The query capabilities provided by itemservicecentral's API are intentionally limited to keep the implementation simple and performant. Indexes are sparse and can be created on optional columns. The recommendation is to be intentional about the design of your data model and only provide the required query patterns via indexes. Creating composite keys with range keys is a common way to add flexibility with querying.

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
