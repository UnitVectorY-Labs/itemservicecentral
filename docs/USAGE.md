# CLI Usage and Configuration

itemservicecentral provides five subcommands to access different functionality:
- `api` - run the API server
- `validate` - validate your config
- `migrate` - run database migrations
- `swagger` - generate OpenAPI YAML for a single table
- `version` - print the application version

Most runtime flags have a corresponding environment variable. When both are set, the CLI flag takes precedence.

## Common Parameters

Some parameters are shared by multiple commands, those are documented here for reference. See the individual command sections below for command-specific parameters.

The following parameters configure the PostgreSQL connection and are shared by the `api` and `migrate` commands which both connect to the database:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-db-host` | `DB_HOST` | `localhost` | Database host |
| `-db-port` | `DB_PORT` | `5432` | Database port |
| `-db-name` | `DB_NAME` | (required) | Database name |
| `-db-user` | `DB_USER` | (required) | Database username |
| `-db-password` | `DB_PASSWORD` | (required) | Database password |
| `-db-sslmode` | `DB_SSLMODE` | `disable` | SSL mode |

The config parameter is required for all commands except `version` as that provides the main configuration file for the service contract and server behavior.

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-config` | `CONFIG` | `config.yaml` | Path to YAML config file, default is `config.yaml` |

## Commands

### `api`

Starts the HTTP API server. Connects to PostgreSQL, validates configuration does not need any migrations before starting, and serves requests.

```bash
go run . api -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass
```

Flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-skip-config-validation` | `SKIP_CONFIG_VALIDATION` | `false` | Skip `_meta` minimal table-structure hash validation at startup (unsafe) |

### `validate`

Validates the YAML configuration file, compiles all JSON schemas, and prints the computed minimal table-structure hash without starting the server. Useful for CI pipelines.

```bash
go run . validate -config config.yaml
```

### `migrate`

Runs database migrations (creates tables and indexes) and updates the stored minimal table-structure hash in `_meta` without starting the server. Not all configuration changes require a migration, but this is a safe way to ensure the database is up to date with the config. The API will refuse to start by default if a migration is needed but has not been applied, so this is a necessary step when deploying config changes.

```bash
go run . migrate -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass
```

Flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-cleanup` | — | `false` | Delete tables and indexes not in config |
| `-dry-run` | — | `false` | Print changes without applying them |

### `version`

Prints the application version and exits.

```bash
go run . version
```

### `swagger`

Generates OpenAPI YAML for a single table without starting the API server or connecting to PostgreSQL.

```bash
go run . swagger -config config.yaml -table users
```

Flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-table` | — | (required) | Table name to generate OpenAPI for |
| `-output` | — | stdout | Write YAML to a file path instead of stdout |


## Docker

Build and run using Docker:

```bash
docker build -t itemservicecentral .
docker run -p 8080:8080 \
  -e CONFIG=/config.yaml \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  -e DB_NAME=appdb \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -v ./config.yaml:/config.yaml:ro \
  itemservicecentral
```
