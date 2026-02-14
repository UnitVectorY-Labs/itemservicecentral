# CLI Usage and Configuration

itemservicecentral provides four subcommands: `api`, `validate`, `migrate`, and `version`.

Every flag has a corresponding environment variable prefixed with `ISC_`. When both are set, the CLI flag takes precedence.

## Commands

### `api`

Starts the HTTP API server. Connects to PostgreSQL, runs migrations, and begins serving requests.

```bash
go run . api -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass
```

Flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-config` | `ISC_CONFIG` | `config.yaml` | Path to YAML config file |
| `-port` | `ISC_PORT` | `8080` | Server port (overrides config file) |
| `-db-host` | `ISC_DB_HOST` | `localhost` | Database host |
| `-db-port` | `ISC_DB_PORT` | `5432` | Database port |
| `-db-name` | `ISC_DB_NAME` | (required) | Database name |
| `-db-user` | `ISC_DB_USER` | (required) | Database username |
| `-db-password` | `ISC_DB_PASSWORD` | (required) | Database password |
| `-db-sslmode` | `ISC_DB_SSLMODE` | `disable` | SSL mode |

### `validate`

Validates the YAML configuration file and compiles all JSON schemas without starting the server. Useful for CI pipelines.

```bash
go run . validate -config config.yaml
```

Flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-config` | `ISC_CONFIG` | `config.yaml` | Path to YAML config file |

### `migrate`

Runs database migrations (creates tables and indexes) without starting the server.

```bash
go run . migrate -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass
```

Flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-config` | `ISC_CONFIG` | `config.yaml` | Path to YAML config file |
| `-db-host` | `ISC_DB_HOST` | `localhost` | Database host |
| `-db-port` | `ISC_DB_PORT` | `5432` | Database port |
| `-db-name` | `ISC_DB_NAME` | (required) | Database name |
| `-db-user` | `ISC_DB_USER` | (required) | Database username |
| `-db-password` | `ISC_DB_PASSWORD` | (required) | Database password |
| `-db-sslmode` | `ISC_DB_SSLMODE` | `disable` | SSL mode |
| `-cleanup` | — | `false` | Delete tables and indexes not in config |
| `-dry-run` | — | `false` | Print changes without applying them |

### `version`

Prints the application version and exits.

```bash
go run . version
```

## Database Connection Parameters

The following parameters configure the PostgreSQL connection and are shared by the `api` and `migrate` commands:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-db-host` | `ISC_DB_HOST` | `localhost` | Database host |
| `-db-port` | `ISC_DB_PORT` | `5432` | Database port |
| `-db-name` | `ISC_DB_NAME` | (required) | Database name |
| `-db-user` | `ISC_DB_USER` | (required) | Database username |
| `-db-password` | `ISC_DB_PASSWORD` | (required) | Database password |
| `-db-sslmode` | `ISC_DB_SSLMODE` | `disable` | SSL mode |

## Docker

Build and run using Docker:

```bash
docker build -t itemservicecentral .
docker run -p 8080:8080 \
  -e ISC_CONFIG=/config.yaml \
  -e ISC_DB_HOST=host \
  -e ISC_DB_PORT=5432 \
  -e ISC_DB_NAME=dbname \
  -e ISC_DB_USER=user \
  -e ISC_DB_PASSWORD=pass \
  -e ISC_DB_SSLMODE=disable \
  -v ./config.yaml:/config.yaml:ro \
  itemservicecentral
```

The Docker image uses a distroless base for minimal attack surface.
