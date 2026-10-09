# Configuration

The application reads its configuration from environment variables when it
starts. The `internal/config` package is the only code that reads them; every
other package receives the values it needs through its constructor.

## How values are resolved

1. A variable set in the process environment wins.
2. Otherwise, a value from a `.env` file in the working directory is used.
   The file is optional.
3. Otherwise, the default listed below applies.

Start from the template:

```bash
cp .env.example .env
```

`.env` is ignored by git. Never commit real secrets.

## Validation

The configuration is validated once, at startup. An invalid value stops the
application with an error listing **every** problem found, so a deployment
can be fixed in a single pass:

```text
config: invalid configuration: APP_ENV must be one of [development test production], got "staging"
HTTP_PORT must be between 1 and 65535, got 0
```

A value of the wrong type (for example `HTTP_PORT=http`) is reported with the
name of the configuration field and the rejected value.

## Variables

Durations use Go syntax: `500ms`, `10s`, `5m`, `1h30m`.

### Application

| Variable | Default | Description |
| --- | --- | --- |
| `APP_NAME` | `go-starter-kit` | Name of the application. |
| `APP_ENV` | `development` | One of `development`, `test`, `production`. |

### HTTP server

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_PORT` | `8080` | Port the server listens on (1 to 65535). |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Maximum time to read the request headers. |
| `HTTP_READ_TIMEOUT` | `10s` | Maximum time to read the whole request, body included. |
| `HTTP_WRITE_TIMEOUT` | `30s` | Maximum time to write the response. |
| `HTTP_IDLE_TIMEOUT` | `60s` | Maximum time a keep-alive connection waits for the next request. |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Maximum time to finish in-flight requests on shutdown. |

All timeouts must be positive.

### Logging

| Variable | Default | Description |
| --- | --- | --- |
| `LOG_LEVEL` | `info` | One of `debug`, `info`, `warn`, `error` (case-insensitive). |
| `LOG_FORMAT` | `json` | `json` for machines, `text` for humans. `.env.example` uses `text`. |

## Adding a variable

1. Add a field to the right struct in `internal/config/config.go`, with an
   `env` tag and either an `envDefault` tag or the `required` option
   (`env:"NAME,required"`).
2. Add a check to `Validate` when some values are invalid.
3. Cover it in `internal/config/config_test.go`, including the list of known
   variables used to clear the environment.
4. Add it to `.env.example` and to this page.
