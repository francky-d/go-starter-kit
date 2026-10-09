package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/auth"
	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/logging"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func protectedServer(tokens *auth.TokenIssuer) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = respond.ErrorHandler(logging.Discard())
	e.GET("/me", func(c echo.Context) error {
		return c.String(http.StatusOK, middleware.CurrentUserID(c).String())
	}, middleware.Authenticated(tokens))

	return e
}

func requestWithAuthorization(e *echo.Echo, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authorization != "" {
		request.Header.Set(echo.HeaderAuthorization, authorization)
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)

	return recorder
}

func issue(t *testing.T, tokens *auth.TokenIssuer, userID uuid.UUID) string {
	t.Helper()
	token, err := tokens.Issue(userID)
	require.NoError(t, err)

	return token.Value
}

func TestAuthenticated_LetsAValidBearerTokenThrough(t *testing.T) {
	tokens := auth.NewTokenIssuer(testutil.Config().Auth)
	userID := uuid.Must(uuid.NewV7())

	response := requestWithAuthorization(protectedServer(tokens), "Bearer "+issue(t, tokens, userID))

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, userID.String(), response.Body.String())
}

func TestAuthenticated_AcceptsTheSchemeInAnyCaseAndSeveralSpaces(t *testing.T) {
	tokens := auth.NewTokenIssuer(testutil.Config().Auth)
	token := issue(t, tokens, uuid.Must(uuid.NewV7()))

	lowercase := requestWithAuthorization(protectedServer(tokens), "bearer "+token)
	spaced := requestWithAuthorization(protectedServer(tokens), "Bearer   "+token)

	assert.Equal(t, http.StatusOK, lowercase.Code)
	assert.Equal(t, http.StatusOK, spaced.Code)
}

func TestAuthenticated_RefusesRequestsWithoutAValidToken(t *testing.T) {
	tokens := auth.NewTokenIssuer(testutil.Config().Auth)
	expired := issue(t, tokens.WithClock(func() time.Time { return time.Now().Add(-time.Hour) }), uuid.Must(uuid.NewV7()))
	const invalidToken = `Bearer error="invalid_token"`

	cases := map[string]struct{ authorization, challenge string }{
		"no header":        {"", "Bearer"},
		"another scheme":   {"Basic YWRhOnNlY3JldA==", "Bearer"},
		"no token":         {"Bearer ", "Bearer"},
		"not a token":      {"Bearer not-a-jwt", invalidToken},
		"an expired token": {"Bearer " + expired, invalidToken},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			response := requestWithAuthorization(protectedServer(tokens), tc.authorization)

			assert.Equal(t, http.StatusUnauthorized, response.Code)
			assert.JSONEq(t, `{"message":"Unauthenticated."}`, response.Body.String())
			assert.Equal(t, tc.challenge, response.Header().Get(echo.HeaderWWWAuthenticate))
		})
	}
}

func TestCurrentUserID_PanicsOnARouteWithoutAuthentication(t *testing.T) {
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())

	assert.Panics(t, func() { middleware.CurrentUserID(c) })
}
