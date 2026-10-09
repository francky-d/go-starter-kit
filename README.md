# Go Starter Kit

A starter kit for REST APIs in Go, built on Echo, GORM and PostgreSQL. It
ships a complete user authentication module, a test suite that runs
against a real database, and instructions and skills for AI coding agents.

The code is written to read like plain English:

```go
api := r.Group("/api/v1")

authentication := api.Group("/auth")

authentication.Middleware(d.CredentialsRateLimit).
    Post("/register", d.Auth.Register).
    Post("/login", d.Auth.Login)

api.Middleware(d.Authenticated).
    Get("/me", d.Users.Me)
```

```go
func (ac *AuthController) Login(c echo.Context) error {
    var input requests.Login
    if err := requests.Bind(c, &input); err != nil {
        return err
    }

    session, err := ac.auth.Login(c.Request().Context(), input.Email, input.Password)
    if err != nil {
        return err
    }

    return respond.OK(c, resources.NewSession(session))
}
```

## Features

- **HTTP:** Echo v4 behind a fluent router; one JSON envelope for every
  success and every error; request validation with field-level messages;
  request id, structured access logs, panic recovery, security headers,
  CORS, body size limit and request timeout on every request.
- **Authentication:** register, login, refresh, logout and current user.
  Short-lived JWT access tokens; opaque refresh tokens stored as hashes,
  rotated on every use, with reuse detection that revokes every session.
  bcrypt passwords, identical answers for unknown emails and wrong
  passwords, and per-IP rate limits.
- **Database:** PostgreSQL 17 through GORM, UUID v7 ids, versioned SQL
  migrations embedded in the binaries, and a migrate command.
- **Errors that disclose nothing:** clients get generic messages; causes go
  to the logs.
- **Configuration:** typed, validated at startup, from the environment or
  a `.env` file.
- **Tests:** unit, repository, endpoint and end-to-end tests against a real
  PostgreSQL database; coverage report.
- **Tooling:** Makefile, Docker Compose, a 28 MB distroless image, live
  reload, `staticcheck`, GitHub Actions CI.
- **Agent-ready:** `AGENTS.md`, `CLAUDE.md` and skills that teach an AI
  agent the conventions, how to add a resource, review the code, keep the
  docs current and commit.

## Quick start

Requirements: Go 1.26 or later, Docker with Compose v2, `make`.

```bash
git clone https://github.com/francky-d/go-starter-kit.git
cd go-starter-kit
cp .env.example .env
```

Set `JWT_SECRET` in `.env` to the output of `openssl rand -hex 32`, then:

```bash
make docker-up      # start PostgreSQL
make migrate-up     # create the schema
make run            # start the API on :8080
```

```bash
curl -s http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"correct horse battery staple"}'
```

Run `make` to list every command, and `make test` to run the test suite.

## Project layout

```text
cmd/
  api/                 the API entry point
  migrate/             the migration command
internal/
  app/                 dependency wiring and the HTTP server
  config/              typed configuration
  http/                controllers, middleware, requests, resources, respond, router
  routes/              every route, in one file
  services/            business rules
  repositories/        queries, one repository per model
  models/              GORM models
  auth/                password hashing, access and refresh tokens
  database/            connection pool and migration runner
  apperror/            errors carrying an HTTP status and a safe message
  logging/             slog logger
  testutil/            test helpers
database/
  migrations/          SQL migrations, embedded in the binaries
  init/                creates the test database in the local PostgreSQL container
docs/                  documentation
.claude/skills/        skills for AI agents
.github/workflows/     CI
```

## Documentation

| Document | Covers |
| --- | --- |
| [Getting started](docs/getting-started.md) | Setup, everyday commands, live reload. |
| [Architecture](docs/architecture.md) | Packages, startup and shutdown, request lifecycle, routes. |
| [API reference](docs/api-reference.md) | Envelopes, errors, status codes, every endpoint. |
| [Authentication](docs/authentication.md) | Passwords, tokens, sessions, rate limiting. |
| [Configuration](docs/configuration.md) | Every environment variable. |
| [Database](docs/database.md) | Local PostgreSQL, migrations, schema. |
| [Testing](docs/testing.md) | Running and writing tests, coverage, CI. |
| [Conventions](docs/conventions.md) | How the code is organised and written. |
| [Deployment](docs/deployment.md) | Docker image, migrations in production, checklist. |

## Working with AI agents

[AGENTS.md](AGENTS.md) holds the rules every agent follows: the stack, the
layout, the commands, the git workflow, and a demand for honest,
argued critique instead of agreement by default. [CLAUDE.md](CLAUDE.md)
imports it and tells Claude Code when to use each skill:

| Skill | Use |
| --- | --- |
| `conventions` | Before writing any Go code, migration or test. |
| `scaffold` | Adding a resource, from the migration to the routes, tests and docs. |
| `review` | A prioritised security and quality review in `review/code-review.md`. |
| `docs` | Before every commit: decides which docs the change affects. |
| `commit` | Atomic Conventional Commits, after the checks and the docs. |

Work is planned on the [project board](PLAN.md): one task per commit, each
task moving from `todo` to `in progress` to `done`.

## Contributing

- One task, one commit, with a [Conventional Commits](https://www.conventionalcommits.org)
  message.
- `make lint` and `make test` pass before every commit.
- Documentation changes ship in the same commit as the code they describe.

## License

No license has been chosen yet.
