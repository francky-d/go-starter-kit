package middleware

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// Recover turns a panic into a generic 500 and logs the stack trace, which
// never reaches the client.
func Recover(log *slog.Logger) echo.MiddlewareFunc {
	return echomiddleware.RecoverWithConfig(echomiddleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			log.ErrorContext(c.Request().Context(), "panic recovered",
				"error", err,
				"stack", string(stack),
			)

			return err
		},
	})
}
