// Package router lets routes be declared as a readable chain:
//
//	api.Group("/auth").Middleware(rateLimit).
//		Post("/register", auth.Register).
//		Post("/login", auth.Login)
//
// Each route is registered on Echo with its full path and its middleware.
// Echo groups are not used: they register a catch-all route per group, so an
// unknown URL under a protected prefix would run that group's middleware.
package router

import (
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"
)

type Router struct {
	echo       *echo.Echo
	prefix     string
	middleware []echo.MiddlewareFunc
}

func New(e *echo.Echo) *Router {
	return &Router{echo: e}
}

// Group returns a new router; the receiver is left unchanged, so the order
// in which groups are declared never matters.
func (r *Router) Group(prefix string, middleware ...echo.MiddlewareFunc) *Router {
	return &Router{
		echo:       r.echo,
		prefix:     r.prefix + prefix,
		middleware: slices.Concat(r.middleware, middleware),
	}
}

func (r *Router) Middleware(middleware ...echo.MiddlewareFunc) *Router {
	return r.Group("", middleware...)
}

func (r *Router) Get(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *Router {
	return r.add(http.MethodGet, path, handler, middleware)
}

func (r *Router) Post(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *Router {
	return r.add(http.MethodPost, path, handler, middleware)
}

func (r *Router) Put(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *Router {
	return r.add(http.MethodPut, path, handler, middleware)
}

func (r *Router) Patch(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *Router {
	return r.add(http.MethodPatch, path, handler, middleware)
}

func (r *Router) Delete(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *Router {
	return r.add(http.MethodDelete, path, handler, middleware)
}

func (r *Router) add(method, path string, handler echo.HandlerFunc, middleware []echo.MiddlewareFunc) *Router {
	r.echo.Add(method, r.prefix+path, handler, slices.Concat(r.middleware, middleware)...)

	return r
}
