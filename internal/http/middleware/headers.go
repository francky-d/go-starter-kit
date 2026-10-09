package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/francky-d/go-starter-kit/internal/config"
)

// SecureHeaders sets the headers that matter for a JSON API. The legacy
// X-XSS-Protection filter is disabled ("0"), as current browsers recommend.
func SecureHeaders() echo.MiddlewareFunc {
	return echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
		XSSProtection:         "0",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'",
		ReferrerPolicy:        "no-referrer",
	})
}

// CORS allows the configured browser origins to call the API. Credentials
// are not allowed: clients authenticate with a bearer token, not cookies.
func CORS(cfg config.CORS) echo.MiddlewareFunc {
	return echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:  cfg.AllowedOrigins,
		AllowMethods:  []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowHeaders:  []string{echo.HeaderAuthorization, echo.HeaderContentType},
		ExposeHeaders: []string{echo.HeaderXRequestID},
		MaxAge:        600,
	})
}
