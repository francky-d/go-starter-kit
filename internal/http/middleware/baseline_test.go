package middleware_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/config"
	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/logging"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

type server struct {
	echo *echo.Echo
	logs *bytes.Buffer
}

func newServer(t *testing.T, change func(*config.Config)) server {
	t.Helper()
	cfg := testutil.Config()
	cfg.HTTP.RequestTimeout = 50 * time.Millisecond
	cfg.HTTP.BodyLimit = "16B"
	if change != nil {
		change(&cfg)
	}

	logs := &bytes.Buffer{}
	log := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatJSON}, logs)

	e := echo.New()
	e.HTTPErrorHandler = respond.ErrorHandler(log)
	e.Use(middleware.Baseline(cfg, log)...)
	registerTestRoutes(e)
	// Like a service logging with the request context.
	e.GET("/log", func(c echo.Context) error {
		log.InfoContext(c.Request().Context(), "handler step")
		return c.NoContent(http.StatusNoContent)
	})

	return server{echo: e, logs: logs}
}

func registerTestRoutes(e *echo.Echo) {
	e.GET("/ok", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	e.GET("/panic", func(echo.Context) error { panic("database exploded") })
	e.GET("/slow", func(c echo.Context) error {
		<-c.Request().Context().Done()
		return c.Request().Context().Err()
	})
	// Like a service whose query was cut by the deadline.
	e.GET("/slow-query", func(c echo.Context) error {
		<-c.Request().Context().Done()
		return apperror.Internal(fmt.Errorf("find user: %w", c.Request().Context().Err()))
	})
	// Like a service whose own outbound call timed out, well before the
	// request deadline.
	e.GET("/inner-timeout", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), time.Millisecond)
		defer cancel()
		<-ctx.Done()
		return apperror.Internal(fmt.Errorf("call partner: %w", ctx.Err()))
	})
	// Like a service that cancelled its own call.
	e.GET("/inner-cancel", func(c echo.Context) error {
		ctx, cancel := context.WithCancel(c.Request().Context())
		cancel()
		return apperror.Internal(fmt.Errorf("call partner: %w", ctx.Err()))
	})
	e.POST("/upload", func(c echo.Context) error {
		var body bytes.Buffer
		if _, err := body.ReadFrom(c.Request().Body); err != nil {
			return err
		}
		return c.String(http.StatusOK, body.String())
	})
}

func (s server) send(request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	s.echo.ServeHTTP(recorder, request)

	return recorder
}

func (s server) get(path string) *httptest.ResponseRecorder {
	return s.send(httptest.NewRequest(http.MethodGet, path, nil))
}

func TestBaseline_GivesEveryResponseARequestID(t *testing.T) {
	response := newServer(t, nil).get("/ok")

	assert.NotEmpty(t, response.Header().Get(echo.HeaderXRequestID))
}

func requestWithID(id string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/ok", nil)
	request.Header.Set(echo.HeaderXRequestID, id)

	return request
}

func TestBaseline_KeepsAPlainClientRequestID(t *testing.T) {
	response := newServer(t, nil).send(requestWithID("trace-42.a_b"))

	assert.Equal(t, "trace-42.a_b", response.Header().Get(echo.HeaderXRequestID))
}

func TestBaseline_ReplacesAnOversizedOrUnusualClientRequestID(t *testing.T) {
	for _, id := range []string{strings.Repeat("a", 65), "id with spaces", "id\nforged-log-line"} {
		response := newServer(t, nil).send(requestWithID(id))

		assert.NotEqual(t, id, response.Header().Get(echo.HeaderXRequestID))
		assert.NotEmpty(t, response.Header().Get(echo.HeaderXRequestID))
	}
}

func TestBaseline_LogsOneRecordPerRequestWithoutTheQueryString(t *testing.T) {
	s := newServer(t, nil)

	response := s.get("/ok?token=secret")

	assert.Contains(t, s.logs.String(), `"msg":"request"`)
	assert.Contains(t, s.logs.String(), `"status":200`)
	assert.Contains(t, s.logs.String(), response.Header().Get(echo.HeaderXRequestID))
	assert.NotContains(t, s.logs.String(), "secret")
}

func TestBaseline_LogsSuccessfulHealthProbesAtDebugLevelOnly(t *testing.T) {
	s := newServer(t, nil)
	s.echo.GET("/health", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	s.get("/health")

	assert.NotContains(t, s.logs.String(), `"msg":"request"`)
}

func TestBaseline_TurnsAPanicIntoAGeneric500AndLogsTheStack(t *testing.T) {
	s := newServer(t, nil)

	response := s.get("/panic")

	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.JSONEq(t, `{"message":"Internal server error."}`, response.Body.String())
	assert.Contains(t, s.logs.String(), "panic recovered")
	assert.Contains(t, s.logs.String(), "database exploded")
	assert.Contains(t, s.logs.String(), `"status":500`)
}

func TestBaseline_SetsSecurityHeaders(t *testing.T) {
	headers := newServer(t, nil).get("/ok").Header()

	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", headers.Get("Content-Security-Policy"))
	assert.Equal(t, "no-referrer", headers.Get("Referrer-Policy"))
}

func TestBaseline_RejectsBodiesOverTheLimit(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(strings.Repeat("x", 17)))

	response := newServer(t, nil).send(request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	assert.JSONEq(t, `{"message":"Request Entity Too Large."}`, response.Body.String())
}

func TestBaseline_AnswersSlowRequestsWith503(t *testing.T) {
	response := newServer(t, nil).get("/slow")

	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.JSONEq(t, `{"message":"Service unavailable."}`, response.Body.String())
}

func TestBaseline_AnswersWith503WhenTheDeadlineCutsWrappedWork(t *testing.T) {
	response := newServer(t, nil).get("/slow-query")

	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.JSONEq(t, `{"message":"Service unavailable."}`, response.Body.String())
}

func TestBaseline_GivesTheRequestIDToRecordsLoggedWithTheRequestContext(t *testing.T) {
	s := newServer(t, nil)

	response := s.get("/log")

	assertEveryRecordNames(t, s.logs, response.Header().Get(echo.HeaderXRequestID), 2)
	assert.Contains(t, s.logs.String(), `"msg":"handler step"`)
}

func TestBaseline_WritesTheRequestIDOnceInEveryRecordOfTheRequest(t *testing.T) {
	s := newServer(t, nil)

	response := s.get("/panic")

	// panic recovered, request failed, then request
	assertEveryRecordNames(t, s.logs, response.Header().Get(echo.HeaderXRequestID), 3)
}

func assertEveryRecordNames(t *testing.T, logs *bytes.Buffer, requestID string, records int) {
	t.Helper()
	require.NotEmpty(t, requestID)
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, records)
	for _, line := range lines {
		assert.Equal(t, 1, strings.Count(line, `"request_id"`), line)
		assert.Contains(t, line, `"request_id":"`+requestID+`"`)
	}
}

// The response reaches no one: what matters is the status in the logs.
func TestBaseline_LogsAClientDisconnectAs499(t *testing.T) {
	s := newServer(t, nil)
	ctx, closeConnection := context.WithCancel(t.Context())
	closeConnection()
	request := httptest.NewRequest(http.MethodGet, "/slow-query", nil).WithContext(ctx)

	response := s.send(request)

	assert.Equal(t, apperror.StatusClientClosedRequest, response.Code)
	assert.Contains(t, s.logs.String(), `"status":499`)
	assert.NotContains(t, s.logs.String(), `"level":"ERROR"`)
}

func TestBaseline_KeepsTheStatusOfACancellationTheHandlerMade(t *testing.T) {
	response := newServer(t, nil).get("/inner-cancel")

	assert.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestBaseline_KeepsTheStatusOfADeadlineTheHandlerSet(t *testing.T) {
	response := newServer(t, nil).get("/inner-timeout")

	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.JSONEq(t, `{"message":"Internal server error."}`, response.Body.String())
}

func preflight(origin string) *http.Request {
	request := httptest.NewRequest(http.MethodOptions, "/ok", nil)
	request.Header.Set(echo.HeaderOrigin, origin)
	request.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)

	return request
}

func TestBaseline_SendsNoCORSHeadersWithoutAllowedOrigins(t *testing.T) {
	response := newServer(t, nil).send(preflight("https://app.example.com"))

	assert.Empty(t, response.Header().Get(echo.HeaderAccessControlAllowOrigin))
}

func TestBaseline_AllowsOnlyTheConfiguredOrigins(t *testing.T) {
	s := newServer(t, func(cfg *config.Config) {
		cfg.CORS.AllowedOrigins = []string{"https://app.example.com"}
	})

	allowed := s.send(preflight("https://app.example.com"))
	refused := s.send(preflight("https://evil.example.com"))

	assert.Equal(t, "https://app.example.com", allowed.Header().Get(echo.HeaderAccessControlAllowOrigin))
	assert.Empty(t, allowed.Header().Get(echo.HeaderAccessControlAllowCredentials))
	assert.Empty(t, refused.Header().Get(echo.HeaderAccessControlAllowOrigin))
}

func forwardedRequest(peer string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = peer + ":51000"
	request.Header.Set(echo.HeaderXForwardedFor, "198.51.100.1")

	return request
}

func TestClientIP_IgnoresForwardedHeadersUnlessTheProxyIsTrusted(t *testing.T) {
	assert.Equal(t, "10.0.0.2", middleware.ClientIP(false)(forwardedRequest("10.0.0.2")))
	assert.Equal(t, "198.51.100.1", middleware.ClientIP(true)(forwardedRequest("10.0.0.2")))
}

func TestClientIP_TrustsForwardedHeadersOnlyFromPrivateNetworkPeers(t *testing.T) {
	assert.Equal(t, "203.0.113.7", middleware.ClientIP(true)(forwardedRequest("203.0.113.7")))
}
