package routes

import (
	"github.com/francky-d/go-starter-kit/internal/http/controllers"
	"github.com/francky-d/go-starter-kit/internal/http/router"
)

type Dependencies struct {
	Health *controllers.HealthController
}

func Register(r *router.Router, d Dependencies) {
	r.Get("/health", d.Health.Show)
}
