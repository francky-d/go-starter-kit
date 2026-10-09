# Getting started

## Prerequisites

- Go 1.26 or later.
- Docker with Compose v2, for the local PostgreSQL database.
- `make`.

## First steps

```bash
git clone https://github.com/francky-d/go-starter-kit.git
cd go-starter-kit
cp .env.example .env
make docker-up
make test
```

`make docker-up` starts PostgreSQL 17 and waits until it accepts
connections. See [database.md](database.md) for details.

Create the schema before running the API:

```bash
make migrate-up
```

## Run the API

```bash
make run
```

The server listens on `HTTP_PORT` (`8080` by default). Check that it
answers:

```bash
curl -s http://localhost:8080/health
# {"data":{"status":"ok","database":"ok"}}
```

The API refuses to start if the database is unreachable within
`DB_CONNECT_TIMEOUT`.

Logs are written to standard output. Stop the server with `Ctrl+C`: it
finishes the requests in flight before exiting.

## Commands

Run `make` to list every command.

| Command | Effect |
| --- | --- |
| `make run` | Run the API with `go run`. |
| `make build` | Compile the API and the migrate command into `bin/`. |
| `make docker-up` | Start PostgreSQL and wait until it is ready. |
| `make docker-down` | Stop PostgreSQL, keeping its data. |
| `make docker-reset` | Stop PostgreSQL and delete its data. |
| `make docker-logs` | Follow the PostgreSQL logs. |
| `make migrate-up` | Apply every pending migration. |
| `make migrate-down` | Roll back the last migration. |
| `make migrate-version` | Print the current migration version. |
| `make migrate-create name=...` | Create an empty up/down migration pair. |
| `make test` | Run the test suite. PostgreSQL must be running. |
| `make lint` | Check formatting and `go.mod`, then run `go vet` and `staticcheck`. |
| `make fmt` | Format every Go file with `gofmt`. |
| `make tidy` | Sync `go.mod` and `go.sum` with the imports. |

`staticcheck` is pinned in `go.mod` with the `tool` directive and runs
through `go tool staticcheck`: nothing to install.
