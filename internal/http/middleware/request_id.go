package middleware

import (
	"regexp"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/francky-d/go-starter-kit/internal/logging"
)

var plainRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestID keeps a client-provided X-Request-Id only when it is short and
// plain: it is copied into every log record of the request, so an arbitrary
// value could flood or pollute the logs. Otherwise a new one is generated.
// The id goes into the request context, where the logger finds it.
func RequestID() echo.MiddlewareFunc {
	assignRequestID := echomiddleware.RequestIDWithConfig(echomiddleware.RequestIDConfig{
		RequestIDHandler: func(c echo.Context, requestID string) {
			ctx := logging.ContextWithRequestID(c.Request().Context(), requestID)
			c.SetRequest(c.Request().WithContext(ctx))
		},
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		handle := assignRequestID(next)

		return func(c echo.Context) error {
			if !plainRequestID.MatchString(c.Request().Header.Get(echo.HeaderXRequestID)) {
				c.Request().Header.Del(echo.HeaderXRequestID)
			}

			return handle(c)
		}
	}
}
