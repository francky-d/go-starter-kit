package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/http/resources"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/services"
)

type UserController struct {
	users *services.UserService
}

func NewUserController(users *services.UserService) *UserController {
	return &UserController{users: users}
}

func (uc *UserController) Me(c echo.Context) error {
	user, err := uc.users.CurrentUser(c.Request().Context(), middleware.CurrentUserID(c))
	if err != nil {
		return err
	}

	return respond.OK(c, resources.NewUser(user))
}
