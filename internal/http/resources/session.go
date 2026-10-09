package resources

import (
	"time"

	"github.com/francky-d/go-starter-kit/internal/services"
)

type Session struct {
	User                  User      `json:"user"`
	TokenType             string    `json:"token_type"`
	AccessToken           string    `json:"access_token"`
	ExpiresIn             int       `json:"expires_in"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// NewSession follows the OAuth 2 token response: expires_in is the number of
// seconds the access token stays valid from now.
func NewSession(session services.Session) Session {
	return Session{
		User:                  NewUser(session.User),
		TokenType:             "Bearer",
		AccessToken:           session.AccessToken.Value,
		ExpiresIn:             int(time.Until(session.AccessToken.ExpiresAt).Round(time.Second).Seconds()),
		RefreshToken:          session.RefreshToken,
		RefreshTokenExpiresAt: session.RefreshTokenExpiresAt,
	}
}
