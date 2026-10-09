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
	CORS     CORS `envPrefix:"CORS_"`
	Log      Log  `envPrefix:"LOG_"`
	Database Database
	Auth     Auth
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
	RequestTimeout    time.Duration `env:"REQUEST_TIMEOUT" envDefault:"15s"`
	BodyLimit         string        `env:"BODY_LIMIT" envDefault:"1M"`
	// TrustProxy reads the client IP from X-Forwarded-For. Enable it only
	// behind a proxy that sets the header, or clients can forge their IP.
	TrustProxy bool `env:"TRUST_PROXY" envDefault:"false"`
}

type CORS struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
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

type Auth struct {
	BcryptCost           int           `env:"BCRYPT_COST" envDefault:"12"`
	JWTSecret            string        `env:"JWT_SECRET"`
	JWTIssuer            string        `env:"JWT_ISSUER" envDefault:"go-starter-kit"`
	AccessTokenLifetime  time.Duration `env:"ACCESS_TOKEN_LIFETIME" envDefault:"15m"`
	RefreshTokenLifetime time.Duration `env:"REFRESH_TOKEN_LIFETIME" envDefault:"720h"`
	// Register and login take credentials an attacker can guess, so they get
	// a tight limit; refresh and logout are routine traffic and get their own.
	CredentialsRateLimitPerMinute int `env:"AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE" envDefault:"10"`
	RefreshRateLimitPerMinute     int `env:"AUTH_REFRESH_RATE_LIMIT_PER_MINUTE" envDefault:"60"`
}

// MinJWTSecretBytes matches the HS256 key size: a shorter secret weakens the
// signature.
const MinJWTSecretBytes = 32

// Load reads the optional .env file, then the environment. A variable
// already set in the environment always wins over the .env file.
func Load() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	return FromEnvironment()
}

// LoadDatabase reads the database settings only, for tools such as the
// migrate command that need neither the HTTP settings nor the secrets.
func LoadDatabase() (Database, error) {
	if err := loadDotEnv(); err != nil {
		return Database{}, err
	}

	database, err := env.ParseAs[Database]()
	if err != nil {
		return Database{}, fmt.Errorf("config: %w", err)
	}

	if err := database.validate(); err != nil {
		return Database{}, fmt.Errorf("config: invalid configuration: %w", err)
	}

	return database, nil
}

func loadDotEnv() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("config: read .env file: %w", err)
	}

	return nil
}

func FromEnvironment() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	cfg.CORS.AllowedOrigins = trimEach(cfg.CORS.AllowedOrigins)

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
		c.CORS.validate(),
		c.Log.validate(),
		c.Database.validate(),
		c.Auth.validate(),
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
		mustBePositive("HTTP_REQUEST_TIMEOUT", h.RequestTimeout),
		mustExceed("HTTP_WRITE_TIMEOUT", h.WriteTimeout, "HTTP_REQUEST_TIMEOUT", h.RequestTimeout),
		mustBeByteSize("HTTP_BODY_LIMIT", h.BodyLimit),
	)
}

func (c CORS) validate() error {
	return mustBeOrigins("CORS_ALLOWED_ORIGINS", c.AllowedOrigins)
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

// validate caps BCRYPT_COST at 16: bcrypt accepts up to 31, but each step
// doubles the time, and the hasher computes one hash at startup. Tests use
// the minimum of 4 to stay fast.
func (a Auth) validate() error {
	return errors.Join(
		mustBeBetween("BCRYPT_COST", a.BcryptCost, 4, 16),
		mustBeLongSecret("JWT_SECRET", a.JWTSecret, MinJWTSecretBytes),
		mustNotBeBlank("JWT_ISSUER", a.JWTIssuer),
		mustBePositive("ACCESS_TOKEN_LIFETIME", a.AccessTokenLifetime),
		mustExceed("REFRESH_TOKEN_LIFETIME", a.RefreshTokenLifetime, "ACCESS_TOKEN_LIFETIME", a.AccessTokenLifetime),
		mustBeAtLeast("AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE", a.CredentialsRateLimitPerMinute, 1),
		mustBeAtLeast("AUTH_REFRESH_RATE_LIMIT_PER_MINUTE", a.RefreshRateLimitPerMinute, 1),
	)
}
