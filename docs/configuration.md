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
| `HTTP_REQUEST_TIMEOUT` | `15s` | Deadline of each request's context; a request whose work is cancelled by the deadline answers `503`. Must be below `HTTP_WRITE_TIMEOUT`. |
| `HTTP_BODY_LIMIT` | `1M` | Largest accepted request body, such as `512K` or `2M`. Larger bodies answer `413`. |
| `HTTP_TRUST_PROXY` | `false` | Read the client IP from `X-Forwarded-For`. See below. |

All timeouts must be positive.

#### Behind a reverse proxy

By default the client IP is the address of the TCP peer. Behind a load
balancer or reverse proxy, that is the proxy's address: set
`HTTP_TRUST_PROXY=true` so the IP is read from `X-Forwarded-For`. The header
is then honoured only when the peer is on a loopback or private network.
Never enable it when clients reach the API directly: they could choose the
IP that the logs and the rate limits see.

The API sends no `Strict-Transport-Security` header: it expects TLS to be
terminated by the proxy, which should send it. If the API terminates TLS
itself, add an `HSTSMaxAge` to the secure headers middleware.

### CORS

| Variable | Default | Description |
| --- | --- | --- |
| `CORS_ALLOWED_ORIGINS` | *(empty)* | Comma-separated browser origins allowed to call the API, such as `https://app.example.com, http://localhost:3000`. Each entry is `scheme://host[:port]` or `*`. |

With no origin configured, the API sends no CORS headers and browsers on
other origins cannot call it. Credentials (cookies) are never allowed:
clients authenticate with a bearer token.

### Database

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | **required** | PostgreSQL connection URL, for example `postgres://app:app@localhost:5432/app?sslmode=disable`. |
| `DB_CONNECT_TIMEOUT` | `5s` | Maximum time to reach the database at startup. |
| `DB_MAX_OPEN_CONNECTIONS` | `25` | Maximum number of open connections (at least 1). |
| `DB_MAX_IDLE_CONNECTIONS` | `25` | Maximum number of idle connections kept in the pool (0 to `DB_MAX_OPEN_CONNECTIONS`). |
| `DB_CONNECTION_MAX_LIFETIME` | `30m` | A connection is replaced after this time. |
| `DB_CONNECTION_MAX_IDLE_TIME` | `5m` | An idle connection is closed after this time. |
| `DB_SLOW_QUERY_THRESHOLD` | `200ms` | Queries slower than this are logged as warnings. |
| `DB_AUTO_MIGRATE` | `false` | Apply pending migrations when the API starts. See [database.md](database.md#migrations). |

Use `sslmode=require` (or stricter) in production.

### Authentication

| Variable | Default | Description |
| --- | --- | --- |
| `BCRYPT_COST` | `12` | bcrypt work factor for password hashes, from 4 to 16. Each step doubles the time to hash and to verify a password; keep at least 12 in production. |
| `JWT_SECRET` | none, required | Key that signs the access tokens, at least 32 bytes and random. Generate it with `openssl rand -hex 32` and keep all 64 hex characters; use a different one per environment and never commit it. Changing it invalidates every access token already issued. |
| `JWT_ISSUER` | `go-starter-kit` | Written into every access token; a token from another issuer is refused. |
| `ACCESS_TOKEN_LIFETIME` | `15m` | How long an access token stays valid. Keep it short: an access token cannot be revoked before it expires. |
| `AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE` | `10` | Requests per minute each client IP may send to register and login together. See [authentication.md](authentication.md#rate-limiting). |
| `AUTH_REFRESH_RATE_LIMIT_PER_MINUTE` | `60` | Requests per minute each client IP may send to refresh and logout together. |
| `REFRESH_TOKEN_LIFETIME` | `720h` | How long a refresh token stays valid (30 days by default). Must be longer than `ACCESS_TOKEN_LIFETIME`. Go durations have no day unit: write days in hours. |

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
