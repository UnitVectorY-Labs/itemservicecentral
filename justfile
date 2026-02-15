
set dotenv-load := true

default:
  @just --list

build:
  go build ./...

test:
  go test ./...

serve:
  go run . migrate
  go run . api
