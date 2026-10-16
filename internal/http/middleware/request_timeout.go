package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

// RequestTimeout gives the request context a deadline. Work cut by that
// deadline answers 503, even when a service already wrapped the cause in a
// 500: Echo's default would keep the inner 500. A deadline a handler sets on
// its own call keeps the status the handler chose.
func RequestTimeout(timeout time.Duration) echo.MiddlewareFunc {
	return echomiddleware.ContextTimeoutWithConfig(echomiddleware.ContextTimeoutConfig{
		Timeout:      timeout,
		ErrorHandler: serviceUnavailableAfterDeadline,
	})
}

func serviceUnavailableAfterDeadline(err error, c echo.Context) error {
	requestDeadlinePassed := errors.Is(c.Request().Context().Err(), context.DeadlineExceeded)
	if requestDeadlinePassed && errors.Is(err, context.DeadlineExceeded) {
		return apperror.ServiceUnavailable().Wrap(err)
	}

	return err
}
