package database_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/database"
	"github.com/francky-d/go-starter-kit/internal/logging"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func TestConnect_OpensAConfiguredPool(t *testing.T) {
	cfg := testutil.DatabaseConfig()

	db, err := database.Connect(t.Context(), cfg, logging.Discard())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, database.Close(db)) })

	var one int
	require.NoError(t, db.Raw("SELECT 1").Scan(&one).Error)
	assert.Equal(t, 1, one)

	pool, err := db.DB()
	require.NoError(t, err)
	assert.Equal(t, cfg.MaxOpenConnections, pool.Stats().MaxOpenConnections)
}

func TestConnect_FailsFastWithoutLeakingThePassword(t *testing.T) {
	cfg := testutil.DatabaseConfig()
	cfg.URL = "postgres://app:s3cret-password@127.0.0.1:1/app?sslmode=disable"
	cfg.ConnectTimeout = 2 * time.Second

	started := time.Now()
	_, err := database.Connect(t.Context(), cfg, logging.Discard())

	require.Error(t, err)
	assert.Less(t, time.Since(started), 5*time.Second)
	assert.Contains(t, err.Error(), "database: connect")
	assert.NotContains(t, err.Error(), "s3cret-password")
}
