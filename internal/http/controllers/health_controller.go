package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/respond"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

type healthStatus struct {
	Status string `json:"status"`
}

func (hc *HealthController) Show(c echo.Context) error {
	return respond.OK(c, healthStatus{Status: "ok"})
}
