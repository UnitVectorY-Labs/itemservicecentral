This is a Go application that provides a configuration driven API that is delivered via a Docker container.

This application minimizes the user of dependencies preferring the standard Go library where possible.  The main.go file is the entry point and the rest of the code is organized under the internal/ directory. Any external files needed by the application such as a database schema or migration files are to be included in the compiled binary using the embed package.

While the bulk of the configuration lives in a configuration file and is referenced at runtime, the actual configuration passed into this application can be accomplished using well documented environment variables that each have a corresponding command line flag.  This allows for the application to be configured in a variety of ways and makes it easy to use in a containerized environment.

## JSON Scheama Restrictions

This application is intended to provide a way to quickly create a new API that is driven by a configuration file. The shape of the items themselves are determined by a JSON Schema file.  That schema must be a restrictive subset of what JSON Schema supports including:
- Limiting what attribute names can be to alphanumeric strings that support underscores and dashes (but must start with a letter).
- Limiting the attribute values for fields specified as primary key or range key for the main table or index to be limited to alphanumeric strings with . - and _ characters.
- The use of $ref and other dynamic features of JSON Schema are not supported.  The schema must be fully defined and self contained.
- The allowAdditionalProperties field must be set to false for all objects in the schema.

## Documentation

The documentation for this application lives in the docs/ directory and the root README.md file is minimal.

The different documentation files in the docs/ directory include:
- README.md: A high level overview of the application and how to use it.
- API.md: A detailed description of the API endpoints, request/response schemas, and query parameters.
- CONFIG.md: A  description of the YAML configuration file used to drive the behavior of the application.
- USAGE.md: A description of the different environment variables and command line flags that can be used to configure the application.
- DATABASE.md: A description of the database schema and how it is dynamically generated and updated.
- EXAMPLE.md: An example of how to use the application in a real world scenario.

The application generates dynamic swagger documentation for each {table} that is supported. Any change to the API such as adding a new endpoint or changing the request/response schema or query parameters must be reflected in this dynamically generated swagger documentation.

## Command Overview

This command line application supports the following commands:

- `api`: This command starts the API server and serves the endpoints defined in the configuration file.
- `validate`: This command validates the configuration file and checks for any errors or inconsistencies.
- `migrate`: This command runs any pending database migrations, unlike other applications the schema is generated dynamically based on the configuration file.
- `version`: This command prints the current version of the application.

## Testing

When testing, use a docker container running the `postgres:18` image.
