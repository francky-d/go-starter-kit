package config_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/config"
)

// knownVariables lists every variable the configuration reads, so tests
// start from a clean environment whatever the developer's shell exports.
var knownVariables = []string{
	"APP_NAME", "APP_ENV",
	"HTTP_PORT", "HTTP_READ_HEADER_TIMEOUT", "HTTP_READ_TIMEOUT",
	"HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT", "HTTP_SHUTDOWN_TIMEOUT",
	"HTTP_REQUEST_TIMEOUT", "HTTP_BODY_LIMIT", "HTTP_TRUST_PROXY",
	"CORS_ALLOWED_ORIGINS",
	"LOG_LEVEL", "LOG_FORMAT",
	"DATABASE_URL", "DB_CONNECT_TIMEOUT", "DB_MAX_OPEN_CONNECTIONS", "DB_MAX_IDLE_CONNECTIONS",
	"DB_CONNECTION_MAX_LIFETIME", "DB_CONNECTION_MAX_IDLE_TIME", "DB_SLOW_QUERY_THRESHOLD",
	"DB_AUTO_MIGRATE",
	"BCRYPT_COST", "JWT_SECRET", "JWT_ISSUER", "ACCESS_TOKEN_LIFETIME", "REFRESH_TOKEN_LIFETIME",
	"AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE", "AUTH_REFRESH_RATE_LIMIT_PER_MINUTE",
}

const (
	databaseURL = "postgres://app:app@localhost:5432/app?sslmode=disable"
	jwtSecret   = "a-test-secret-that-is-at-least-32-bytes-long"
)

// clearEnvironment unsets every known variable, then sets the ones without
// a default so each test starts from a valid configuration.
func clearEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range knownVariables {
		t.Setenv(name, "") // registers the original value for restoration
		require.NoError(t, os.Unsetenv(name))
	}
	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("JWT_SECRET", jwtSecret)
}

func TestFromEnvironment_AppliesDefaults(t *testing.T) {
	clearEnvironment(t)

	cfg, err := config.FromEnvironment()

	require.NoError(t, err)
	assert.Equal(t, "go-starter-kit", cfg.App.Name)
	assert.Equal(t, config.EnvironmentDevelopment, cfg.App.Environment)
	assert.Equal(t, 8080, cfg.HTTP.Port)
	assert.Equal(t, 5*time.Second, cfg.HTTP.ReadHeaderTimeout)
	assert.Equal(t, 10*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.HTTP.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.HTTP.IdleTimeout)
	assert.Equal(t, 10*time.Second, cfg.HTTP.ShutdownTimeout)
	assert.Equal(t, 15*time.Second, cfg.HTTP.RequestTimeout)
	assert.Equal(t, "1M", cfg.HTTP.BodyLimit)
	assert.False(t, cfg.HTTP.TrustProxy)
	assert.Empty(t, cfg.CORS.AllowedOrigins)
	assert.Equal(t, slog.LevelInfo, cfg.Log.Level)
	assert.Equal(t, config.LogFormatJSON, cfg.Log.Format)
	assert.Equal(t, databaseURL, cfg.Database.URL)
	assert.Equal(t, 5*time.Second, cfg.Database.ConnectTimeout)
	assert.Equal(t, 25, cfg.Database.MaxOpenConnections)
	assert.Equal(t, 25, cfg.Database.MaxIdleConnections)
	assert.Equal(t, 30*time.Minute, cfg.Database.ConnectionMaxLifetime)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnectionMaxIdleTime)
	assert.Equal(t, 200*time.Millisecond, cfg.Database.SlowQueryThreshold)
	assert.False(t, cfg.Database.AutoMigrate)
	assert.Equal(t, 12, cfg.Auth.BcryptCost)
	assert.Equal(t, jwtSecret, cfg.Auth.JWTSecret)
	assert.Equal(t, "go-starter-kit", cfg.Auth.JWTIssuer)
	assert.Equal(t, 15*time.Minute, cfg.Auth.AccessTokenLifetime)
	assert.Equal(t, 30*24*time.Hour, cfg.Auth.RefreshTokenLifetime)
	assert.Equal(t, 10, cfg.Auth.CredentialsRateLimitPerMinute)
	assert.Equal(t, 60, cfg.Auth.RefreshRateLimitPerMinute)
}

func TestFromEnvironment_RequiresDatabaseURL(t *testing.T) {
	clearEnvironment(t)
	require.NoError(t, os.Unsetenv("DATABASE_URL"))

	_, err := config.FromEnvironment()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestFromEnvironment_RequiresAJWTSecretWithoutQuotingIt(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("JWT_SECRET", "too-short-secret")

	_, err := config.FromEnvironment()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 bytes long")
	assert.NotContains(t, err.Error(), "too-short-secret")
}

func TestFromEnvironment_ReadsVariables(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("APP_NAME", "billing-api")
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("HTTP_WRITE_TIMEOUT", "45s")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("DB_AUTO_MIGRATE", "true")
	t.Setenv("HTTP_TRUST_PROXY", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, http://localhost:3000")

	cfg, err := config.FromEnvironment()

	require.NoError(t, err)
	assert.Equal(t, "billing-api", cfg.App.Name)
	assert.True(t, cfg.App.IsProduction())
	assert.Equal(t, 9000, cfg.HTTP.Port)
	assert.Equal(t, 45*time.Second, cfg.HTTP.WriteTimeout)
	assert.Equal(t, slog.LevelDebug, cfg.Log.Level)
	assert.Equal(t, config.LogFormatText, cfg.Log.Format)
	assert.True(t, cfg.Database.AutoMigrate)
	assert.True(t, cfg.HTTP.TrustProxy)
	assert.Equal(t, []string{"https://app.example.com", "http://localhost:3000"}, cfg.CORS.AllowedOrigins)
}

func TestFromEnvironment_RejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name     string
		variable string
		value    string
		mentions string // the variable name, or the bad value for parse errors
	}{
		{"unknown environment", "APP_ENV", "staging", "APP_ENV"},
		{"port out of range", "HTTP_PORT", "70000", "HTTP_PORT"},
		{"port not a number", "HTTP_PORT", "http", `"http"`},
		{"negative timeout", "HTTP_READ_TIMEOUT", "-1s", "HTTP_READ_TIMEOUT"},
		{"unknown log level", "LOG_LEVEL", "verbose", `"verbose"`},
		{"unknown log format", "LOG_FORMAT", "xml", "LOG_FORMAT"},
		{"no open connections", "DB_MAX_OPEN_CONNECTIONS", "0", "DB_MAX_OPEN_CONNECTIONS"},
		{"more idle than open connections", "DB_MAX_IDLE_CONNECTIONS", "26", "DB_MAX_IDLE_CONNECTIONS"},
		{"zero connect timeout", "DB_CONNECT_TIMEOUT", "0s", "DB_CONNECT_TIMEOUT"},
		{"request timeout not below write timeout", "HTTP_REQUEST_TIMEOUT", "30s", "HTTP_WRITE_TIMEOUT"},
		{"body limit without size", "HTTP_BODY_LIMIT", "lots", "HTTP_BODY_LIMIT"},
		{"origin without scheme", "CORS_ALLOWED_ORIGINS", "app.example.com", "CORS_ALLOWED_ORIGINS"},
		{"origin with a path", "CORS_ALLOWED_ORIGINS", "https://app.example.com/login", "CORS_ALLOWED_ORIGINS"},
		{"bcrypt cost too low", "BCRYPT_COST", "3", "BCRYPT_COST"},
		{"bcrypt cost too high", "BCRYPT_COST", "17", "BCRYPT_COST"},
		{"missing JWT secret", "JWT_SECRET", "", "JWT_SECRET"},
		{"blank JWT issuer", "JWT_ISSUER", " ", "JWT_ISSUER"},
		{"zero access token lifetime", "ACCESS_TOKEN_LIFETIME", "0s", "ACCESS_TOKEN_LIFETIME"},
		{"refresh token outlived by access token", "REFRESH_TOKEN_LIFETIME", "10m", "REFRESH_TOKEN_LIFETIME"},
		{"no login allowed", "AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE", "0", "AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE"},
		{"no refresh allowed", "AUTH_REFRESH_RATE_LIMIT_PER_MINUTE", "0", "AUTH_REFRESH_RATE_LIMIT_PER_MINUTE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnvironment(t)
			t.Setenv(tc.variable, tc.value)

			_, err := config.FromEnvironment()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.mentions)
		})
	}
}

func TestValidate_ReportsEveryProblemAtOnce(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_PORT", "0")

	_, err := config.FromEnvironment()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_ENV")
	assert.Contains(t, err.Error(), "HTTP_PORT")
}

func TestLoad_ReadsDotEnvFileFromWorkingDirectory(t *testing.T) {
	clearEnvironment(t)
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile(".env", []byte("APP_NAME=from-dotenv\n"), 0o600))

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "from-dotenv", cfg.App.Name)
}

func TestLoad_EnvironmentWinsOverDotEnvFile(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("APP_NAME", "from-environment")
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile(".env", []byte("APP_NAME=from-dotenv\n"), 0o600))

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "from-environment", cfg.App.Name)
}

func TestLoad_WorksWithoutDotEnvFile(t *testing.T) {
	clearEnvironment(t)
	t.Chdir(t.TempDir())

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "go-starter-kit", cfg.App.Name)
}

func TestLoadDatabase_NeedsOnlyTheDatabaseVariables(t *testing.T) {
	clearEnvironment(t)
	require.NoError(t, os.Unsetenv("JWT_SECRET"))
	t.Chdir(t.TempDir())

	database, err := config.LoadDatabase()

	require.NoError(t, err)
	assert.Equal(t, databaseURL, database.URL)
	assert.Equal(t, 5*time.Second, database.ConnectTimeout)
}

func TestLoadDatabase_RejectsInvalidDatabaseSettings(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("DB_MAX_OPEN_CONNECTIONS", "0")
	t.Chdir(t.TempDir())

	_, err := config.LoadDatabase()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_MAX_OPEN_CONNECTIONS")
}
