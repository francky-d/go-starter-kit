// Package middleware holds the HTTP middleware of the application.
package middleware

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/francky-d/go-starter-kit/internal/config"
)

// Baseline returns the middleware every request goes through, outermost
// first. The request logger wraps Recover so that a recovered panic is logged
// with its final 500 status. ClientDisconnect runs inside RequestTimeout: it
// reads the request context before RequestTimeout cancels it on return.
func Baseline(cfg config.Config, log *slog.Logger) []echo.MiddlewareFunc {
	stack := []echo.MiddlewareFunc{
		RequestID(),
		RequestLogger(log, "/health"),
		Recover(log),
		SecureHeaders(),
	}

	if len(cfg.CORS.AllowedOrigins) > 0 {
		stack = append(stack, CORS(cfg.CORS))
	}

	return append(stack,
		echomiddleware.BodyLimit(cfg.HTTP.BodyLimit),
		RequestTimeout(cfg.HTTP.RequestTimeout),
		ClientDisconnect(),
	)
}
