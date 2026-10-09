// Package services holds the business rules. Services return apperror values
// and know nothing about HTTP.
package services

import (
	"time"

	"github.com/francky-d/go-starter-kit/internal/auth"
	"github.com/francky-d/go-starter-kit/internal/models"
)

// Session is what a client receives when it signs in: the user, a
// short-lived access token and the refresh token that renews it.
type Session struct {
	User                  *models.User
	AccessToken           auth.AccessToken
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}
