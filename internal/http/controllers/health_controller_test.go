package controllers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/http/controllers"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/logging"
)

type fakePinger struct{ err error }

func (p fakePinger) PingContext(context.Context) error { return p.err }

func showHealth(t *testing.T, database controllers.DatabasePinger) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.HTTPErrorHandler = respond.ErrorHandler(logging.Discard())
	e.GET("/health", controllers.NewHealthController(database).Show)

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	return recorder
}

func TestHealth_ReportsAReachableDatabase(t *testing.T) {
	response := showHealth(t, fakePinger{})

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"data":{"status":"ok","database":"ok"}}`, response.Body.String())
}

func TestHealth_AnswersServiceUnavailableWithoutDetails(t *testing.T) {
	response := showHealth(t, fakePinger{err: errors.New("dial tcp 10.0.0.5:5432: connection refused")})

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.JSONEq(t, `{"message":"Service unavailable."}`, response.Body.String())
}
