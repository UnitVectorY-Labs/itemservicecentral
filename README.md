# itemservicecentral

A configuration-driven service that exposes JSON Schema validated CRUD and index-style query REST endpoints backed by PostgreSQL JSONB with JWT-based access control.

## Quick Start

```bash
# Validate configuration
go run . validate -config config.yaml

# Run database migrations
go run . migrate -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass

# Start the API server
go run . api -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass

# Print version
go run . version
```

## Configuration

Configuration via environment variables or CLI flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-config` | `ISC_CONFIG` | `config.yaml` | Path to YAML config file |
| `-port` | `ISC_PORT` | `8080` | Server port |
| `-db-host` | `ISC_DB_HOST` | `localhost` | Database host |
| `-db-port` | `ISC_DB_PORT` | `5432` | Database port |
| `-db-name` | `ISC_DB_NAME` | (required) | Database name |
| `-db-user` | `ISC_DB_USER` | (required) | Database username |
| `-db-password` | `ISC_DB_PASSWORD` | (required) | Database password |
| `-db-sslmode` | `ISC_DB_SSLMODE` | `disable` | SSL mode |

## Documentation

- [Usage](docs/USAGE.md) — CLI commands, flags, and environment variables
- [API Reference](docs/API.md) — REST API endpoints and conventions
- [Configuration](docs/CONFIG.md) — YAML configuration file format
- [Examples](docs/EXAMPLE.md) — Quick reference with example requests and responses
- [Example Config](docs/example-config.yaml) — Complete example configuration file
