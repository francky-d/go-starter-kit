package app_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/testutil"
)

type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// flow drives the API the way a client application does, failing the test
// at the first unexpected status.
type flow struct {
	t      *testing.T
	client *testutil.Client
}

func (f flow) expect(response *testutil.Response, status int) *testutil.Response {
	f.t.Helper()
	require.Equal(f.t, status, response.Code, response.Body.String())

	return response
}

func (f flow) session(response *testutil.Response) tokens {
	f.t.Helper()
	var session tokens
	f.expect(response, http.StatusOK).Data(&session)

	return session
}

func (f flow) refresh(refreshToken string) *testutil.Response {
	f.t.Helper()
	return f.client.Post("/api/v1/auth/refresh", map[string]string{"refresh_token": refreshToken})
}

func TestAuthenticationFlow(t *testing.T) {
	testutil.DB(t)
	f := flow{t: t, client: testutil.NewClient(t, newApp(t).Handler())}
	credentials := map[string]string{"email": "ada@example.com", "password": "correct horse battery staple"}

	var registered tokens
	f.expect(f.client.Post("/api/v1/auth/register", map[string]string{
		"name": "Ada Lovelace", "email": credentials["email"], "password": credentials["password"],
	}), http.StatusCreated).Data(&registered)
	f.expect(f.client.WithToken(registered.AccessToken).Get("/api/v1/me"), http.StatusOK)

	refreshed := f.session(f.refresh(registered.RefreshToken))
	f.expect(f.client.WithToken(refreshed.AccessToken).Get("/api/v1/me"), http.StatusOK)

	f.expect(f.client.Post("/api/v1/auth/logout", map[string]string{"refresh_token": refreshed.RefreshToken}), http.StatusNoContent)
	f.expect(f.refresh(refreshed.RefreshToken), http.StatusUnauthorized)
	f.expect(f.client.WithToken(refreshed.AccessToken).Get("/api/v1/me"), http.StatusOK) // valid until it expires

	loggedIn := f.session(f.client.Post("/api/v1/auth/login", credentials))
	f.expect(f.refresh(registered.RefreshToken), http.StatusUnauthorized)
	f.expect(f.refresh(loggedIn.RefreshToken), http.StatusUnauthorized)
}
