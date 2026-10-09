package services_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/repositories"
	"github.com/francky-d/go-starter-kit/internal/services"
)

func TestUserService_CurrentUserReturnsTheUser(t *testing.T) {
	f := newAuthFixture(t)
	registered := f.register(t, "ada@example.com")
	users := services.NewUserService(repositories.NewUserRepository(f.db))

	user, err := users.CurrentUser(t.Context(), registered.User.ID)

	require.NoError(t, err)
	assert.Equal(t, "ada@example.com", user.Email)
}

func TestUserService_CurrentUserOfADeletedAccountIsUnauthenticated(t *testing.T) {
	f := newAuthFixture(t)
	users := services.NewUserService(repositories.NewUserRepository(f.db))

	_, err := users.CurrentUser(t.Context(), uuid.Must(uuid.NewV7()))

	requireStatus(t, err, http.StatusUnauthorized)
}
