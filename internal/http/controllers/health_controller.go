package controllers

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
)

const databasePingTimeout = 2 * time.Second

type DatabasePinger interface {
	PingContext(ctx context.Context) error
}

type HealthController struct {
	database DatabasePinger
}

func NewHealthController(database DatabasePinger) *HealthController {
	return &HealthController{database: database}
}

type healthStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func (hc *HealthController) Show(c echo.Context) error {
	if err := hc.pingDatabase(c.Request().Context()); err != nil {
		return apperror.ServiceUnavailable().Wrap(err)
	}

	return respond.OK(c, healthStatus{Status: "ok", Database: "ok"})
}

func (hc *HealthController) pingDatabase(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, databasePingTimeout)
	defer cancel()

	return hc.database.PingContext(ctx)
}
