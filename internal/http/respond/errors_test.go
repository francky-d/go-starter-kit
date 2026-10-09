package respond_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
)

func newHandler() (echo.HTTPErrorHandler, *bytes.Buffer) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return respond.ErrorHandler(logger), &logs
}

func TestErrorHandler_AnswersApplicationErrors(t *testing.T) {
	handle, _ := newHandler()
	c, recorder := newContext(http.MethodPost)

	handle(apperror.Validation(map[string][]string{"email": {"must be a valid email address"}}), c)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.JSONEq(t,
		`{"message":"The given data was invalid.","errors":{"email":["must be a valid email address"]}}`,
		recorder.Body.String())
}

func TestErrorHandler_FindsWrappedApplicationErrors(t *testing.T) {
	handle, _ := newHandler()
	c, recorder := newContext(http.MethodGet)

	handle(fmt.Errorf("show project: %w", apperror.NotFound()), c)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.JSONEq(t, `{"message":"Resource not found."}`, recorder.Body.String())
}

func TestErrorHandler_HidesEchoErrorDetails(t *testing.T) {
	handle, _ := newHandler()
	c, recorder := newContext(http.MethodPost)

	handle(echo.NewHTTPError(http.StatusBadRequest, "Syntax error: offset=12, error=invalid character"), c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.JSONEq(t, `{"message":"Bad Request."}`, recorder.Body.String())
}

func TestErrorHandler_AnswersUnknownErrorsWith500AndLogsTheCause(t *testing.T) {
	handle, logs := newHandler()
	c, recorder := newContext(http.MethodGet)

	handle(errors.New("dial tcp 10.0.0.5:5432: connection refused"), c)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"message":"Internal server error."}`, recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), "10.0.0.5")
	assert.Contains(t, logs.String(), `"level":"ERROR"`)
	assert.Contains(t, logs.String(), "connection refused")
}

func TestErrorHandler_LogsOrdinaryClientErrorsAtDebugLevel(t *testing.T) {
	handle, logs := newHandler()
	c, _ := newContext(http.MethodGet)

	handle(apperror.NotFound(), c)

	assert.Contains(t, logs.String(), `"level":"DEBUG"`)
}

func TestErrorHandler_LogsSecurityRejectionsAtInfoLevel(t *testing.T) {
	for _, err := range []*apperror.Error{apperror.Unauthorized(), apperror.Forbidden(), apperror.TooManyRequests()} {
		t.Run(err.Message, func(t *testing.T) {
			handle, logs := newHandler()
			c, _ := newContext(http.MethodGet)

			handle(err.Wrap(errors.New("token is expired")), c)

			assert.Contains(t, logs.String(), `"level":"INFO"`)
			assert.Contains(t, logs.String(), "token is expired")
		})
	}
}

func TestErrorHandler_AnswersHeadRequestsWithoutBody(t *testing.T) {
	handle, _ := newHandler()
	c, recorder := newContext(http.MethodHead)

	handle(apperror.NotFound(), c)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func TestErrorHandler_LeavesCommittedResponsesAlone(t *testing.T) {
	handle, _ := newHandler()
	c, recorder := newContext(http.MethodGet)
	require.NoError(t, c.String(http.StatusOK, "already sent"))

	handle(apperror.Internal(errors.New("late failure")), c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "already sent", recorder.Body.String())
}
