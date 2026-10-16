package middleware

import (
	"context"
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

// ClientDisconnect answers 499 for work cut because the client closed the
// connection: it is not a server failure, so it is not logged as one.
func ClientDisconnect() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil && clientClosed(c, err) {
				return apperror.ClientClosedRequest().Wrap(err)
			}

			return err
		}
	}
}

func clientClosed(c echo.Context, err error) bool {
	return errors.Is(c.Request().Context().Err(), context.Canceled) && errors.Is(err, context.Canceled)
}
