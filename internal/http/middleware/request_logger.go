package middleware

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// RequestLogger writes one record per request. It logs the path without the
// query string, which may carry tokens or personal data. Successful requests
// to quietRoutes, such as health probes polled every few seconds, are logged
// at debug level only.
func RequestLogger(log *slog.Logger, quietRoutes ...string) echo.MiddlewareFunc {
	return echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		// Runs the error handler first, so the logged status is the final one.
		HandleError:  true,
		LogMethod:    true,
		LogURIPath:   true,
		LogRoutePath: true,
		LogStatus:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogRequestID: true,
		LogValuesFunc: func(c echo.Context, request echomiddleware.RequestLoggerValues) error {
			log.LogAttrs(c.Request().Context(), requestLevel(request, quietRoutes), "request",
				slog.String("method", request.Method),
				slog.String("path", request.URIPath),
				slog.String("route", request.RoutePath),
				slog.Int("status", request.Status),
				slog.Duration("latency", request.Latency),
				slog.String("remote_ip", request.RemoteIP),
				slog.String("request_id", request.RequestID),
			)

			return nil
		},
	})
}

func requestLevel(request echomiddleware.RequestLoggerValues, quietRoutes []string) slog.Level {
	if request.Status < http.StatusBadRequest && slices.Contains(quietRoutes, request.RoutePath) {
		return slog.LevelDebug
	}

	return slog.LevelInfo
}
