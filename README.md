# itemservicecentral

itemservicecentral is a configuration-driven API service for storing and querying JSON documents with predictable key-based access patterns.

## Get Started

1. Create a config file (see `docs/example-config.yaml`).
2. Validate the config:

```bash
go run . validate -config config.yaml
```

3. Run migrations:

```bash
go run . migrate -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass
```

4. Start the API:

```bash
go run . api -config config.yaml -db-host localhost -db-port 5432 -db-name mydb -db-user myuser -db-password mypass
```

## Documentation

- [Overview](docs/README.md)
- [Usage](docs/USAGE.md)
- [Configuration](docs/CONFIG.md)
- [Database](docs/DATABASE.md)
- [API Reference](docs/API.md)
- [Swagger / OpenAPI](docs/SWAGGER.md)
- [Examples](docs/EXAMPLE.md)
- [Example Config](docs/example-config.yaml)
