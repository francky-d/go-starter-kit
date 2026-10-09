package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/requests"
	"github.com/francky-d/go-starter-kit/internal/http/resources"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/services"
)

type AuthController struct {
	auth *services.AuthService
}

func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

func (ac *AuthController) Register(c echo.Context) error {
	var input requests.Register
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	session, err := ac.auth.Register(c.Request().Context(), services.Registration{
		Name:     input.Name,
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return err
	}

	return respond.Created(c, resources.NewSession(session))
}

func (ac *AuthController) Login(c echo.Context) error {
	var input requests.Login
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	session, err := ac.auth.Login(c.Request().Context(), input.Email, input.Password)
	if err != nil {
		return err
	}

	return respond.OK(c, resources.NewSession(session))
}

func (ac *AuthController) Refresh(c echo.Context) error {
	var input requests.RefreshToken
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	session, err := ac.auth.Refresh(c.Request().Context(), input.RefreshToken)
	if err != nil {
		return err
	}

	return respond.OK(c, resources.NewSession(session))
}

func (ac *AuthController) Logout(c echo.Context) error {
	var input requests.RefreshToken
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	if err := ac.auth.Logout(c.Request().Context(), input.RefreshToken); err != nil {
		return err
	}

	return respond.NoContent(c)
}
