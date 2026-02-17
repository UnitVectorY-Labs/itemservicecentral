# itemservicecentral

A configuration driven service that exposes JSON Schema validated CRUD and index style query REST endpoints backed by PostgreSQL JSONB with JWT based access control.

## Get Started

1. Create a config file (or use the example `docs/example-config.yaml`).

2. Validate the config:

```bash
go run . validate -config config.yaml
```

3. Run database migrations (required for the API to start):

```bash
go run . migrate -config config.yaml -db-host localhost -db-port 5432 -db-name appdb -db-user postgres -db-password postgres
```

4. Start the API:

```bash
go run . api -config config.yaml -db-host localhost -db-port 5432 -db-name appdb -db-user postgres -db-password postgres
```

`api` startup validates the minimal table-structure hash (tables, key fields, index key fields) stored in `_meta`.

## Documentation

- [Overview](docs/README.md)
- [Usage](docs/USAGE.md)
- [Configuration](docs/CONFIG.md)
- [Database](docs/DATABASE.md)
- [API Reference](docs/API.md)
- [Swagger / OpenAPI](docs/SWAGGER.md)
- [Examples](docs/EXAMPLE.md)
