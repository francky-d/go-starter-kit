package middleware

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/auth"
)

const currentUserIDKey = "current_user_id"

// Authenticated lets a request through only with a valid bearer access
// token, and records its user for CurrentUserID.
func Authenticated(tokens *auth.TokenIssuer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, ok := bearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if !ok {
				return challenge(c, "Bearer", apperror.Unauthorized())
			}

			userID, err := tokens.UserID(token)
			if err != nil {
				return challenge(c, `Bearer error="invalid_token"`, apperror.Unauthorized().Wrap(err))
			}

			c.Set(currentUserIDKey, userID)

			return next(c)
		}
	}
}

// challenge tells the client how to authenticate, as RFC 6750 asks.
// invalid_token says that the token sent was refused, never why.
func challenge(c echo.Context, header string, err error) error {
	c.Response().Header().Set(echo.HeaderWWWAuthenticate, header)

	return err
}

// CurrentUserID returns the user recorded by Authenticated. Calling it on a
// route without that middleware is a wiring bug, so it panics.
func CurrentUserID(c echo.Context) uuid.UUID {
	userID, ok := c.Get(currentUserIDKey).(uuid.UUID)
	if !ok {
		panic("middleware: CurrentUserID called on a route without Authenticated")
	}

	return userID
}

// bearerToken reads "Bearer <token>". The scheme is case-insensitive and may
// be followed by several spaces.
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	token = strings.TrimLeft(token, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}

	return token, true
}
