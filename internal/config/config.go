// Package config is the only package allowed to read the environment. Every
// other package receives the values it needs through its constructor.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const (
	EnvironmentDevelopment = "development"
	EnvironmentTest        = "test"
	EnvironmentProduction  = "production"
)

const (
	LogFormatJSON = "json"
	LogFormatText = "text"
)

type Config struct {
	App      App  `envPrefix:"APP_"`
	HTTP     HTTP `envPrefix:"HTTP_"`
	Log      Log  `envPrefix:"LOG_"`
	Database Database
}

type App struct {
	Name        string `env:"NAME" envDefault:"go-starter-kit"`
	Environment string `env:"ENV" envDefault:"development"`
}

type HTTP struct {
	Port              int           `env:"PORT" envDefault:"8080"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type Log struct {
	Level  slog.Level `env:"LEVEL" envDefault:"info"`
	Format string     `env:"FORMAT" envDefault:"json"`
}

type Database struct {
	URL                   string        `env:"DATABASE_URL,required"`
	ConnectTimeout        time.Duration `env:"DB_CONNECT_TIMEOUT" envDefault:"5s"`
	MaxOpenConnections    int           `env:"DB_MAX_OPEN_CONNECTIONS" envDefault:"25"`
	MaxIdleConnections    int           `env:"DB_MAX_IDLE_CONNECTIONS" envDefault:"25"`
	ConnectionMaxLifetime time.Duration `env:"DB_CONNECTION_MAX_LIFETIME" envDefault:"30m"`
	ConnectionMaxIdleTime time.Duration `env:"DB_CONNECTION_MAX_IDLE_TIME" envDefault:"5m"`
	SlowQueryThreshold    time.Duration `env:"DB_SLOW_QUERY_THRESHOLD" envDefault:"200ms"`
	AutoMigrate           bool          `env:"DB_AUTO_MIGRATE" envDefault:"false"`
}

// Load reads the optional .env file, then the environment. A variable
// already set in the environment always wins over the .env file.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("config: read .env file: %w", err)
	}

	return FromEnvironment()
}

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
// deployment is fixed in a single pass.
func (c Config) Validate() error {
	problems := errors.Join(
		c.App.validate(),
		c.HTTP.validate(),
		c.Log.validate(),
		c.Database.validate(),
	)
	if problems != nil {
		return fmt.Errorf("config: invalid configuration: %w", problems)
	}

	return nil
}

func (a App) IsProduction() bool {
	return a.Environment == EnvironmentProduction
}

func (a App) validate() error {
	return mustBeOneOf("APP_ENV", a.Environment, EnvironmentDevelopment, EnvironmentTest, EnvironmentProduction)
}

func (h HTTP) validate() error {
	return errors.Join(
		mustBeBetween("HTTP_PORT", h.Port, 1, 65535),
		mustBePositive("HTTP_READ_HEADER_TIMEOUT", h.ReadHeaderTimeout),
		mustBePositive("HTTP_READ_TIMEOUT", h.ReadTimeout),
		mustBePositive("HTTP_WRITE_TIMEOUT", h.WriteTimeout),
		mustBePositive("HTTP_IDLE_TIMEOUT", h.IdleTimeout),
		mustBePositive("HTTP_SHUTDOWN_TIMEOUT", h.ShutdownTimeout),
	)
}

func (l Log) validate() error {
	return mustBeOneOf("LOG_FORMAT", l.Format, LogFormatJSON, LogFormatText)
}

func (d Database) validate() error {
	return errors.Join(
		mustBeAtLeast("DB_MAX_OPEN_CONNECTIONS", d.MaxOpenConnections, 1),
		mustBeBetween("DB_MAX_IDLE_CONNECTIONS", d.MaxIdleConnections, 0, d.MaxOpenConnections),
		mustBePositive("DB_CONNECT_TIMEOUT", d.ConnectTimeout),
		mustBePositive("DB_CONNECTION_MAX_LIFETIME", d.ConnectionMaxLifetime),
		mustBePositive("DB_CONNECTION_MAX_IDLE_TIME", d.ConnectionMaxIdleTime),
		mustBePositive("DB_SLOW_QUERY_THRESHOLD", d.SlowQueryThreshold),
	)
}
