package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/francky-d/go-starter-kit/internal/http/router"
)

func answer(body string) echo.HandlerFunc {
	return func(c echo.Context) error { return c.String(http.StatusOK, body) }
}

// tag appends name to the X-Trace header, to observe which middleware ran
// and in which order.
func tag(name string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-Trace", name)
			return next(c)
		}
	}
}

func send(e *echo.Echo, method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))

	return recorder
}

func trace(response *httptest.ResponseRecorder) string {
	return strings.Join(response.Header().Values("X-Trace"), ",")
}

func TestRouter_RegistersChainedRoutesUnderTheGroupPrefix(t *testing.T) {
	e := echo.New()

	router.New(e).Group("/api/v1").Group("/projects").
		Get("", answer("list")).
		Post("", answer("create")).
		Put("/:id", answer("replace")).
		Patch("/:id", answer("update")).
		Delete("/:id", answer("delete"))

	assert.Equal(t, "list", send(e, http.MethodGet, "/api/v1/projects").Body.String())
	assert.Equal(t, "create", send(e, http.MethodPost, "/api/v1/projects").Body.String())
	assert.Equal(t, "replace", send(e, http.MethodPut, "/api/v1/projects/7").Body.String())
	assert.Equal(t, "update", send(e, http.MethodPatch, "/api/v1/projects/7").Body.String())
	assert.Equal(t, "delete", send(e, http.MethodDelete, "/api/v1/projects/7").Body.String())
}

func TestRouter_ScopesMiddlewareToItsGroup(t *testing.T) {
	e := echo.New()
	api := router.New(e).Group("/api")

	api.Middleware(tag("auth")).Get("/me", answer("me"))
	api.Get("/public", answer("public"))

	assert.Equal(t, "auth", trace(send(e, http.MethodGet, "/api/me")))
	assert.Empty(t, trace(send(e, http.MethodGet, "/api/public")))
}

func TestRouter_RunsGroupMiddlewareBeforeRouteMiddleware(t *testing.T) {
	e := echo.New()

	router.New(e).Group("/api", tag("api")).Middleware(tag("auth")).Get("/me", answer("me"), tag("route"))

	assert.Equal(t, "api,auth,route", trace(send(e, http.MethodGet, "/api/me")))
}

func TestRouter_LeavesUnknownURLsUnderAProtectedPrefixAsNotFound(t *testing.T) {
	e := echo.New()

	router.New(e).Group("/api", tag("auth")).Get("/me", answer("me"))

	response := send(e, http.MethodGet, "/api/unknown")
	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Empty(t, trace(response))
}

func TestRouter_KeepsMethodNotAllowed(t *testing.T) {
	e := echo.New()

	router.New(e).Get("/health", answer("ok"))

	assert.Equal(t, http.StatusMethodNotAllowed, send(e, http.MethodPost, "/health").Code)
}
