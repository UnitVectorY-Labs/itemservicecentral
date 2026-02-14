# itemservicecentral

A configuration-driven service that exposes JSON Schema validated CRUD and index-style query REST endpoints backed by PostgreSQL JSONB with JWT-based access control.

## Quick Start

```bash
# Validate configuration
go run . validate -config config.yaml

# Run database migrations
go run . migrate -config config.yaml -db-url "postgres://user:pass@localhost:5432/dbname?sslmode=disable"

# Start the API server
go run . api -config config.yaml -db-url "postgres://user:pass@localhost:5432/dbname?sslmode=disable"

# Print version
go run . version
```

## Configuration

Configuration via environment variables or CLI flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-config` | `ISC_CONFIG` | `config.yaml` | Path to YAML config file |
| `-port` | `ISC_PORT` | `8080` | Server port |
| `-db-url` | `ISC_DATABASE_URL` | (required) | PostgreSQL connection string |

See [docs/](docs/) for full documentation.
