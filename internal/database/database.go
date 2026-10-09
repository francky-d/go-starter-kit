// Package database opens the PostgreSQL connection pool, used through GORM.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/francky-d/go-starter-kit/internal/config"
)

// Connect fails within the connect timeout when the database is unreachable,
// so a misconfigured deployment fails at startup, not on the first request.
func Connect(ctx context.Context, cfg config.Database, log *slog.Logger) (*gorm.DB, error) {
	db, err := open(cfg, log)
	if err != nil {
		return nil, err
	}

	pool, err := db.DB()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("database: %w", err), Close(db))
	}

	configurePool(pool, cfg)

	if err := ping(ctx, pool, cfg); err != nil {
		return nil, errors.Join(err, pool.Close())
	}

	return db, nil
}

func Close(db *gorm.DB) error {
	pool, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: close: %w", err)
	}

	return pool.Close()
}

func open(cfg config.Database, log *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.URL), &gorm.Config{
		TranslateError:       true,
		DisableAutomaticPing: true,
		Logger:               queryLogger(cfg, log),
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	return db, nil
}

// queryLogger reports slow and failed queries. Parameterized queries keep
// bound values (emails, password hashes, tokens) out of the logs.
func queryLogger(cfg config.Database, log *slog.Logger) logger.Interface {
	return logger.NewSlogLogger(log, logger.Config{
		SlowThreshold:             cfg.SlowQueryThreshold,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		LogLevel:                  logger.Warn,
	})
}

func configurePool(pool *sql.DB, cfg config.Database) {
	pool.SetMaxOpenConns(cfg.MaxOpenConnections)
	pool.SetMaxIdleConns(cfg.MaxIdleConnections)
	pool.SetConnMaxLifetime(cfg.ConnectionMaxLifetime)
	pool.SetConnMaxIdleTime(cfg.ConnectionMaxIdleTime)
}

func ping(ctx context.Context, pool *sql.DB, cfg config.Database) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := pool.PingContext(ctx); err != nil {
		return fmt.Errorf("database: connect: %w", err)
	}

	return nil
}
