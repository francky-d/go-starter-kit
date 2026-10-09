# Testing

Every change is covered by a test. Tests use the standard `testing`
package with [testify](https://github.com/stretchr/testify), and run
against a real PostgreSQL database: the database is never mocked.

## Running the tests

```bash
make docker-up     # once: start PostgreSQL
make test          # the whole suite
make test-cover    # the whole suite, with coverage
go test ./internal/repositories/...   # one package while iterating
```

`make test` runs `go test -p 1 ./...`. Every package shares the same test
database, so packages run one at a time; tests inside a package still run
in order.

Tests connect to `app_test` on `localhost:5432`. Set `TEST_DATABASE_URL` to
use another database (see [database.md](database.md#test-database)).

## Coverage

`make test-cover` runs the suite with coverage of every package under
`internal/`, including code reached from another package's tests: the
end-to-end tests count for the controllers and services they go through.
It prints the total and writes two files, both ignored by git:

| File | Content |
| --- | --- |
| `coverage.out` | The raw profile, for `go tool cover`. |
| `coverage.html` | A page that colours every line by coverage. |

Read the `total:` line and the HTML page. While the tests run, a package
without test files prints `coverage: 0.0% of statements` even when other
packages' tests cover it; the profile counts that coverage.

## End-to-end tests

`internal/app/auth_flow_test.go` drives the API the way a client
application does: register, read the current user, refresh, log out,
log in again, then present an exchanged refresh token and check that
every session is revoked. It covers how the endpoints work together,
which the tests of each endpoint do not.

## Helpers

The `internal/testutil` package provides:

| Helper | Gives |
| --- | --- |
| `testutil.DB(t)` | A connection to the migrated test database. Every table is emptied before the test and when it ends. |
| `testutil.Truncate(t, db)` | Empties every table except the migration history. |
| `testutil.Config()` | A complete, valid configuration for tests, pointing at the test database. |
| `testutil.DatabaseConfig()` | The database part of that configuration. |
| `testutil.DatabaseURL()` | `TEST_DATABASE_URL`, or the local test database. |
| `testutil.NewClient(t, handler)` | A client that sends JSON requests to a handler in memory: `Get`, `Post`, and `WithToken` for a bearer token. |

A response from the client is an `httptest.ResponseRecorder` with two
decoders: `Data(&target)` reads the `data` member of a success response,
and `ErrorBody()` returns the `message` and `errors` of an error
response.

Migrations are applied once per test binary, the first time `testutil.DB`
is called.

## Writing a test

```go
func TestUserRepository_ReportsATakenEmailAsDuplicate(t *testing.T) {
    users := repositories.NewUserRepository(testutil.DB(t))
    require.NoError(t, users.Create(t.Context(), newUser("ada@example.com")))

    err := users.Create(t.Context(), newUser("ada@example.com"))

    assert.ErrorIs(t, err, repositories.ErrDuplicate)
}
```

### Testing an endpoint

Endpoint tests go through the real application, with every middleware,
on the test database:

```go
func newClient(t *testing.T) *testutil.Client {
    t.Helper()
    testutil.DB(t) // empties the tables around the test
    application, err := app.New(t.Context(), testutil.Config(), logging.Discard())
    require.NoError(t, err)
    t.Cleanup(func() { assert.NoError(t, application.Close()) })

    return testutil.NewClient(t, application.Handler())
}

func TestLogin_AnswersTheSameForAnUnknownEmailAndAWrongPassword(t *testing.T) {
    client := newClient(t)
    register(t, client, "ada@example.com")

    unknownEmail := client.Post("/api/v1/auth/login", map[string]string{"email": "grace@example.com", "password": password})
    wrongPassword := client.Post("/api/v1/auth/login", map[string]string{"email": "ada@example.com", "password": "wrong"})

    assert.Equal(t, http.StatusUnauthorized, unknownEmail.Code)
    assert.Equal(t, unknownEmail.Body.String(), wrongPassword.Body.String())
}
```

The test configuration sets very high authentication rate limits, so
only a test that lowers one reaches it.

### Rules

- Name the test after what it proves: `Test<Subject>_<Behaviour>`.
- Use `require` for preconditions and `assert` for the checks.
- Put the test file next to the code, in the `_test` package, so the test
  only uses the exported API.
- Use `t.Context()` for contexts; it is cancelled when the test ends.
