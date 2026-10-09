package testutil

import (
	"log/slog"
	"time"

	"github.com/francky-d/go-starter-kit/internal/config"
)

func Config() config.Config {
	return config.Config{
		App: config.App{Name: "go-starter-kit-test", Environment: config.EnvironmentTest},
		HTTP: config.HTTP{
			Port:              8080,
			ReadHeaderTimeout: time.Second,
			ReadTimeout:       time.Second,
			WriteTimeout:      time.Second,
			IdleTimeout:       time.Second,
			ShutdownTimeout:   time.Second,
			RequestTimeout:    500 * time.Millisecond,
			BodyLimit:         "1M",
		},
		Log:      config.Log{Level: slog.LevelError, Format: config.LogFormatJSON},
		Database: DatabaseConfig(),
	}
}
