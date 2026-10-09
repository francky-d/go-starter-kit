# Conventions

The code of this kit is written to read like plain English while staying
idiomatic Go. This page summarises the rules. The complete, authoritative
version is the [conventions skill](../.claude/skills/conventions/SKILL.md),
which AI agents load before writing code and which humans can read as a
style guide.

## Reading order of a request

```text
router -> middleware -> controller -> service -> repository -> database
```

| Layer | Responsibility |
| --- | --- |
| Controller | Binds the input, calls one service method, returns a response. |
| Request | Declares input fields and their validation rules. |
| Resource | Shapes a model into the JSON a client sees. |
| Service | Holds business rules and transactions; returns application errors. |
| Repository | Runs queries; returns models or sentinel errors. |
| Model | Maps a table; small helpers on a single record. |

## Key rules

- **Names say what things do.** `users.FindByEmail(ctx, email)`,
  `respond.Created(c, resources.NewUser(user))`.
- **Explicit dependencies.** Every type is built by a `NewXxx` constructor
  that receives its dependencies. There is no global container.
- **Controllers stay thin.** They never build error JSON and never query the
  database.
- **Errors disclose nothing.** Clients get generic messages for 401, 403,
  404 and 500. Internal causes are logged, never returned. The only
  accepted disclosure is a 409 when registering a taken email.
- **Models are never serialised directly.** A resource decides which fields
  leave the server.
- **Configuration is typed.** Only `internal/config` reads environment
  variables.
- **Context everywhere.** Every function doing I/O takes `ctx` first.
- **Migrations are append-only.** Never edit a migration that has been
  applied outside your machine; write a new one.
- **Every change is tested**, against a real PostgreSQL database when the
  database is involved.
