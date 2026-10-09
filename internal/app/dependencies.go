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

	users := repositories.NewUserRepository(a.database)
	accessTokens := auth.NewTokenIssuer(a.config.Auth)

	authService, err := a.authService(users, accessTokens)
	if err != nil {
		return routes.Dependencies{}, err
	}

	return routes.Dependencies{
		Health:               controllers.NewHealthController(pool),
		Auth:                 controllers.NewAuthController(authService),
		Users:                controllers.NewUserController(services.NewUserService(users)),
		Authenticated:        middleware.Authenticated(accessTokens),
		CredentialsRateLimit: middleware.RateLimit(a.config.Auth.CredentialsRateLimitPerMinute),
		RefreshRateLimit:     middleware.RateLimit(a.config.Auth.RefreshRateLimitPerMinute),
	}, nil
}

func (a *App) authService(users *repositories.UserRepository, accessTokens *auth.TokenIssuer) (*services.AuthService, error) {
	passwords, err := auth.NewPasswordHasher(a.config.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}

	return services.NewAuthService(
		a.database,
		users,
		repositories.NewRefreshTokenRepository(a.database),
		passwords,
		accessTokens,
		a.config.Auth.RefreshTokenLifetime,
		a.log,
	), nil
}
