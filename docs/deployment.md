# Deployment

This page describes the Docker image, how to run the whole stack in
containers, and what to check before going to production.

## Docker image

The `Dockerfile` builds in two stages:

1. `golang:1.26-alpine` compiles static binaries for the API and the
   migrate command.
2. `gcr.io/distroless/static-debian13:nonroot` runs them. The image has no
   shell and no package manager, and runs as an unprivileged user.

```bash
docker build -t go-starter-kit .
```

| Path in the image | Content |
| --- | --- |
| `/usr/local/bin/api` | The API, the entry point of the image. It listens on `HTTP_PORT` (`8080`). |
| `/usr/local/bin/migrate` | The migrate command, with the migrations embedded. |

The image holds no configuration: pass every variable from
[configuration.md](configuration.md) through the environment. `.env`
files are excluded from the build context and never copied into the image.

## The whole stack in containers

`make docker-full` builds the image and starts PostgreSQL and the API
together:

```bash
make docker-full
curl -s http://localhost:8080/health
```

The API container applies the migrations at startup (`DB_AUTO_MIGRATE`)
and reads `JWT_SECRET` from `.env` or the shell; without it, the API exits
with a configuration error. It listens on `localhost:8080`; set
`API_FORWARD_PORT` to use another port. `make docker-logs` follows the logs
of both containers, and `make docker-down` stops them.

The image has no health check, so `make docker-full` returns once the API
process has started, not once it serves requests: check `/health` as above.

For day-to-day development, run PostgreSQL alone with `make docker-up` and
the API on the host with `make run`.

## Migrations in production

Apply migrations before the new version serves traffic, in one of two
ways:

- **A one-off job** before the rollout, with the same image:

  ```bash
  docker run --rm -e DATABASE_URL="$DATABASE_URL" \
    --entrypoint /usr/local/bin/migrate go-starter-kit up
  ```

- **At startup**, with `DB_AUTO_MIGRATE=true`. A PostgreSQL advisory lock
  makes concurrent instances apply each migration once, but every instance
  waits for the lock while it starts.

Write migrations that the previous version of the API can run against:
during a rollout, old and new instances share the database.

## Before going to production

- Set `APP_ENV=production`. Every log record carries it.
- Generate a `JWT_SECRET` for this environment only (`openssl rand -hex 32`)
  and keep it in a secret manager, never in the image or the repository.
- Use `sslmode=require` or stricter in `DATABASE_URL`.
- Keep `LOG_FORMAT=json` for log collectors.
- Terminate TLS in front of the API. Enable `HTTP_TRUST_PROXY` only when a
  proxy you control sets `X-Forwarded-For`.
- Set `CORS_ALLOWED_ORIGINS` to the exact origins of your web clients.
- Use `/health` as the readiness probe, not as the liveness probe: see
  [api-reference.md](api-reference.md#get-health).
- With several instances, the authentication rate limits count per
  instance; add a shared limit in the load balancer or gateway if needed.
- Keep `BCRYPT_COST` at 12 or more.
