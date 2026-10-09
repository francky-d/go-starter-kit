package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/logging"
)

func rateLimitedServer(requestsPerMinute int) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = respond.ErrorHandler(logging.Discard())
	e.IPExtractor = echo.ExtractIPDirect()
	e.GET("/login", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }, middleware.RateLimit(requestsPerMinute))

	return e
}

func requestFrom(e *echo.Echo, remoteAddress string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	request.RemoteAddr = remoteAddress
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)

	return recorder
}

func TestRateLimit_RefusesRequestsBeyondTheLimit(t *testing.T) {
	e := rateLimitedServer(2)

	requestFrom(e, "203.0.113.7:1000")
	requestFrom(e, "203.0.113.7:1001")
	limited := requestFrom(e, "203.0.113.7:1002")

	assert.Equal(t, http.StatusTooManyRequests, limited.Code)
	assert.JSONEq(t, `{"message":"Too many requests."}`, limited.Body.String())
	assert.Equal(t, "60", limited.Header().Get(echo.HeaderRetryAfter))
}

func TestRateLimit_CountsEachClientIPSeparately(t *testing.T) {
	e := rateLimitedServer(1)

	first := requestFrom(e, "203.0.113.7:1000")
	other := requestFrom(e, "198.51.100.4:1000")

	assert.Equal(t, http.StatusNoContent, first.Code)
	assert.Equal(t, http.StatusNoContent, other.Code)
}
