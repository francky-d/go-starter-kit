package services_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/auth"
	"github.com/francky-d/go-starter-kit/internal/logging"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
	"github.com/francky-d/go-starter-kit/internal/services"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

const password = "correct horse battery staple"

type authFixture struct {
	db      *gorm.DB
	service *services.AuthService
	tokens  *auth.TokenIssuer
}

func newAuthFixture(t *testing.T) authFixture {
	t.Helper()
	return newAuthService(t, testutil.DB(t), bcrypt.MinCost)
}

// withCost returns a service on the same database whose hasher uses cost,
// as after a change of BCRYPT_COST.
func (f authFixture) withCost(t *testing.T, cost int) authFixture {
	t.Helper()
	return newAuthService(t, f.db, cost)
}

func newAuthService(t *testing.T, db *gorm.DB, cost int) authFixture {
	t.Helper()
	cfg := testutil.Config().Auth
	passwords, err := auth.NewPasswordHasher(cost)
	require.NoError(t, err)
	tokens := auth.NewTokenIssuer(cfg)
	service := services.NewAuthService(
		db,
		repositories.NewUserRepository(db),
		repositories.NewRefreshTokenRepository(db),
		passwords,
		tokens,
		cfg.RefreshTokenLifetime,
		logging.Discard(),
	)

	return authFixture{db: db, service: service, tokens: tokens}
}

func (f authFixture) register(t *testing.T, email string) services.Session {
	t.Helper()
	session, err := f.service.Register(t.Context(), services.Registration{Name: "Ada Lovelace", Email: email, Password: password})
	require.NoError(t, err)

	return session
}

func (f authFixture) storedToken(t *testing.T, refreshToken string) *models.RefreshToken {
	t.Helper()
	stored, err := repositories.NewRefreshTokenRepository(f.db).FindByHash(t.Context(), auth.HashRefreshToken(refreshToken))
	require.NoError(t, err)

	return stored
}

func requireStatus(t *testing.T, err error, status int) *apperror.Error {
	t.Helper()
	appErr, ok := apperror.As(err)
	require.True(t, ok, "expected an application error, got %v", err)
	require.Equal(t, status, appErr.Status)

	return appErr
}

func TestAuthService_RegisterStartsASession(t *testing.T) {
	f := newAuthFixture(t)

	session := f.register(t, "Ada@Example.com")

	assert.Equal(t, "ada@example.com", session.User.Email)
	assert.NotEqual(t, password, session.User.PasswordHash)
	userID, err := f.tokens.UserID(session.AccessToken.Value)
	require.NoError(t, err)
	assert.Equal(t, session.User.ID, userID)
	stored := f.storedToken(t, session.RefreshToken)
	assert.Equal(t, session.User.ID, stored.UserID)
	assert.NotEqual(t, session.RefreshToken, stored.TokenHash)
}

func TestAuthService_RegisterRefusesATakenEmail(t *testing.T) {
	f := newAuthFixture(t)
	f.register(t, "ada@example.com")

	_, err := f.service.Register(t.Context(), services.Registration{Name: "Ada", Email: "ADA@example.com", Password: password})

	appErr := requireStatus(t, err, http.StatusConflict)
	assert.Equal(t, "Unable to register with the provided details.", appErr.Message)
}

func TestAuthService_RegisterRefusesAPasswordOverSeventyTwoBytes(t *testing.T) {
	f := newAuthFixture(t)

	_, err := f.service.Register(t.Context(), services.Registration{Name: "Ada", Email: "ada@example.com", Password: strings.Repeat("é", 37)})

	appErr := requireStatus(t, err, http.StatusUnprocessableEntity)
	assert.Contains(t, appErr.Fields, "password")
}

func TestAuthService_LoginStartsASession(t *testing.T) {
	f := newAuthFixture(t)
	registered := f.register(t, "ada@example.com")

	session, err := f.service.Login(t.Context(), " ADA@example.com ", password)

	require.NoError(t, err)
	assert.Equal(t, registered.User.ID, session.User.ID)
	assert.NotEqual(t, registered.RefreshToken, session.RefreshToken)
}

func TestAuthService_LoginAnswersTheSameForAnUnknownEmailAndAWrongPassword(t *testing.T) {
	f := newAuthFixture(t)
	f.register(t, "ada@example.com")

	_, unknownEmail := f.service.Login(t.Context(), "grace@example.com", password)
	_, wrongPassword := f.service.Login(t.Context(), "ada@example.com", "wrong password")
	_, extendedPassword := f.service.Login(t.Context(), "ada@example.com", password+strings.Repeat("x", 72))

	expected := requireStatus(t, unknownEmail, http.StatusUnauthorized)
	assert.Equal(t, "Invalid credentials.", expected.Message)
	assert.Equal(t, expected.Message, requireStatus(t, wrongPassword, http.StatusUnauthorized).Message)
	assert.Equal(t, expected.Message, requireStatus(t, extendedPassword, http.StatusUnauthorized).Message)
}

func TestAuthService_LoginUpgradesAHashMadeWithAnOlderCost(t *testing.T) {
	older := newAuthFixture(t)
	registered := older.register(t, "ada@example.com")
	newer := older.withCost(t, bcrypt.MinCost+1)

	_, err := newer.service.Login(t.Context(), "ada@example.com", password)
	require.NoError(t, err)

	user, err := repositories.NewUserRepository(newer.db).FindByID(t.Context(), registered.User.ID)
	require.NoError(t, err)
	cost, err := bcrypt.Cost([]byte(user.PasswordHash))
	require.NoError(t, err)
	assert.Equal(t, bcrypt.MinCost+1, cost)
}

func TestAuthService_RefreshRotatesTheRefreshToken(t *testing.T) {
	f := newAuthFixture(t)
	first := f.register(t, "ada@example.com")

	second, err := f.service.Refresh(t.Context(), first.RefreshToken)

	require.NoError(t, err)
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken)
	assert.Equal(t, first.User.ID, second.User.ID)
	replaced := f.storedToken(t, first.RefreshToken)
	assert.True(t, replaced.IsRevoked())
	assert.Equal(t, f.storedToken(t, second.RefreshToken).ID, *replaced.ReplacedByID)
}

func TestAuthService_RefreshWithAReplacedTokenRevokesEverySession(t *testing.T) {
	f := newAuthFixture(t)
	first := f.register(t, "ada@example.com")
	otherDevice, err := f.service.Login(t.Context(), "ada@example.com", password)
	require.NoError(t, err)
	rotated, err := f.service.Refresh(t.Context(), first.RefreshToken)
	require.NoError(t, err)

	_, err = f.service.Refresh(t.Context(), first.RefreshToken)

	requireStatus(t, err, http.StatusUnauthorized)
	assert.True(t, f.storedToken(t, rotated.RefreshToken).IsRevoked())
	assert.True(t, f.storedToken(t, otherDevice.RefreshToken).IsRevoked())
}

func TestAuthService_ConcurrentRefreshesWithOneTokenRevokeEverySession(t *testing.T) {
	f := newAuthFixture(t)
	session := f.register(t, "ada@example.com")

	results := make(chan refreshResult, 2)
	for range 2 {
		go func() {
			refreshed, err := f.service.Refresh(t.Context(), session.RefreshToken)
			results <- refreshResult{refreshed, err}
		}()
	}
	first, second := <-results, <-results

	winner, loser := first, second
	if first.err != nil {
		winner, loser = second, first
	}
	require.NoError(t, winner.err)
	requireStatus(t, loser.err, http.StatusUnauthorized)
	assert.True(t, f.storedToken(t, winner.session.RefreshToken).IsRevoked())
}

type refreshResult struct {
	session services.Session
	err     error
}

func TestAuthService_RefreshRefusesAnExpiredToken(t *testing.T) {
	f := newAuthFixture(t)
	session := f.register(t, "ada@example.com")
	require.NoError(t, f.db.Model(&models.RefreshToken{}).
		Where("token_hash = ?", auth.HashRefreshToken(session.RefreshToken)).
		Update("expires_at", time.Now().Add(-time.Second)).Error)

	_, err := f.service.Refresh(t.Context(), session.RefreshToken)

	requireStatus(t, err, http.StatusUnauthorized)
}

func TestAuthService_RefreshRefusesAnUnknownToken(t *testing.T) {
	f := newAuthFixture(t)

	_, err := f.service.Refresh(t.Context(), "unknown-token")

	requireStatus(t, err, http.StatusUnauthorized)
}

func TestAuthService_LogoutRevokesOnlyThatSession(t *testing.T) {
	f := newAuthFixture(t)
	session := f.register(t, "ada@example.com")
	otherDevice, err := f.service.Login(t.Context(), "ada@example.com", password)
	require.NoError(t, err)

	require.NoError(t, f.service.Logout(t.Context(), session.RefreshToken))
	require.NoError(t, f.service.Logout(t.Context(), session.RefreshToken))

	_, err = f.service.Refresh(t.Context(), session.RefreshToken)
	requireStatus(t, err, http.StatusUnauthorized)
	assert.False(t, f.storedToken(t, otherDevice.RefreshToken).IsRevoked())
}

func TestAuthService_LogoutIgnoresAnUnknownToken(t *testing.T) {
	f := newAuthFixture(t)

	assert.NoError(t, f.service.Logout(t.Context(), "unknown-token"))
}
