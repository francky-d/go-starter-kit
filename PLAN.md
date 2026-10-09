# Project Board

This board tracks the work on the starter kit. Each row is one task, and each
task ships as exactly one atomic commit.

## How to use this board

- **Statuses:** `todo`, `in progress`, `done`.
- **Start a task:** set its status to `in progress`. Only one task is in
  progress at a time.
- **Finish a task:** set its status to `done` in the same commit as the work
  itself. The commit subject matches the **Commit** column, apart from a
  trailing `[skip ci]` on documentation-only commits.
- **Add a task:** append a row with the next free number. Never renumber
  existing rows.
- **Adopting the kit:** clear the tables and start your own backlog.

## Phase 1 - Repository skeleton and agent files

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 1 | Repository skeleton and project board | `chore: add repository skeleton and project board` | done |
| 2 | Agent instructions and Claude Code settings | `docs: add agent instructions and Claude Code settings` | done |
| 3 | Commit and docs skills | `docs: add commit and docs skills` | done |
| 4 | Conventions skill and conventions overview | `docs: add conventions skill and conventions overview` | done |
| 5 | Scaffold and review skills | `docs: add scaffold and review skills` | done |

## Phase 2 - Module, configuration, logging and bootstrap

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 6 | Go module and typed configuration loading | `feat(config): initialize module and typed configuration loading` | done |
| 7 | Structured logger factory | `feat(logging): add slog logger factory` | done |
| 8 | Response envelope and application errors | `feat(http): add response envelope and application error handling` | done |
| 9 | Server bootstrap, graceful shutdown, health endpoint | `feat(app): add server bootstrap with graceful shutdown and health endpoint` | done |

## Phase 3 - Database

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 10 | PostgreSQL connection and Docker Compose service | `feat(database): add Postgres connection and docker compose service` | done |
| 11 | Embedded migration runner and users table | `feat(database): add embedded migration runner and users table` | done |

## Phase 4 - HTTP helpers

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 12 | Request binding with validation errors | `feat(http): add request binding with validation errors` | done |
| 13 | Fluent router wrapper | `feat(http): add fluent router wrapper` | done |
| 14 | Baseline middleware stack | `feat(http): add baseline middleware stack` | done |

## Phase 5 - Users

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 15 | User model, repository and database test helpers | `feat(users): add user model, repository and database test helpers` | todo |

## Phase 6 - Authentication

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 16 | Password hashing | `feat(auth): add password hashing` | todo |
| 17 | JWT access token issuer and parser | `feat(auth): add JWT access token issuer and parser` | todo |
| 18 | Refresh token model, migration and repository | `feat(auth): add refresh token model, migration and repository` | todo |
| 19 | Authentication service with refresh token rotation | `feat(auth): add authentication service with refresh token rotation` | todo |
| 20 | Register, login, refresh and logout endpoints | `feat(auth): add register, login, refresh and logout endpoints` | todo |
| 21 | Bearer authentication middleware and current user endpoint | `feat(auth): add bearer authentication middleware and current user endpoint` | todo |

## Phase 7 - End-to-end tests

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 22 | End-to-end authentication flow and coverage target | `test: add end-to-end authentication flow and coverage target` | todo |

## Phase 8 - Tooling

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 23 | Multi-stage Dockerfile and API service in Compose | `build: add multi-stage Dockerfile and api service in compose` | todo |
| 24 | GitHub Actions workflow | `ci: add GitHub Actions workflow` | todo |
| 25 | Live reload configuration | `chore: add air live reload configuration` | todo |

## Phase 9 - Documentation

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 26 | Complete README and documentation index | `docs: complete README and documentation index` | todo |
| 27 | Close the project board | `docs: close project board` | todo |

## Added during the build

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 28 | Readability conventions: single-purpose functions, comments only when necessary | `refactor: apply readability conventions to existing code` | done |
