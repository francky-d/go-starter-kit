package controllers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func TestMe_ReturnsTheAuthenticatedUser(t *testing.T) {
	client := newClient(t)
	registered := register(t, client, "ada@example.com")

	response := client.WithToken(registered.AccessToken).Get("/api/v1/me")

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var me struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	response.Data(&me)
	assert.Equal(t, registered.User.ID, me.ID)
	assert.Equal(t, "ada@example.com", me.Email)
}

func TestMe_RequiresAnAccessToken(t *testing.T) {
	client := newClient(t)
	registered := register(t, client, "ada@example.com")

	withoutToken := client.Get("/api/v1/me")
	withRefreshToken := client.WithToken(registered.RefreshToken).Get("/api/v1/me")

	assert.Equal(t, http.StatusUnauthorized, withoutToken.Code)
	assert.Equal(t, http.StatusUnauthorized, withRefreshToken.Code)
}

func TestMe_RefusesTheTokenOfADeletedUser(t *testing.T) {
	client, db := newClientAndDB(t, testutil.Config())
	registered := register(t, client, "ada@example.com")
	deleted := db.Exec("DELETE FROM users WHERE id = ?", registered.User.ID)
	require.NoError(t, deleted.Error)
	require.EqualValues(t, 1, deleted.RowsAffected)

	response := client.WithToken(registered.AccessToken).Get("/api/v1/me")

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Equal(t, "Unauthenticated.", response.ErrorBody().Message)
}

func TestAPI_AnswersNotFoundForUnknownRoutesEvenWithoutAToken(t *testing.T) {
	response := newClient(t).Get("/api/v1/unknown")

	assert.Equal(t, http.StatusNotFound, response.Code)
}
