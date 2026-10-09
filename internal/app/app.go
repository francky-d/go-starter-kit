// Package app wires every dependency together and runs the HTTP server.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/config"
	"github.com/francky-d/go-starter-kit/internal/database"
	"github.com/francky-d/go-starter-kit/internal/http/controllers"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/routes"
)

type App struct {
	config   config.Config
	log      *slog.Logger
	database *gorm.DB
	echo     *echo.Echo
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	db, err := connectDatabase(ctx, cfg.Database, log)
	if err != nil {
		return nil, err
	}

	application := &App{config: cfg, log: log, database: db, echo: newEcho(log)}
	if err := application.registerRoutes(); err != nil {
		return nil, errors.Join(err, application.Close())
	}

	return application, nil
}

// connectDatabase connects before migrating, so an unreachable database
// fails within the connect timeout instead of hanging in the migrator.
func connectDatabase(ctx context.Context, cfg config.Database, log *slog.Logger) (*gorm.DB, error) {
	db, err := database.Connect(ctx, cfg, log)
	if err != nil || !cfg.AutoMigrate {
		return db, err
	}

	if err := database.ApplyMigrations(ctx, cfg); err != nil {
		return nil, errors.Join(err, database.Close(db))
	}

	log.Info("database migrations applied")

	return db, nil
}

func (a *App) Close() error {
	return database.Close(a.database)
}

func (a *App) registerRoutes() error {
	pool, err := a.database.DB()
	if err != nil {
		return fmt.Errorf("app: %w", err)
	}

	routes.Register(a.echo, routes.Dependencies{
		Health: controllers.NewHealthController(pool),
	})

	return nil
}

func newEcho(log *slog.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = respond.ErrorHandler(log)

	return e
}

func (a *App) Handler() http.Handler {
	return a.echo
}

func (a *App) Run(ctx context.Context) error {
	address := fmt.Sprintf(":%d", a.config.HTTP.Port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("app: listen on %s: %w", address, err)
	}

	return a.Serve(ctx, listener)
}

// Serve blocks until ctx is cancelled, then shuts the server down gracefully.
func (a *App) Serve(ctx context.Context, listener net.Listener) error {
	server := a.newServer()
	serving := startServing(server, listener)
	a.log.Info("server started", "address", listener.Addr().String())

	select {
	case err := <-serving:
		return fmt.Errorf("app: serve: %w", err)
	case <-ctx.Done():
		return a.shutDown(ctx, server, serving)
	}
}

func (a *App) newServer() *http.Server {
	return &http.Server{
		Handler:           a.echo,
		ReadHeaderTimeout: a.config.HTTP.ReadHeaderTimeout,
		ReadTimeout:       a.config.HTTP.ReadTimeout,
		WriteTimeout:      a.config.HTTP.WriteTimeout,
		IdleTimeout:       a.config.HTTP.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(a.log.Handler(), slog.LevelWarn),
	}
}

func startServing(server *http.Server, listener net.Listener) <-chan error {
	serving := make(chan error, 1)
	go func() { serving <- server.Serve(listener) }()

	return serving
}

// shutDown lets in-flight requests finish within the shutdown timeout, then
// closes whatever is still open. ctx is already cancelled at this point,
// hence the timeout is derived from context.WithoutCancel(ctx).
func (a *App) shutDown(ctx context.Context, server *http.Server, serving <-chan error) error {
	a.log.Info("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		a.log.Warn("graceful shutdown timed out, closing remaining connections", "error", err)
		return errors.Join(fmt.Errorf("app: shutdown: %w", err), server.Close())
	}

	if err := <-serving; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("app: serve: %w", err)
	}

	a.log.Info("server stopped")

	return nil
}
