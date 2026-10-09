// Package config loads the application configuration from environment
// variables into typed structs. It is the only package allowed to read the
// environment: every other package receives the values it needs through its
// constructor.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Supported application environments.
const (
	EnvironmentDevelopment = "development"
	EnvironmentTest        = "test"
	EnvironmentProduction  = "production"
)

// Supported log output formats.
const (
	LogFormatJSON = "json"
	LogFormatText = "text"
)

// Config is the complete application configuration.
type Config struct {
	App  App  `envPrefix:"APP_"`
	HTTP HTTP `envPrefix:"HTTP_"`
	Log  Log  `envPrefix:"LOG_"`
}

// App describes the running application.
type App struct {
	Name        string `env:"NAME" envDefault:"go-starter-kit"`
	Environment string `env:"ENV" envDefault:"development"`
}

// HTTP configures the HTTP server.
type HTTP struct {
	Port              int           `env:"PORT" envDefault:"8080"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

// Log configures the structured logger.
type Log struct {
	Level  slog.Level `env:"LEVEL" envDefault:"info"`
	Format string     `env:"FORMAT" envDefault:"json"`
}

// Load reads an optional .env file from the working directory, then builds
// the configuration from the environment. Variables already set in the
// environment always win over the .env file.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("config: read .env file: %w", err)
	}

	return FromEnvironment()
}

// FromEnvironment builds and validates the configuration from the process
// environment only, without reading any file.
func FromEnvironment() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate reports every invalid value at once, so a misconfigured
// deployment can be fixed in a single pass.
func (c Config) Validate() error {
	var problems []error

	environments := []string{EnvironmentDevelopment, EnvironmentTest, EnvironmentProduction}
	if !slices.Contains(environments, c.App.Environment) {
		problems = append(problems, fmt.Errorf("APP_ENV must be one of %v, got %q", environments, c.App.Environment))
	}

	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		problems = append(problems, fmt.Errorf("HTTP_PORT must be between 1 and 65535, got %d", c.HTTP.Port))
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{"HTTP_READ_HEADER_TIMEOUT", c.HTTP.ReadHeaderTimeout},
		{"HTTP_READ_TIMEOUT", c.HTTP.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", c.HTTP.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", c.HTTP.IdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", c.HTTP.ShutdownTimeout},
	}
	for _, timeout := range timeouts {
		if timeout.value <= 0 {
			problems = append(problems, fmt.Errorf("%s must be positive, got %s", timeout.name, timeout.value))
		}
	}

	formats := []string{LogFormatJSON, LogFormatText}
	if !slices.Contains(formats, c.Log.Format) {
		problems = append(problems, fmt.Errorf("LOG_FORMAT must be one of %v, got %q", formats, c.Log.Format))
	}

	if len(problems) > 0 {
		return fmt.Errorf("config: invalid configuration: %w", errors.Join(problems...))
	}

	return nil
}

// IsProduction reports whether the application runs in production.
func (a App) IsProduction() bool {
	return a.Environment == EnvironmentProduction
}
