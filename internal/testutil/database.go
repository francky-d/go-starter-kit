// Package testutil provides helpers for tests that need the real database or
// the real application. It is the only package besides config that reads the
// environment, for TEST_DATABASE_URL.
package testutil

import (
	"os"
	"time"

	"github.com/francky-d/go-starter-kit/internal/config"
)

// defaultDatabaseURL points to the test database created by `make docker-up`.
const defaultDatabaseURL = "postgres://app:app@localhost:5432/app_test?sslmode=disable"

func DatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}

	return defaultDatabaseURL
}

func DatabaseConfig() config.Database {
	return config.Database{
		URL:                   DatabaseURL(),
		ConnectTimeout:        5 * time.Second,
		MaxOpenConnections:    10,
		MaxIdleConnections:    10,
		ConnectionMaxLifetime: time.Minute,
		ConnectionMaxIdleTime: time.Minute,
		SlowQueryThreshold:    time.Second,
	}
}
