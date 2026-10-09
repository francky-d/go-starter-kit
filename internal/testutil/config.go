package testutil

import (
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

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
		Auth: config.Auth{
			BcryptCost:           bcrypt.MinCost,
			JWTSecret:            "a-test-secret-that-is-at-least-32-bytes-long",
			JWTIssuer:            "go-starter-kit",
			AccessTokenLifetime:  15 * time.Minute,
			RefreshTokenLifetime: 24 * time.Hour,
			// High enough that only the rate limit tests reach them.
			CredentialsRateLimitPerMinute: 10000,
			RefreshRateLimitPerMinute:     10000,
		},
	}
}
