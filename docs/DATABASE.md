# Database Model and Migrations

This document covers how configuration maps to PostgreSQL tables and indexes.

## Table Storage Model

Each configured table maps to one PostgreSQL table with these columns:

| Column | Type | Description |
|--------|------|-------------|
| `pk` | `TEXT NOT NULL` | Primary Key value |
| `rk` | `TEXT` | Range Key value (`NOT NULL` when range key is configured) |
| `data` | `JSONB NOT NULL` | Payload data with configured key fields removed |
| `created_at` | `TIMESTAMPTZ` | Row creation time |
| `updated_at` | `TIMESTAMPTZ` | Last update time |

Primary key layout:

- Tables with only a Primary Key use `PRIMARY KEY (pk)`.
- Tables with both Primary Key and Range Key use `PRIMARY KEY (pk, rk)`.

## Key Field Handling

On write (`PUT`/`PATCH`):

- configured key fields are validated,
- key values are stored in `pk` and `rk`,
- key fields are stripped from the JSON payload before writing `data`.

On read (`GET`/query/list):

- `pk` and `rk` values are injected back into response JSON using configured field names.

If key fields are included in request bodies, they must match key values in the URL path.

## Index Storage Model

Configured indexes are created as PostgreSQL indexes on JSONB expressions.

- Indexes are sparse: only rows that contain the index key field(s) in `data` are indexed.
- Optional projection settings control which attributes are returned when reading through index endpoints.

## Migration Behavior

Migrations use the `_meta` table to track table/index metadata and the active minimal table-structure hash.

On each migration run:

- missing tables and indexes are created,
- existing objects are reconciled with current config,
- key field immutability is enforced (existing table key field names cannot be changed),
- the hash of the minimal table-structure configuration is updated in `_meta`.

## API Startup Validation

On startup, the `api` command validates that the current minimal table-structure hash matches the hash stored in `_meta`.

The minimal table-structure hash includes only:

- table names,
- each table primary/range key field names,
- index names,
- each index primary/range key field names.

It does not include non-structural settings such as JSON Schema definitions, scan flags, key patterns, JWT, or Swagger settings.

- if hashes match, startup continues,
- if hashes differ (or hash metadata is missing), startup fails and instructs you to run `migrate`,
- `-skip-config-validation` (or `SKIP_CONFIG_VALIDATION=true`) bypasses this check.

The `api` command does not create, alter, or drop tables/indexes.

Additional migration flags:

- `-cleanup` removes tables/indexes no longer present in config.
- `-dry-run` prints planned changes without applying them.
