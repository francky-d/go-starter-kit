package services

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/auth"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
)

// errRefreshTokenReused rolls back a rotation that lost the race for a token.
var errRefreshTokenReused = errors.New("services: refresh token already used")

type Registration struct {
	Name     string
	Email    string
	Password string
}

type AuthService struct {
	db                   *gorm.DB
	users                *repositories.UserRepository
	refreshTokens        *repositories.RefreshTokenRepository
	passwords            *auth.PasswordHasher
	accessTokens         *auth.TokenIssuer
	refreshTokenLifetime time.Duration
	log                  *slog.Logger
	now                  func() time.Time
}

// NewAuthService takes db only to open the rotation transaction; every
// query goes through the repositories.
func NewAuthService(
	db *gorm.DB,
	users *repositories.UserRepository,
	refreshTokens *repositories.RefreshTokenRepository,
	passwords *auth.PasswordHasher,
	accessTokens *auth.TokenIssuer,
	refreshTokenLifetime time.Duration,
	log *slog.Logger,
) *AuthService {
	return &AuthService{
		db:                   db,
		users:                users,
		refreshTokens:        refreshTokens,
		passwords:            passwords,
		accessTokens:         accessTokens,
		refreshTokenLifetime: refreshTokenLifetime,
		log:                  log,
		now:                  time.Now,
	}
}

func (s *AuthService) Register(ctx context.Context, registration Registration) (Session, error) {
	user, err := s.createUser(ctx, registration)
	if err != nil {
		return Session{}, err
	}

	return s.startSession(ctx, user)
}

// Login answers the same error, after the same bcrypt work, whether the
// email is unknown or the password is wrong: neither the message nor the
// timing tells an attacker which emails have an account.
func (s *AuthService) Login(ctx context.Context, email, password string) (Session, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, repositories.ErrNotFound) {
		s.passwords.SimulateVerify(password)
		return Session{}, invalidCredentials()
	}
	if err != nil {
		return Session{}, apperror.Internal(err)
	}

	if err := s.passwords.Verify(user.PasswordHash, password); err != nil {
		return Session{}, s.rejectPassword(err)
	}

	s.upgradePasswordHash(ctx, user, password)

	return s.startSession(ctx, user)
}

// Refresh exchanges a refresh token for a new session and revokes it. A
// token presented after it was already exchanged revokes every session of
// its user.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	stored, err := s.usableRefreshToken(ctx, refreshToken)
	if err != nil {
		return Session{}, err
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return Session{}, unauthorizedUnlessInternal(err)
	}

	return s.rotate(ctx, user, stored)
}

// Logout revokes a refresh token. Holding the token is the proof of
// ownership, so no access token is needed: a client can sign out after its
// access token expired. An unknown token is ignored, so logging out twice is
// not an error.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	stored, err := s.refreshTokens.FindByHash(ctx, auth.HashRefreshToken(refreshToken))
	if errors.Is(err, repositories.ErrNotFound) {
		return nil
	}
	if err != nil {
		return apperror.Internal(err)
	}

	if _, err := s.refreshTokens.RevokeIfActive(ctx, stored.ID, nil, s.now()); err != nil {
		return apperror.Internal(err)
	}

	return nil
}

func (s *AuthService) createUser(ctx context.Context, registration Registration) (*models.User, error) {
	hash, err := s.passwords.Hash(registration.Password)
	if errors.Is(err, auth.ErrPasswordTooLong) {
		return nil, apperror.Validation(map[string][]string{"password": {"must be at most 72 bytes"}})
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	user := &models.User{Name: registration.Name, Email: registration.Email, PasswordHash: hash}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, registrationError(err)
	}

	return user, nil
}

func (s *AuthService) startSession(ctx context.Context, user *models.User) (Session, error) {
	session, _, err := s.issueSession(ctx, s.refreshTokens, user)
	if err != nil {
		return Session{}, apperror.Internal(err)
	}

	return session, nil
}

// issueSession returns the session and the stored refresh token, whose id
// a rotation records as the replacement of the old token.
func (s *AuthService) issueSession(
	ctx context.Context,
	refreshTokens *repositories.RefreshTokenRepository,
	user *models.User,
) (Session, *models.RefreshToken, error) {
	accessToken, err := s.accessTokens.Issue(user.ID)
	if err != nil {
		return Session{}, nil, err
	}

	refreshToken, hash := auth.NewRefreshToken()
	stored := &models.RefreshToken{UserID: user.ID, TokenHash: hash, ExpiresAt: s.now().Add(s.refreshTokenLifetime)}
	if err := refreshTokens.Create(ctx, stored); err != nil {
		return Session{}, nil, err
	}

	session := Session{User: user, AccessToken: accessToken, RefreshToken: refreshToken, RefreshTokenExpiresAt: stored.ExpiresAt}

	return session, stored, nil
}

func (s *AuthService) usableRefreshToken(ctx context.Context, refreshToken string) (*models.RefreshToken, error) {
	stored, err := s.refreshTokens.FindByHash(ctx, auth.HashRefreshToken(refreshToken))
	if err != nil {
		return nil, unauthorizedUnlessInternal(err)
	}

	if stored.WasReplaced() {
		return nil, s.revokeEverySession(ctx, stored.UserID)
	}

	if stored.IsRevoked() || stored.HasExpired(s.now()) {
		return nil, apperror.Unauthorized()
	}

	return stored, nil
}

// rotate replaces the old token in a transaction. If another request
// exchanged the token in the meantime, the rotation is rolled back and
// treated as a reuse.
func (s *AuthService) rotate(ctx context.Context, user *models.User, old *models.RefreshToken) (Session, error) {
	var session Session
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		session, err = s.replace(ctx, s.refreshTokens.WithTx(tx), user, old)
		return err
	})

	if errors.Is(err, errRefreshTokenReused) {
		return Session{}, s.revokeEverySession(ctx, user.ID)
	}
	if err != nil {
		return Session{}, apperror.Internal(err)
	}

	return session, nil
}

// replace issues the new session first, then revokes the old token only if
// it is still active, recording the new token as its replacement.
func (s *AuthService) replace(
	ctx context.Context,
	refreshTokens *repositories.RefreshTokenRepository,
	user *models.User,
	old *models.RefreshToken,
) (Session, error) {
	session, replacement, err := s.issueSession(ctx, refreshTokens, user)
	if err != nil {
		return Session{}, err
	}

	revoked, err := refreshTokens.RevokeIfActive(ctx, old.ID, &replacement.ID, s.now())
	if err != nil {
		return Session{}, err
	}
	if !revoked {
		return Session{}, errRefreshTokenReused
	}

	return session, nil
}

// revokeEverySession answers a reused refresh token: the token was stolen or
// replayed, so every device of the user must sign in again.
func (s *AuthService) revokeEverySession(ctx context.Context, userID uuid.UUID) error {
	s.log.WarnContext(ctx, "refresh token reused, revoking every session of the user", "user_id", userID)

	if err := s.refreshTokens.RevokeAllForUser(ctx, userID, s.now()); err != nil {
		return apperror.Internal(err)
	}

	return apperror.Unauthorized()
}

// upgradePasswordHash rehashes a password stored with another BCRYPT_COST,
// so that every hash ends up at the configured cost. A failure is only
// logged: the password was correct, and the next login tries again.
func (s *AuthService) upgradePasswordHash(ctx context.Context, user *models.User, password string) {
	if !s.passwords.NeedsRehash(user.PasswordHash) {
		return
	}

	hash, err := s.passwords.Hash(password)
	if err == nil {
		err = s.users.UpdatePasswordHash(ctx, user.ID, hash)
	}
	if err != nil {
		s.log.WarnContext(ctx, "password hash upgrade failed", "user_id", user.ID, "error", err)
		return
	}

	user.PasswordHash = hash
}

func (s *AuthService) rejectPassword(err error) error {
	if errors.Is(err, auth.ErrPasswordMismatch) {
		return invalidCredentials()
	}

	return apperror.Internal(err)
}

func invalidCredentials() *apperror.Error {
	return apperror.New(http.StatusUnauthorized, "Invalid credentials.")
}

func unauthorizedUnlessInternal(err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.Unauthorized()
	}

	return apperror.Internal(err)
}

// registrationError admits that an email is taken: hiding it would need an
// email verification flow, which the kit does not have. Rate limiting on the
// registration route limits enumeration.
func registrationError(err error) error {
	if errors.Is(err, repositories.ErrDuplicate) {
		return apperror.Conflict("Unable to register with the provided details.")
	}

	return apperror.Internal(err)
}
