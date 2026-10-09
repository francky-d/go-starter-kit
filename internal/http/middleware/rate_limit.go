package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

// RateLimit allows each client IP requestsPerMinute requests, with bursts of
// the same size. The counters live in memory: with several instances, each
// one counts on its own. A refused request is told to retry after a minute,
// the time for the bucket to fill up again.
func RateLimit(requestsPerMinute int) echo.MiddlewareFunc {
	store := echomiddleware.NewRateLimiterMemoryStoreWithConfig(echomiddleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(float64(requestsPerMinute) / 60),
		Burst:     requestsPerMinute,
		ExpiresIn: 3 * time.Minute,
	})

	return echomiddleware.RateLimiterWithConfig(echomiddleware.RateLimiterConfig{
		Store: store,
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			c.Response().Header().Set(echo.HeaderRetryAfter, "60")
			return apperror.TooManyRequests()
		},
	})
}
