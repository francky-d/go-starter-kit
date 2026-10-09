package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/controllers"
)

type Dependencies struct {
	Health *controllers.HealthController
}

func Register(e *echo.Echo, d Dependencies) {
	e.GET("/health", d.Health.Show)
}
