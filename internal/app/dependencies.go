package app

import (
	"github.com/francky-d/go-starter-kit/internal/auth"
	"github.com/francky-d/go-starter-kit/internal/http/controllers"
	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/repositories"
	"github.com/francky-d/go-starter-kit/internal/routes"
	"github.com/francky-d/go-starter-kit/internal/services"
)

// dependencies builds every controller and middleware the routes use, with
// explicit constructors: reading this function shows what depends on what.
func (a *App) dependencies() (routes.Dependencies, error) {
	pool, err := a.database.DB()
	if err != nil {
		return routes.Dependencies{}, err
	}

	authService, err := a.authService()
	if err != nil {
		return routes.Dependencies{}, err
	}

	return routes.Dependencies{
		Health:               controllers.NewHealthController(pool),
		Auth:                 controllers.NewAuthController(authService),
		CredentialsRateLimit: middleware.RateLimit(a.config.Auth.CredentialsRateLimitPerMinute),
		RefreshRateLimit:     middleware.RateLimit(a.config.Auth.RefreshRateLimitPerMinute),
	}, nil
}

func (a *App) authService() (*services.AuthService, error) {
	passwords, err := auth.NewPasswordHasher(a.config.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}

	return services.NewAuthService(
		a.database,
		repositories.NewUserRepository(a.database),
		repositories.NewRefreshTokenRepository(a.database),
		passwords,
		auth.NewTokenIssuer(a.config.Auth),
		a.config.Auth.RefreshTokenLifetime,
		a.log,
	), nil
}
