package repositories_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/auth"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

type tokenFixture struct {
	db     *gorm.DB
	user   *models.User
	tokens *repositories.RefreshTokenRepository
}

func newTokenFixture(t *testing.T) tokenFixture {
	t.Helper()
	db := testutil.DB(t)
	user := newUser("ada@example.com")
	require.NoError(t, repositories.NewUserRepository(db).Create(t.Context(), user))

	return tokenFixture{db: db, user: user, tokens: repositories.NewRefreshTokenRepository(db)}
}

func (f tokenFixture) storeToken(t *testing.T) (*models.RefreshToken, string) {
	t.Helper()
	value, hash := auth.NewRefreshToken()
	token := &models.RefreshToken{UserID: f.user.ID, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, f.tokens.Create(t.Context(), token))

	return token, value
}

func TestRefreshTokenRepository_FindsATokenByItsHash(t *testing.T) {
	f := newTokenFixture(t)
	stored, value := f.storeToken(t)

	found, err := f.tokens.FindByHash(t.Context(), auth.HashRefreshToken(value))

	require.NoError(t, err)
	assert.Equal(t, stored.ID, found.ID)
	assert.False(t, found.IsRevoked())

	_, err = f.tokens.FindByHash(t.Context(), auth.HashRefreshToken("unknown"))
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestRefreshTokenRepository_RevokesAnActiveTokenOnlyOnce(t *testing.T) {
	f := newTokenFixture(t)
	old, _ := f.storeToken(t)
	replacement, _ := f.storeToken(t)

	first, err := f.tokens.RevokeIfActive(t.Context(), old.ID, &replacement.ID, time.Now())
	require.NoError(t, err)
	second, err := f.tokens.RevokeIfActive(t.Context(), old.ID, nil, time.Now())
	require.NoError(t, err)

	assert.True(t, first)
	assert.False(t, second)

	revoked, err := f.tokens.FindByHash(t.Context(), old.TokenHash)
	require.NoError(t, err)
	assert.True(t, revoked.IsRevoked())
	assert.Equal(t, replacement.ID, *revoked.ReplacedByID)
}

func TestRefreshTokenRepository_RevokesEveryTokenOfAUser(t *testing.T) {
	f := newTokenFixture(t)
	first, _ := f.storeToken(t)
	second, _ := f.storeToken(t)

	require.NoError(t, f.tokens.RevokeAllForUser(t.Context(), f.user.ID, time.Now()))

	for _, token := range []*models.RefreshToken{first, second} {
		reloaded, err := f.tokens.FindByHash(t.Context(), token.TokenHash)
		require.NoError(t, err)
		assert.True(t, reloaded.IsRevoked())
	}
}

func TestRefreshTokenRepository_DeletesTokensWithTheirUser(t *testing.T) {
	f := newTokenFixture(t)
	token, _ := f.storeToken(t)

	require.NoError(t, f.db.Delete(f.user).Error)

	_, err := f.tokens.FindByHash(t.Context(), token.TokenHash)
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}
