
set dotenv-load := true

DB_USER := env("DB_USER")
DB_PASSWORD := env("DB_PASSWORD")
DB_NAME := env("DB_NAME")

# List all available commands
default:
  @just --list

# Build the Go application
build:
  go build ./...

# Run the Go tests
test:
  go test ./...

# Run the validation checks
validate:
  go run . validate

# Run the migrate command to apply database migrations
migrate:
  go run . migrate

# Run the API server locally for development (without auto-migration)
api:
  go run . api

# Run the API server locally (with auto-migration) for development
serve:
  go run . migrate
  go run . api

# Run a local Postgres container for development
postgres-container:
  container rm -f itemservicecentral-local-postgres || true
  container run --name itemservicecentral-local-postgres \
    -e POSTGRES_USER="{{DB_USER}}" \
    -e POSTGRES_PASSWORD="{{DB_PASSWORD}}" \
    -e POSTGRES_DB="{{DB_NAME}}" \
    -p 5432:5432 \
    -d postgres:18