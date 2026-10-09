package controllers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/app"
	"github.com/francky-d/go-starter-kit/internal/config"
	"github.com/francky-d/go-starter-kit/internal/logging"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

const password = "correct horse battery staple"

type session struct {
	User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func newClient(t *testing.T) *testutil.Client {
	t.Helper()
	return newClientWith(t, testutil.Config())
}

func newClientWith(t *testing.T, cfg config.Config) *testutil.Client {
	t.Helper()
	client, _ := newClientAndDB(t, cfg)

	return client
}

// newClientAndDB also returns the test database, for tests that change
// data behind the API's back.
func newClientAndDB(t *testing.T, cfg config.Config) (*testutil.Client, *gorm.DB) {
	t.Helper()
	db := testutil.DB(t)
	application, err := app.New(t.Context(), cfg, logging.Discard())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, application.Close()) })

	return testutil.NewClient(t, application.Handler()), db
}

func register(t *testing.T, client *testutil.Client, email string) session {
	t.Helper()
	response := client.Post("/api/v1/auth/register", map[string]string{"name": "Ada Lovelace", "email": email, "password": password})
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var registered session
	response.Data(&registered)

	return registered
}

func TestRegister_CreatesTheUserAndStartsASession(t *testing.T) {
	registered := register(t, newClient(t), "Ada@Example.com")

	assert.Equal(t, "ada@example.com", registered.User.Email)
	assert.Equal(t, "Ada Lovelace", registered.User.Name)
	assert.Equal(t, "Bearer", registered.TokenType)
	assert.NotEmpty(t, registered.AccessToken)
	assert.NotEmpty(t, registered.RefreshToken)
	assert.InDelta(t, 900, registered.ExpiresIn, 1)
}

func TestRegister_NeverReturnsThePasswordHash(t *testing.T) {
	response := newClient(t).Post("/api/v1/auth/register", map[string]string{"name": "Ada", "email": "ada@example.com", "password": password})

	assert.NotContains(t, response.Body.String(), "password")
}

func TestRegister_ValidatesTheInput(t *testing.T) {
	response := newClient(t).Post("/api/v1/auth/register", map[string]string{"email": "not-an-email", "password": "short"})

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Equal(t, map[string][]string{
		"name":     {"is required"},
		"email":    {"must be a valid email address"},
		"password": {"must be at least 8 characters"},
	}, response.ErrorBody().Errors)
}

func TestRegister_RefusesATakenEmail(t *testing.T) {
	client := newClient(t)
	register(t, client, "ada@example.com")

	response := client.Post("/api/v1/auth/register", map[string]string{"name": "Ada", "email": "ADA@example.com", "password": password})

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, "Unable to register with the provided details.", response.ErrorBody().Message)
}

func TestRegister_RefusesAPasswordOverSeventyTwoBytes(t *testing.T) {
	response := newClient(t).Post("/api/v1/auth/register", map[string]string{"name": "Ada", "email": "ada@example.com", "password": strings.Repeat("é", 37)})

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Equal(t, []string{"must be at most 72 bytes"}, response.ErrorBody().Errors["password"])
}

func TestLogin_StartsASession(t *testing.T) {
	client := newClient(t)
	registered := register(t, client, "ada@example.com")

	response := client.Post("/api/v1/auth/login", map[string]string{"email": "ada@example.com", "password": password})

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var loggedIn session
	response.Data(&loggedIn)
	assert.Equal(t, registered.User.ID, loggedIn.User.ID)
}

func TestLogin_AnswersTheSameForAnUnknownEmailAndAWrongPassword(t *testing.T) {
	client := newClient(t)
	register(t, client, "ada@example.com")

	unknownEmail := client.Post("/api/v1/auth/login", map[string]string{"email": "grace@example.com", "password": password})
	wrongPassword := client.Post("/api/v1/auth/login", map[string]string{"email": "ada@example.com", "password": "wrong password"})

	assert.Equal(t, http.StatusUnauthorized, unknownEmail.Code)
	assert.Equal(t, unknownEmail.Code, wrongPassword.Code)
	assert.Equal(t, unknownEmail.Body.String(), wrongPassword.Body.String())
	assert.Equal(t, "Invalid credentials.", unknownEmail.ErrorBody().Message)
}

func TestRefresh_RotatesTheRefreshToken(t *testing.T) {
	client := newClient(t)
	registered := register(t, client, "ada@example.com")

	response := client.Post("/api/v1/auth/refresh", map[string]string{"refresh_token": registered.RefreshToken})
	reused := client.Post("/api/v1/auth/refresh", map[string]string{"refresh_token": registered.RefreshToken})

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var refreshed session
	response.Data(&refreshed)
	assert.NotEqual(t, registered.RefreshToken, refreshed.RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, reused.Code)
	assert.Equal(t, "Unauthenticated.", reused.ErrorBody().Message)
}

func TestLogout_RevokesTheRefreshToken(t *testing.T) {
	client := newClient(t)
	registered := register(t, client, "ada@example.com")

	logout := client.Post("/api/v1/auth/logout", map[string]string{"refresh_token": registered.RefreshToken})
	again := client.Post("/api/v1/auth/logout", map[string]string{"refresh_token": registered.RefreshToken})
	refresh := client.Post("/api/v1/auth/refresh", map[string]string{"refresh_token": registered.RefreshToken})

	assert.Equal(t, http.StatusNoContent, logout.Code)
	assert.Equal(t, http.StatusNoContent, again.Code)
	assert.Equal(t, http.StatusUnauthorized, refresh.Code)
}

func TestLogin_IsRateLimitedPerClient(t *testing.T) {
	cfg := testutil.Config()
	cfg.Auth.CredentialsRateLimitPerMinute = 2
	client := newClientWith(t, cfg)
	credentials := map[string]string{"email": "ada@example.com", "password": password}

	client.Post("/api/v1/auth/login", credentials)
	client.Post("/api/v1/auth/login", credentials)
	limited := client.Post("/api/v1/auth/login", credentials)

	assert.Equal(t, http.StatusTooManyRequests, limited.Code)
	assert.Equal(t, "Too many requests.", limited.ErrorBody().Message)
}

func TestRefresh_HasItsOwnRateLimit(t *testing.T) {
	cfg := testutil.Config()
	cfg.Auth.CredentialsRateLimitPerMinute = 1
	client := newClientWith(t, cfg)
	registered := register(t, client, "ada@example.com")

	refresh := client.Post("/api/v1/auth/refresh", map[string]string{"refresh_token": registered.RefreshToken})
	login := client.Post("/api/v1/auth/login", map[string]string{"email": "ada@example.com", "password": password})

	assert.Equal(t, http.StatusOK, refresh.Code)
	assert.Equal(t, http.StatusTooManyRequests, login.Code)
}
