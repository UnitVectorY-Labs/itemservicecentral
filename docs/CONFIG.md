# Configuration File Format

The service is configured via a YAML file. See [example-config.yaml](example-config.yaml) for a complete example.

## Server Section

```yaml
server:
  port: 8080
  jwt:
    enabled: true
    jwksUrl: "https://auth.example.com/.well-known/jwks.json"
    issuer: "https://auth.example.com"
    audience: "my-api"
```

| Field | Default | Description |
|-------|---------|-------------|
| `server.port` | `8080` | HTTP listen port. Can be overridden with `-port` flag or `ISC_PORT`. |
| `server.jwt.enabled` | `false` | When `true`, all requests require a valid Bearer token. |
| `server.jwt.jwksUrl` | — | URL to fetch RS256 public keys (required when JWT is enabled). |
| `server.jwt.issuer` | — | Expected `iss` claim (optional). |
| `server.jwt.audience` | — | Expected `aud` claim (optional). |

## Tables Section

Each entry in `tables` defines a REST resource with its own database table.

### Table Name Rules

Table names must match `^[a-z][a-z0-9_]*$` — lowercase letters, digits, and underscores, starting with a letter.

### Table Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Table name. Must match `^[a-z][a-z0-9_]*$`. |
| `primaryKey.field` | Yes | JSON field name used as partition key. Must match `^[A-Za-z][A-Za-z0-9_-]*$`. |
| `primaryKey.pattern` | Yes | Regex pattern that PK values in URLs must match. |
| `rangeKey` | No | Range key configuration (same structure as `primaryKey`). Enables PK+RK composite keys. |
| `rangeKey.field` | When `rangeKey` set | JSON field name used as range key. Must differ from `primaryKey.field`. |
| `rangeKey.pattern` | When `rangeKey` set | Regex pattern that RK values in URLs must match. |
| `schema` | Yes | JSON Schema (Draft 2020-12 compatible) for validating request bodies. |
| `allowTableScan` | No | When `true`, enables the `GET /v1/{table}/_items` full table scan endpoint. Default: `false`. |
| `indexes` | No | List of Global Secondary Index definitions. |

### Schema

The `schema` field contains a JSON Schema definition written inline in YAML. All PUT and PATCH (after merge) request bodies are validated against this schema.

The schema must define the key fields (both `primaryKey.field` and `rangeKey.field` if present) as properties with `type: "string"` and a `pattern` constraint. This ensures that key values are validated consistently at both the URL and payload level.

```yaml
schema:
  type: object
  properties:
    userId:
      type: string
      pattern: "^[A-Za-z_][A-Za-z0-9._-]*$"
    name:
      type: string
  required:
    - userId
    - name
```

### Indexes

Each index defines an alternate query path over the table data using JSON fields from the JSONB payload.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Index name. Must match `^[a-z][a-z0-9_]*$`. Must be unique within the table. |
| `primaryKey.field` | Yes | JSON field from the data payload to use as the index partition key. Must differ from the base table PK and RK fields. |
| `rangeKey` | No | Optional range key for the index (same structure as `primaryKey`). Must differ from base PK, base RK, and index PK fields. |
| `projection` | No | Projection configuration for this index. If omitted, all fields are returned. |
| `projection.type` | When `projection` set | Must be `ALL`, `KEYS_ONLY`, or `INCLUDE`. |
| `projection.nonKeyAttributes` | When type is `INCLUDE` | List of non-key fields to include. Must be non-empty for `INCLUDE` and empty for `ALL`/`KEYS_ONLY`. |
| `allowIndexScan` | No | When `true`, enables the `GET /v1/{table}/_index/{name}/_items` full index scan endpoint. Default: `false`. |

Example with all index options:

```yaml
indexes:
  - name: by_status
    primaryKey:
      field: status
    rangeKey:
      field: product
    projection:
      type: INCLUDE
      nonKeyAttributes:
        - amount
    allowIndexScan: true
```

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

## Migration Behavior

The migration system uses a `_meta` table to track table configurations. On each migration run:

- Tables and indexes are created or updated to match the current configuration.
- Key immutability is enforced: the `primaryKey.field` and `rangeKey.field` of an existing table cannot be changed.
- The `--cleanup` flag deletes tables and indexes that are no longer in the configuration.
- The `--dry-run` flag prints planned changes without applying them.
