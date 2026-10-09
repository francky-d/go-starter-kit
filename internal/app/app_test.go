package app_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/app"
	"github.com/francky-d/go-starter-kit/internal/database"
	"github.com/francky-d/go-starter-kit/internal/logging"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func newApp(t *testing.T) *app.App {
	t.Helper()
	application, err := app.New(t.Context(), testutil.Config(), logging.Discard())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, application.Close()) })

	return application
}

func get(t *testing.T, application *app.App, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(method, path, nil))

	return recorder
}

func TestHealth_AnswersOK(t *testing.T) {
	response := get(t, newApp(t), http.MethodGet, "/health")

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"data":{"status":"ok","database":"ok"}}`, response.Body.String())
}

func TestNew_AppliesMigrationsWhenAutoMigrateIsEnabled(t *testing.T) {
	migrator, err := database.NewMigrator(t.Context(), testutil.DatabaseConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, migrator.Close()) })
	require.NoError(t, migrator.Up())
	version, _, err := migrator.Version()
	require.NoError(t, err)
	require.NoError(t, migrator.Down(int(version)))

	cfg := testutil.Config()
	cfg.Database.AutoMigrate = true
	application, err := app.New(t.Context(), cfg, logging.Discard())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, application.Close()) })

	migrated, _, err := migrator.Version()
	require.NoError(t, err)
	assert.Equal(t, version, migrated)
}

func TestNew_FailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	cfg := testutil.Config()
	cfg.Database.URL = "postgres://app:app@127.0.0.1:1/app?sslmode=disable"

	_, err := app.New(t.Context(), cfg, logging.Discard())

	assert.ErrorContains(t, err, "database: connect")
}

func TestUnknownRoute_AnswersTheErrorEnvelope(t *testing.T) {
	response := get(t, newApp(t), http.MethodGet, "/does-not-exist")

	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.JSONEq(t, `{"message":"Resource not found."}`, response.Body.String())
}

func TestWrongMethod_AnswersMethodNotAllowed(t *testing.T) {
	response := get(t, newApp(t), http.MethodPost, "/health")

	assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
	assert.JSONEq(t, `{"message":"Method Not Allowed."}`, response.Body.String())
}

func TestServe_StopsGracefullyWhenTheContextIsCancelled(t *testing.T) {
	application := newApp(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- application.Serve(ctx, listener) }()

	response, err := http.Get("http://" + listener.Addr().String() + "/health")
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.JSONEq(t, `{"data":{"status":"ok","database":"ok"}}`, string(body))

	cancel()

	select {
	case err := <-stopped:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after the context was cancelled")
	}
}
