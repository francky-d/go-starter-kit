package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/francky-d/go-starter-kit/internal/config"
)

// clockSkew tolerates small clock differences between servers.
const clockSkew = 30 * time.Second

var ErrInvalidToken = errors.New("auth: invalid token")

type AccessToken struct {
	Value     string
	ExpiresAt time.Time
}

type TokenIssuer struct {
	secret   []byte
	issuer   string
	lifetime time.Duration
	now      func() time.Time
}

func NewTokenIssuer(cfg config.Auth) *TokenIssuer {
	return &TokenIssuer{
		secret:   []byte(cfg.JWTSecret),
		issuer:   cfg.JWTIssuer,
		lifetime: cfg.AccessTokenLifetime,
		now:      time.Now,
	}
}

// WithClock returns a copy that reads the time from now. Tests use it to
// move past an expiry without waiting.
func (i *TokenIssuer) WithClock(now func() time.Time) *TokenIssuer {
	clocked := *i
	clocked.now = now

	return &clocked
}

func (i *TokenIssuer) Issue(userID uuid.UUID) (AccessToken, error) {
	tokenID, err := uuid.NewV7()
	if err != nil {
		return AccessToken{}, fmt.Errorf("auth: issue access token: %w", err)
	}

	issuedAt := i.now()
	expiresAt := issuedAt.Add(i.lifetime)

	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ID:        tokenID.String(),
		Subject:   userID.String(),
		Issuer:    i.issuer,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}).SignedString(i.secret)
	if err != nil {
		return AccessToken{}, fmt.Errorf("auth: sign access token: %w", err)
	}

	return AccessToken{Value: value, ExpiresAt: expiresAt}, nil
}

// UserID returns the user an access token was issued to. Every failure is
// ErrInvalidToken, so callers cannot leak why a token was refused.
func (i *TokenIssuer) UserID(token string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims
	if _, err := jwt.ParseWithClaims(token, &claims, i.signingKey, i.parserOptions()...); err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: subject: %w", ErrInvalidToken, err)
	}

	return userID, nil
}

func (i *TokenIssuer) signingKey(*jwt.Token) (any, error) {
	return i.secret, nil
}

// parserOptions pins the algorithm, so a token signed with "none" or another
// algorithm is refused before its signature is even checked.
func (i *TokenIssuer) parserOptions() []jwt.ParserOption {
	return []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(i.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(clockSkew),
		jwt.WithTimeFunc(i.now),
	}
}
