package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/controllers"
	"github.com/francky-d/go-starter-kit/internal/http/router"
)

type Dependencies struct {
	Health               *controllers.HealthController
	Auth                 *controllers.AuthController
	Users                *controllers.UserController
	Authenticated        echo.MiddlewareFunc
	CredentialsRateLimit echo.MiddlewareFunc
	RefreshRateLimit     echo.MiddlewareFunc
}

func Register(r *router.Router, d Dependencies) {
	r.Get("/health", d.Health.Show)

	api := r.Group("/api/v1")

	authentication := api.Group("/auth")

	authentication.Middleware(d.CredentialsRateLimit).
		Post("/register", d.Auth.Register).
		Post("/login", d.Auth.Login)

	authentication.Middleware(d.RefreshRateLimit).
		Post("/refresh", d.Auth.Refresh).
		Post("/logout", d.Auth.Logout)

	api.Middleware(d.Authenticated).
		Get("/me", d.Users.Me)
}
