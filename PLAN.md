# Project Board

This board tracks the work on the starter kit. Each row is one task, and each
task ships as exactly one atomic commit.

**Status:** the initial build is complete; every task of phases 1 to 9 is
done. The [follow-ups](#follow-ups) found during the build wait for a
decision. The [documentation in French and guides](#documentation-in-french-and-guides)
is in progress.

## How to use this board

- **Statuses:** `todo`, `in progress`, `done`.
- **Start a task:** set its status to `in progress`. Only one task is in
  progress at a time.
- **Finish a task:** set its status to `done` in the same commit as the work
  itself. The commit subject matches the **Commit** column, apart from a
  trailing `[skip ci]` on documentation-only commits.
- **Add a task:** append a row with the next free number. Never renumber
  existing rows.
- **Adopting the kit:** keep this section, delete the tables below, and
  start your own backlog from row 1.

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
| 15 | User model, repository and database test helpers | `feat(users): add user model, repository and database test helpers` | done |

## Phase 6 - Authentication

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 16 | Password hashing | `feat(auth): add password hashing` | done |
| 17 | JWT access token issuer and parser | `feat(auth): add JWT access token issuer and parser` | done |
| 18 | Refresh token model, migration and repository | `feat(auth): add refresh token model, migration and repository` | done |
| 19 | Authentication service with refresh token rotation | `feat(auth): add authentication service with refresh token rotation` | done |
| 20 | Register, login, refresh and logout endpoints | `feat(auth): add register, login, refresh and logout endpoints` | done |
| 21 | Bearer authentication middleware and current user endpoint | `feat(auth): add bearer authentication middleware and current user endpoint` | done |

## Phase 7 - End-to-end tests

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 22 | End-to-end authentication flow and coverage target | `test: add end-to-end authentication flow and coverage target` | done |

## Phase 8 - Tooling

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 23 | Multi-stage Dockerfile and API service in Compose | `build: add multi-stage Dockerfile and api service in compose` | done |
| 24 | GitHub Actions workflow | `ci: add GitHub Actions workflow` | done |
| 25 | Live reload configuration | `chore: add air live reload configuration` | done |

## Phase 9 - Documentation

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 26 | Complete README and documentation index | `docs: complete README and documentation index` | done |
| 27 | Close the project board | `docs: close project board` | done |

## Added during the build

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 28 | Readability conventions: single-purpose functions, comments only when necessary | `refactor: apply readability conventions to existing code` | done |
| 29 | Remove the unused `App.IsProduction` helper | `refactor(config): remove unused IsProduction helper` | done |

## Follow-ups

Found during the build and left for a decision. Each one is picked up like
any other task.

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 30 | Upgrade `golang.org/x/net` to v0.60.0: four advisories reachable through Echo | `build: upgrade golang.org/x/net to v0.60.0` | todo |
| 31 | Run `govulncheck` in CI, at a pinned version (after row 30, or CI fails) | `ci: run govulncheck` | todo |
| 32 | Purge expired refresh tokens, with its caller: a migrate-style command or a documented scheduled job | `feat(auth): purge expired refresh tokens` | todo |
| 33 | Choose a license and add the `LICENSE` file | `chore: add license` | todo |
| 37 | Test that repositories turn a check constraint violation into `ErrInvalid`: no test asserts the sentinel today | `test(repositories): cover the ErrInvalid translation` | todo |
| 39 | Scaffold skill: replace `Save` for updates (it inserts the record again when a concurrent delete made the update match nothing) with `Select(...).Updates` and an affected-rows check | `docs(scaffold): update records without Save` | todo |
| 40 | Test client: add `Put`, `Patch`, `Delete` and `Meta`, which the scaffold skill's endpoint tests need | `test: add PUT, PATCH, DELETE and meta to the test client` | todo |

## Documentation in French and guides

The documentation (`docs/` and `README.md`) is written in French; an
English translation will follow.

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 34 | Write the documentation in French and update the language rule | `docs: translate the documentation to French` | done |
| 35 | Document what switching to another database involves | `docs(database): document switching to another database` | done |
| 36 | Use case guide: a to-do list with sharing and roles, with verified code | `docs: add to-do list use case guide` | done |
| 38 | Validation messages say "1 character", not "1 characters" (found while building the guide) | `fix(http): use the singular for a validation limit of one` | done |
| 41 | Remove the announcement of a future translation from the docs index: the docs describe the current state | `docs: drop the translation plan from the docs index` | done |
| 43 | Describe the code style in the docs without claiming that it reads like English | `docs: describe the code style without the plain English claim` | done |

## Continuous integration

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 42 | GitLab CI pipeline equivalent to the GitHub Actions workflow | `ci: add GitLab CI pipeline` | done |

## Development environment

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 44 | Development environment in containers: Docker and make as the only requirements, `make dev` runs PostgreSQL and the API with live reload, every Go target runs in a `go` container | `build: run the development environment in containers` | done |

## Fixes

| # | Task | Commit | Status |
| --- | --- | --- | --- |
| 45 | A query cut by `HTTP_REQUEST_TIMEOUT` answers `500`: the service's `apperror.Internal` hides the timeout's `503` (found while documenting the request lifecycle) | `fix(http): answer 503 when the request deadline cuts the work` | done |
