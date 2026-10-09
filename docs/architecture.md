# Architecture

This page explains how the application starts, how a request travels
through the code, and where each responsibility lives.

## Packages

| Package | Responsibility |
| --- | --- |
| `cmd/api` | Entry point: loads the configuration, builds the logger and the application, runs it until a signal arrives. |
| `internal/config` | Reads and validates environment variables into typed structs. The only package that reads the environment. |
| `internal/logging` | Builds the `slog` logger from the logging configuration. |
| `internal/app` | Wires every dependency with explicit constructors and runs the HTTP server. |
| `internal/routes` | Declares every route in one place. |
| `internal/http/controllers` | HTTP handlers. A controller returns a response or an error, never an error body. |
| `internal/http/respond` | Success envelopes and the central error handler. |
| `internal/apperror` | Errors that carry an HTTP status and a client-safe message. |

Dependencies point inwards: `cmd/api` depends on `internal/app`, which
depends on everything else. Nothing depends on `internal/app`.

## Startup and shutdown

```text
main
 ├─ config.Load()          read .env (optional) and the environment, validate
 ├─ logging.New()          JSON or text logger on stdout
 ├─ app.New(cfg, log)      build Echo, install the error handler, register routes
 └─ app.Run(ctx)           listen on HTTP_PORT and serve
      └─ SIGINT / SIGTERM  stop accepting connections,
                           wait up to HTTP_SHUTDOWN_TIMEOUT for in-flight requests
```

Any startup error (invalid configuration, port already in use) is printed
on standard error and the process exits with status 1.

Every log record carries the `app` and `environment` attributes.

The server applies the read, write and idle timeouts from the
[configuration](configuration.md#http-server), so a slow or idle client
cannot hold a connection forever.

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
