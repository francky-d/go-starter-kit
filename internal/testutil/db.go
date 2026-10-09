package testutil

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/database"
	"github.com/francky-d/go-starter-kit/internal/logging"
)

var (
	migrateOnce  sync.Once
	migrateError error
)

// DB connects to the migrated test database. Tables are emptied before the
// test, in case an earlier run crashed, and again when it ends.
func DB(t *testing.T) *gorm.DB {
	t.Helper()
	requireMigrated(t)

	db, err := database.Connect(context.Background(), DatabaseConfig(), logging.Discard())
	require.NoError(t, err, "start the test database with `make docker-up`")

	Truncate(t, db)
	t.Cleanup(func() {
		Truncate(t, db)
		require.NoError(t, database.Close(db))
	})

	return db
}

func Truncate(t *testing.T, db *gorm.DB) {
	t.Helper()

	var tables []string
	require.NoError(t, db.Raw(
		"SELECT quote_ident(tablename) FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'schema_migrations'",
	).Scan(&tables).Error)

	if len(tables) > 0 {
		require.NoError(t, db.Exec("TRUNCATE "+strings.Join(tables, ", ")+" CASCADE").Error)
	}
}

func requireMigrated(t *testing.T) {
	t.Helper()
	migrateOnce.Do(func() {
		migrateError = database.ApplyMigrations(context.Background(), DatabaseConfig())
	})
	require.NoError(t, migrateError, "start the test database with `make docker-up`")
}
