package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/francky-d/go-starter-kit/internal/app"
	"github.com/francky-d/go-starter-kit/internal/config"
	"github.com/francky-d/go-starter-kit/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logging.New(cfg.Log, os.Stdout).With("app", cfg.App.Name, "environment", cfg.App.Environment)

	application, err := app.New(cfg, log)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Restore default signal handling once shutdown starts, so a second
	// Ctrl+C kills the process instead of waiting for the shutdown timeout.
	context.AfterFunc(ctx, stop)

	return application.Run(ctx)
}
