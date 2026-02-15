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

Migrations use the `_meta` table to track table and index metadata.

On each migration run:

- missing tables and indexes are created,
- existing objects are reconciled with current config,
- key field immutability is enforced (existing table key field names cannot be changed).

Additional migration flags:

- `-cleanup` removes tables/indexes no longer present in config.
- `-dry-run` prints planned changes without applying them.
