# Testing

Every change is covered by a test. Tests use the standard `testing`
package with [testify](https://github.com/stretchr/testify), and run
against a real PostgreSQL database: the database is never mocked.

## Running the tests

```bash
make docker-up     # once: start PostgreSQL
make test          # the whole suite
go test ./internal/repositories/...   # one package while iterating
```

`make test` runs `go test -p 1 ./...`. Every package shares the same test
database, so packages run one at a time; tests inside a package still run
in order.

Tests connect to `app_test` on `localhost:5432`. Set `TEST_DATABASE_URL` to
use another database (see [database.md](database.md#test-database)).

## Helpers

The `internal/testutil` package provides:

| Helper | Gives |
| --- | --- |
| `testutil.DB(t)` | A connection to the migrated test database. Every table is emptied before the test and when it ends. |
| `testutil.Truncate(t, db)` | Empties every table except the migration history. |
| `testutil.Config()` | A complete, valid configuration for tests, pointing at the test database. |
| `testutil.DatabaseConfig()` | The database part of that configuration. |
| `testutil.DatabaseURL()` | `TEST_DATABASE_URL`, or the local test database. |

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

- Name the test after what it proves: `Test<Subject>_<Behaviour>`.
- Use `require` for preconditions and `assert` for the checks.
- Put the test file next to the code, in the `_test` package, so the test
  only uses the exported API.
- Use `t.Context()` for contexts; it is cancelled when the test ends.
