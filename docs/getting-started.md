# Getting started

## Prerequisites

- Go 1.26 or later.
- `make`.

## First steps

```bash
git clone https://github.com/francky-d/go-starter-kit.git
cd go-starter-kit
cp .env.example .env
make test
```

## Commands

Run `make` to list every command.

| Command | Effect |
| --- | --- |
| `make test` | Run the test suite. |
| `make lint` | Check formatting and `go.mod`, then run `go vet` and `staticcheck`. |
| `make fmt` | Format every Go file with `gofmt`. |
| `make tidy` | Sync `go.mod` and `go.sum` with the imports. |

`staticcheck` is pinned in `go.mod` with the `tool` directive and runs
through `go tool staticcheck`: nothing to install.
