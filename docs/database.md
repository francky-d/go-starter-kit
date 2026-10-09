# Database

The API stores its data in PostgreSQL 17 and reaches it through
[GORM](https://gorm.io) with the `pgx` driver.

## Local database

`docker-compose.yml` runs PostgreSQL for development and tests:

```bash
make docker-up      # start and wait until ready
make docker-down    # stop, keep the data
make docker-reset   # stop and delete the data
```

| Setting | Value |
| --- | --- |
| Host and port | `localhost:5432`, reachable from this machine only |
| User and password | `app` / `app` |
| Application database | `app` |
| Test database | `app_test` |

The test database is created by `database/init/01_create_test_database.sql`
the first time the container starts with an empty volume. If you created
the volume before that script existed, run `make docker-reset`.

These credentials are for local use only.

To use another host port, set `DB_FORWARD_PORT` (in `.env` or the shell)
and change the port in `DATABASE_URL` and `TEST_DATABASE_URL` to match.

## Test database

Tests connect to `app_test` on `localhost:5432` by default. Set
`TEST_DATABASE_URL` to use another database, for example in CI or with a
different port:

```bash
TEST_DATABASE_URL=postgres://app:app@localhost:5433/app_test?sslmode=disable make test
```

## Connection

At startup the API opens a connection pool and pings the database. If the
ping does not succeed within `DB_CONNECT_TIMEOUT`, the API exits with an
error instead of failing later on the first request.

The pool size and connection lifetimes are set by the `DB_*` variables in
[configuration.md](configuration.md#database).

## Query logging

Queries slower than `DB_SLOW_QUERY_THRESHOLD` and failed queries are logged
as warnings. Logged queries show placeholders (`$1`, `$2`) instead of bound
values, so emails, password hashes and tokens never reach the logs. A query
that finds no record is not logged.
