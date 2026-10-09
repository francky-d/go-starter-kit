package database_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/database"
	"github.com/francky-d/go-starter-kit/internal/logging"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func newMigrator(t *testing.T) *database.Migrator {
	t.Helper()
	migrator, err := database.NewMigrator(t.Context(), testutil.DatabaseConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, migrator.Close()) })

	return migrator
}

func TestMigrator_AppliesAndRollsBackEveryMigration(t *testing.T) {
	migrator := newMigrator(t)
	require.NoError(t, migrator.Up())

	version, dirty, err := migrator.Version()
	require.NoError(t, err)
	require.False(t, dirty)
	require.GreaterOrEqual(t, version, uint(1))

	require.NoError(t, migrator.Down(int(version)))
	version, _, err = migrator.Version()
	require.NoError(t, err)
	assert.Zero(t, version)

	require.NoError(t, migrator.Up(), "migrations must apply again after a full rollback")
}

func TestMigrator_UpTwiceIsANoOp(t *testing.T) {
	migrator := newMigrator(t)

	require.NoError(t, migrator.Up())
	assert.NoError(t, migrator.Up())
}

func TestMigrator_DownBeyondTheAppliedCountRollsEverythingBack(t *testing.T) {
	migrator := newMigrator(t)
	require.NoError(t, migrator.Up())
	t.Cleanup(func() { assert.NoError(t, migrator.Up()) })

	require.NoError(t, migrator.Down(1000))

	version, _, err := migrator.Version()
	require.NoError(t, err)
	assert.Zero(t, version)
}

func TestNewMigrator_FailsFastWhenTheDatabaseIsUnreachable(t *testing.T) {
	cfg := testutil.DatabaseConfig()
	cfg.URL = "postgres://app:app@127.0.0.1:1/app?sslmode=disable"
	cfg.ConnectTimeout = time.Second

	_, err := database.NewMigrator(t.Context(), cfg)

	assert.ErrorContains(t, err, "database: connect")
}

func TestMigrator_RejectsDownWithoutSteps(t *testing.T) {
	assert.Error(t, newMigrator(t).Down(0))
}

// rolledBackTransaction gives a test a migrated database whose writes are
// discarded at the end of the test.
func rolledBackTransaction(t *testing.T) *gorm.DB {
	t.Helper()
	require.NoError(t, newMigrator(t).Up())

	db, err := database.Connect(t.Context(), testutil.DatabaseConfig(), logging.Discard())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, database.Close(db)) })

	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	return tx
}

func insertUser(tx *gorm.DB, email string) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}

	return tx.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, 'Ada', ?, 'hash')", id, email).Error
}

func TestUsersTable_RejectsUppercaseEmails(t *testing.T) {
	tx := rolledBackTransaction(t)

	assert.ErrorIs(t, insertUser(tx, "Ada@Example.com"), gorm.ErrCheckConstraintViolated)
}

func TestUsersTable_RejectsDuplicateEmails(t *testing.T) {
	tx := rolledBackTransaction(t)
	require.NoError(t, insertUser(tx, "ada@example.com"))

	assert.ErrorIs(t, insertUser(tx, "ada@example.com"), gorm.ErrDuplicatedKey)
}
