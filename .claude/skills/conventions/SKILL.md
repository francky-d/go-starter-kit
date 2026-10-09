---
name: conventions
description: Coding conventions of the starter kit. Use this skill before writing or changing any Go code, SQL migration or test in this repository, and whenever the user asks how code should be structured, named, validated, returned, logged or tested here. It covers the layer responsibilities (controllers, requests, resources, services, repositories, models), the fluent helpers that make the code read like English, error handling and non-disclosure of error details, configuration access, context propagation, migrations and tests.
---

# Conventions

The code of this kit reads like plain English. A reader should understand
what a function does from its call site alone:

```go
authentication.Middleware(d.CredentialsRateLimit).
    Post("/register", d.Auth.Register).
    Post("/login", d.Auth.Login)

user, err := users.FindByEmail(ctx, email)

return respond.Created(c, resources.NewSession(session))
```

Readability never wins over idiomatic Go: errors are values, `ctx` flows
through every call that does I/O, and nothing panics on a request path
(the one exception is `middleware.CurrentUserID` on a route registered
without `Authenticated`: a wiring bug, not a client error).

Some packages named below are added as the kit grows. When one does not
exist yet, create it following these rules. When a sibling file already
exists, follow it first: consistency beats preference.

## 1. Layers and responsibilities

A request flows through the layers in one direction:

```text
router -> middleware -> controller -> service -> repository -> database
                           |            |
                       requests      models
                       resources     apperror
```

| Layer | Package | Does | Never does |
| --- | --- | --- | --- |
| Controller | `internal/http/controllers` | Bind the request, call one service method, return a response | Business rules, queries, building error JSON |
| Request | `internal/http/requests` | Declare input fields with `json` and `validate` tags | Logic beyond validation |
| Resource | `internal/http/resources` | Shape a model into the JSON a client sees | Database access |
| Service | `internal/services` | Business rules, transactions, mapping to `apperror` | Read `echo.Context`, write HTTP responses, build queries |
| Repository | `internal/repositories` | Queries through GORM, return models or sentinel errors | Business rules, HTTP concerns |
| Model | `internal/models` | Table mapping and small helpers on one record | Queries on other tables |

A service may open a transaction with `db.Transaction(...)` and hand `tx`
to repositories through their `WithTx(tx)` method. It never builds a query
itself.

## 2. Design: one function, one job

The kit applies SOLID the Go way.

- **A function does exactly what its name says, and nothing else.** When
  a function needs "and" to describe it, split it.
- **Short functions.** Aim for 20 to 25 lines. Extract each step into a
  well-named private function so the top-level function reads like a
  sentence:

  ```go
  func (a *App) Serve(ctx context.Context, listener net.Listener) error {
      server := a.newServer()
      serving := startServing(server, listener)

      select {
      case err := <-serving:
          return fmt.Errorf("app: serve: %w", err)
      case <-ctx.Done():
          return a.shutDown(ctx, server, serving)
      }
  }
  ```

- **One reason to change per type.** A controller handles HTTP, a service
  holds rules, a repository holds queries. A type that does two of these
  is split.
- **Small interfaces, declared by the consumer.** A component that only
  pings the database depends on `interface{ PingContext(context.Context) error }`,
  not on `*sql.DB`. Declare the interface next to the code that uses it.
- **No interface by reflex.** Do not put an interface in front of every
  struct. Add one when a test or a second implementation needs it.
- **Dependencies are injected** through constructors (see Naming).

## 3. Naming

- **Files:** `snake_case.go`, named after the main type:
  `user_repository.go`, `auth_controller.go`.
- **Types and constructors:** `UserRepository` with `NewUserRepository(...)`.
  Constructors take every dependency explicitly. There is no global
  container and no reflection-based injection.
- **Controller methods** are named after the action: `List`, `Show`,
  `Create`, `Update`, `Delete`, or a verb for non-CRUD actions (`Login`,
  `Refresh`).
- **Booleans** read as questions: `IsActive`, `HasExpired`, `CanRefresh`.
- **Finders** say what they match: `FindByID`, `FindByEmail`. A finder that
  returns nothing yields `repositories.ErrNotFound`, never `nil, nil`.
- **No stutter:** `auth.TokenIssuer`, not `auth.AuthTokenIssuer`.

## 4. Requests and validation

- One struct per endpoint input, in `internal/http/requests`, with `json`
  and `validate` tags (`github.com/go-playground/validator/v10`).
- Controllers call `requests.Bind(c, &input)`. It binds, validates, and
  returns an `apperror` the error handler turns into 400 (malformed body),
  415 (not JSON) or 422 (invalid fields).
- Validation messages are keyed by the JSON field name.
- Limit every string: `max=` on free text, `max=72` on a new password.
- bcrypt rejects passwords longer than 72 **bytes**, while `max=72` counts
  characters: a 72-character password with accents passes validation and
  still fails hashing. The password hasher therefore checks the byte length
  and the service maps that failure to a 422 on the `password` field, never
  to a 500.
- Credentials being checked (a login) get no format rule, only `required`
  and a generous `max` that bounds the input: a wrong value is a wrong
  credential, and must get the same answer as any other.

## 5. Responses

- Success responses go through `respond`: `respond.OK`, `respond.Created`,
  `respond.NoContent`, `respond.Paginated`. Never call `c.JSON` in a
  controller.
- The success envelope is `{"data": ..., "meta": {...}}`; `meta` is omitted
  when empty.
- Never serialise a model directly. Map it through a resource
  (`resources.NewUser(user)`) so a new column never leaks by accident.

## 6. Errors

- Services and middleware return `*apperror.Error` values:
  `apperror.BadRequest(msg)`, `apperror.Unauthorized()`,
  `apperror.Forbidden()`, `apperror.NotFound()`, `apperror.Conflict(msg)`,
  `apperror.Validation(fields)`, `apperror.Internal(err)`.
- Controllers `return err` unchanged. The central error handler writes the
  error envelope `{"message": "...", "errors": {...}}` and logs the cause.
- Wrap internal causes with `.Wrap(err)` or `apperror.Internal(err)`: the
  cause is logged, never sent to the client.
- **Disclose nothing.** An error message must not teach anything to someone
  who has no access to the resource:
  - 403 and 404 use the default generic messages;
  - "does not exist" and "exists but is not yours" both answer 404;
  - login and refresh failures answer the same 401 whether the account
    exists or not;
  - a 500 never carries an exception message, a file path or a stack trace.
- **Registration is the one accepted disclosure.** A taken email answers
  409 with a generic message, because hiding it would require an email
  verification flow the kit does not have. Rate limiting on the
  authentication routes mitigates enumeration.
- Repositories translate GORM errors into sentinels
  (`repositories.ErrNotFound`, `repositories.ErrDuplicate`) so services
  never depend on GORM error values.

## 7. Configuration

- Configuration is read once, at startup, into the typed struct of
  `internal/config`. Pass the part a component needs to its constructor.
- Never call `os.Getenv` outside `internal/config`. The one exception is
  `internal/testutil`, which reads `TEST_DATABASE_URL` for tests.
- A new variable gets a default when one is safe, the `required` option
  (`env:"NAME,required"`) when not,
  a line in `.env.example`, and a row in `docs/configuration.md`.

## 8. Context and logging

- Every function that does I/O takes `ctx context.Context` as its first
  parameter. Controllers pass `c.Request().Context()`.
- Repositories always query through `db.WithContext(ctx)`.
- Log with the injected `*slog.Logger` and key-value attributes:
  `log.Error("refresh failed", "user_id", id, "error", err)`. Never log
  passwords, tokens, secrets or full request bodies.

## 9. Models

- Embed `models.Base`: UUID v7 primary key set in `BeforeCreate`,
  `CreatedAt`, `UpdatedAt`.
- A model that defines its own `BeforeCreate` calls `m.Base.BeforeCreate(tx)`
  first.
- Sensitive fields carry `json:"-"` as a second line of defence; resources
  remain the first.

## 10. Migrations

- One change per migration, created with `make migrate-create name=...`.
  Every `up` file has a `down` file that reverses it.
- Never edit a migration that has been applied anywhere outside your
  machine. Write a new one.
- Primary keys are `uuid` without a database default (the application
  generates UUID v7). Timestamps are `timestamptz NOT NULL DEFAULT now()`.
- Index every foreign key and every column that queries filter or join
  on, unless the primary key already narrows the query.

## 11. Tests

- Every change is tested. Test files sit next to the code
  (`user_repository_test.go`).
- Use `testing` with `github.com/stretchr/testify/require` for preconditions
  and `assert` for checks. Use `t.Run` sub-tests for cases.
- Tests that need the database use the real PostgreSQL test database through
  the `internal/testutil` helpers. Do not mock GORM.
- HTTP tests go through the real application handler with
  `testutil.NewClient`, so middleware, routing and the error handler are
  exercised too.
- Test names say what they prove: `TestLogin_AnswersTheSameForAnUnknownEmailAndAWrongPassword`.

## 12. Formatting and comments

- `gofmt`, `go vet` and `staticcheck` must pass (`make lint`).
- **The code explains itself.** Names, small functions and types carry the
  meaning. A comment is the exception, not the rule.
- **Write a comment only when the code cannot say it:** a concept, a
  constraint, a non-obvious line or a deliberate choice. It explains *why*.
- **Never restate a name or a signature.** `// NewUserRepository returns a
  UserRepository` is noise; delete it.
- **Package comment:** one sentence, only when the role of the package is
  not obvious from its name or carries a rule (for example "the only
  package allowed to read the environment").

Good comments, kept because the code cannot express them:

```go
// ctx is already cancelled here, hence the timeout derives from
// context.WithoutCancel(ctx).

// Echo may put parser details in its message, so only the status is kept.
```
