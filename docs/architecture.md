# Architecture

This page explains how the application starts, how a request travels
through the code, and where each responsibility lives.

## Packages

| Package | Responsibility |
| --- | --- |
| `cmd/api` | Entry point: loads the configuration, builds the logger and the application, runs it until a signal arrives. |
| `cmd/migrate` | Applies, rolls back and creates migrations. |
| `database/migrations` | The SQL migrations, embedded in the binaries. |
| `internal/config` | Reads and validates environment variables into typed structs. The only package that reads the environment. |
| `internal/logging` | Builds the `slog` logger from the logging configuration. |
| `internal/app` | Wires every dependency with explicit constructors and runs the HTTP server. |
| `internal/database` | Opens the PostgreSQL connection pool used through GORM, and runs the embedded migrations. |
| `internal/routes` | Declares every route in one place. |
| `internal/http/router` | Fluent wrapper over Echo routing: groups, middleware and routes declared as a chain. |
| `internal/http/controllers` | HTTP handlers. A controller returns a response or an error, never an error body. |
| `internal/http/respond` | Success envelopes and the central error handler. |
| `internal/apperror` | Errors that carry an HTTP status and a client-safe message. |
| `internal/testutil` | Test helpers: test configuration and test database URL. |

Dependencies point inwards: `cmd/api` depends on `internal/app`, which
depends on everything else. Nothing depends on `internal/app`.

## Startup and shutdown

```text
main
 ├─ config.Load()          read .env (optional) and the environment, validate
 ├─ logging.New()          JSON or text logger on stdout
 ├─ app.New(ctx, cfg, log)
 │   ├─ database.Connect   open the pool, fail within DB_CONNECT_TIMEOUT if unreachable
 │   ├─ migrations         applied here only when DB_AUTO_MIGRATE=true
 │   └─ routes             build Echo, install the error handler, register routes
 ├─ app.Run(ctx)           listen on HTTP_PORT and serve
 │   └─ SIGINT / SIGTERM   stop accepting connections,
 │                         wait up to HTTP_SHUTDOWN_TIMEOUT for in-flight requests
 └─ app.Close()            close the database pool
```

Any startup error (invalid configuration, unreachable database, port
already in use) is printed
on standard error and the process exits with status 1.

Every log record carries the `app` and `environment` attributes.

The server applies the read, write and idle timeouts from the
[configuration](configuration.md#http-server), so a slow or idle client
cannot hold a connection forever.

## Routes

Every route is declared in `internal/routes/routes.go` through the router
wrapper, so the routing table reads as a list:

```go
api := r.Group("/api/v1")

api.Group("/auth").Middleware(rateLimit).
    Post("/register", auth.Register).
    Post("/login", auth.Login)

api.Middleware(authenticated).Get("/me", users.Me)
```

- `Group(prefix, middleware...)` and `Middleware(middleware...)` return a
  new router and leave the receiver unchanged, so the order in which groups
  and middleware are declared never matters.
- Group middleware runs before route middleware, in declaration order.
- Middleware only runs on routes that exist: an unknown URL under a
  protected prefix answers `404`, not `401`.

## Request lifecycle

```text
HTTP request
 └─ Echo router ─ no route? ───────────────────────┐
     └─ controller                                 │
         ├─ success ─ respond.OK / Created / ...   │
         └─ return err ────────────────────────────┤
                                                   ▼
                                   respond.ErrorHandler
                                    ├─ *apperror.Error  → its status, message, field errors
                                    ├─ *echo.HTTPError  → its status, generic message
                                    └─ anything else    → 500, generic message
```

The error handler is the single place where error bodies are written. It
logs server errors with their cause and never sends internal details to
the client. See [API reference](api-reference.md#errors) for the format.
