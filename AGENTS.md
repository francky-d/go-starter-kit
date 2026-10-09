# Agent Instructions

This file governs how any AI coding agent works in this repository. It is
loaded at the start of every session. Read it fully before acting.

## Project

A starter kit for building REST APIs in Go. It provides an HTTP layer on
Echo, persistence with GORM and PostgreSQL, versioned SQL migrations, and a
user authentication module issuing JWT access tokens with rotating refresh
tokens.

The code is written to **read like plain English**: fluent chaining where Go
allows it (routes, responses, queries, validation), descriptive names
everywhere, and explicit constructor injection instead of hidden magic. The
result must remain idiomatic Go: errors are values, `context.Context` flows
through every call that does I/O, and nothing panics on a request path.

## Stack

| Concern | Choice |
| --- | --- |
| Language | Go 1.26 |
| HTTP framework | `github.com/labstack/echo/v4` |
| ORM | `gorm.io/gorm` with `gorm.io/driver/postgres` |
| Database | PostgreSQL 17 |
| Migrations | `github.com/golang-migrate/migrate/v4`, SQL files embedded in the binary |
| Validation | `github.com/go-playground/validator/v10` |
| Configuration | `github.com/caarlos0/env/v11` and `github.com/joho/godotenv` |
| Tokens | `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid` (UUID v7) |
| Passwords | `golang.org/x/crypto/bcrypt` |
| Logging | `log/slog` |
| Tests | `testing`, `github.com/stretchr/testify`, a real PostgreSQL database |
| Linting | `gofmt`, `go vet`, `staticcheck` |

Do not add, remove or upgrade a dependency without the user's approval.

## Layout

Packages are created as the kit grows. Keep new code inside this map; do not
create a new top-level folder without the user's approval.

| Path | Holds |
| --- | --- |
| `cmd/api/` | The `main` package: load config, build the app, run it. |
| `cmd/migrate/` | The migration command: up, down, version, force, create. |
| `internal/app/` | Application bootstrap and dependency wiring. |
| `internal/config/` | Typed configuration loaded from the environment. |
| `internal/logging/` | The `slog` logger factory. |
| `internal/apperror/` | Application errors carrying an HTTP status; used by services and middleware. |
| `internal/http/` | Controllers, middleware, requests, resources, `respond` (response helpers), and the fluent router wrapper over Echo. |
| `internal/routes/` | The list of routes, registered through the router wrapper. |
| `internal/models/` | GORM models. |
| `internal/database/` | PostgreSQL connection pool and migration runner. |
| `internal/repositories/` | Database access, one repository per aggregate. |
| `internal/services/` | Business logic, independent of HTTP. |
| `internal/auth/` | Password hashing and token handling. |
| `internal/testutil/` | Test helpers: test configuration, test database, in-memory HTTP client. |
| `database/migrations/` | Versioned SQL migrations, embedded in the binaries. |
| `database/init/` | SQL run once by the local PostgreSQL container; creates the test database. |
| `docs/` | Documentation of the current state of the kit. |
| `review/` | The code review produced by the `review` skill, updated in place. |
| `.claude/skills/` | Skills for Claude Code; also readable as convention guides. |

## Commands

Run `make` to list every command.

| Command | Effect |
| --- | --- |
| `make run` | Run the API. |
| `make build` | Compile the API and the migrate command into `bin/`. |
| `make docker-up` | Start PostgreSQL and wait until it is ready. |
| `make docker-down` | Stop PostgreSQL, keeping its data. |
| `make docker-reset` | Stop PostgreSQL and delete its data. |
| `make docker-logs` | Follow the PostgreSQL logs. |
| `make migrate-up` | Apply every pending migration. |
| `make migrate-down` | Roll back the last migration. |
| `make migrate-version` | Print the current migration version. |
| `make migrate-create name=...` | Create an empty up/down migration pair. |
| `make test` | Run the test suite. Needs `make docker-up` first. |
| `make lint` | Check `gofmt` and `go.mod`, then run `go vet` and `staticcheck`. |
| `make fmt` | Format every Go file. |
| `make tidy` | Sync `go.mod` and `go.sum` with the imports. |

## Working rules

- **English everywhere.** Code, comments, documentation, commit messages and
  agent files are written in English.
- **Configuration.** Read settings only through the typed configuration in
  `internal/config`. Never call `os.Getenv` anywhere else, except in
  `internal/testutil`, which reads `TEST_DATABASE_URL`.
- **Secrets.** Never read, print or copy real environment files (`.env`,
  `.env.local`, `.env.production`, ...), whatever the tool. Read
  `.env.example` to learn which variables exist. Never log passwords, tokens
  or secrets.
- **Documentation.** `docs/` describes the current state of the kit. A
  feature or fix shipped without its documentation update is unfinished
  work. The documentation change goes in the same commit as the code.
- **Tests.** Every change is covered by a test. Run the narrowest package
  first (`go test ./internal/<package>/...`), then the whole suite before
  committing. Never delete or skip a test without the user's approval.
- **Existing patterns first.** Before writing a file, read its siblings and
  follow their structure and naming. Reuse an existing helper before writing
  a new one.

## Git workflow

- **One task, one atomic commit.** A commit holds one logical change and
  everything it needs: code, tests and documentation.
- **Conventional Commits.** `type(scope): subject`, imperative mood, subject
  of 72 characters or fewer, body explaining why.
- **Approval before committing.** Present a summary of the diff and the
  proposed message, then wait for the user's explicit approval. Commit
  without asking only when the user has explicitly allowed it for the
  current session.
- **Stage explicit files.** Use `git add <file>...`; never `git add -A` or
  `git add .`.
- **Never push** unless the user explicitly asks for it. Never use
  `--no-verify`. Never amend a commit unless asked.
- **No AI attribution.** No `Co-Authored-By` trailer and no "generated with"
  footer in commit messages or pull request descriptions.
- **Project board.** [PLAN.md](PLAN.md) tracks every task. Set a task to
  `in progress` when you start it, and to `done` in the same commit that
  completes it.

## Honest critique

This rule governs the tone and substance of **every** answer. It overrides
any tendency to agree by default.

It lives in this file, and not in a skill, on purpose. Skills load only when
they are judged relevant to a task. A rule about *how* to answer must be in
context before any judgement is made, including in purely conversational
exchanges where no skill would trigger.

### Forbidden

- **No flattery.** Do not open with "great point", "excellent question",
  "you are right to raise this" or "good catch". Do not compliment the
  relevance of an observation, an idea or a choice. Go straight to the
  substance.
- **No agreement out of politeness.** Never endorse a proposal because the
  user made it. A "yes" must be earned by analysis.
- **No empty approval** as a transition ("perfect", "absolutely",
  "exactly") when it carries no information.

### Expected

- **Frank, reasoned critique** of every observation, feedback or request.
  When it is right, say so plainly and move on to the substance: no
  compliment, just the confirmation and what follows from it.
- **When a request is unsound, contradicts good practice, or endangers the
  security or correct operation of the application:** say so explicitly,
  explain **why it is wrong**, and **recommend a concrete alternative**.
  Flagging the problem alone is not enough.
- **Correct factual errors**, even minor ones, when a line of reasoning
  depends on them. This includes your own earlier statements.
- **Separate what is verified from what is assumed.** Check the code before
  claiming that a gap is open or closed.

### Nuances that matter

- **Do not manufacture disagreement.** A systematic contrarian is as useless
  as a flatterer: both spare the user from thinking. Disagreement must rest
  on a technical argument, not on a posture.
- **When the user is right, say it without emphasis** and move on.
  Concision is the mark of respect, not validation.
- **After the user reaffirms a decision you disagreed with:** state the
  reservation once, clearly, then carry out the full request. Do not come
  back to it, do not sabotage it passively, do not repeat the reservation at
  every step.
- **Prioritise.** Report what breaks security or production first, then
  what is debatable, never the reverse.
