package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratedatabase "github.com/golang-migrate/migrate/v4/database"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"github.com/francky-d/go-starter-kit/database/migrations"
	"github.com/francky-d/go-starter-kit/internal/config"
)

// Migrator applies the migrations embedded in the binary. It opens its own
// connection, so closing it never touches the application's pool.
type Migrator struct {
	migrate *migrate.Migrate
}

func NewMigrator(ctx context.Context, cfg config.Database) (*Migrator, error) {
	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("database: read migrations: %w", err)
	}

	target, err := openMigrationTarget(ctx, cfg)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance("embedded", source, "postgres", target)
	if err != nil {
		return nil, fmt.Errorf("database: prepare migrations: %w", err)
	}

	return &Migrator{migrate: m}, nil
}

// openMigrationTarget pings with the connect timeout first: the migration
// driver pings without a deadline and would hang on an unreachable host.
func openMigrationTarget(ctx context.Context, cfg config.Database) (migratedatabase.Driver, error) {
	connection, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("database: open migration connection: %w", err)
	}

	if err := ping(ctx, connection, cfg); err != nil {
		return nil, errors.Join(err, connection.Close())
	}

	target, err := migratepgx.WithInstance(connection, &migratepgx.Config{})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("database: connect for migrations: %w", err), connection.Close())
	}

	return target, nil
}

func ApplyMigrations(ctx context.Context, cfg config.Database) error {
	migrator, err := NewMigrator(ctx, cfg)
	if err != nil {
		return err
	}

	return errors.Join(migrator.Up(), migrator.Close())
}

func (m *Migrator) Up() error {
	return ignoreNoChange(m.migrate.Up())
}

func (m *Migrator) Down(steps int) error {
	if steps < 1 {
		return fmt.Errorf("database: down needs at least 1 step, got %d", steps)
	}

	return ignoreShortLimit(ignoreNoChange(m.migrate.Steps(-steps)))
}

// Version returns 0 when no migration has been applied yet. A dirty version
// means a migration failed halfway and must be fixed, then forced.
func (m *Migrator) Version() (version uint, dirty bool, err error) {
	version, dirty, err = m.migrate.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}

	return version, dirty, err
}

func (m *Migrator) Force(version int) error {
	return m.migrate.Force(version)
}

func (m *Migrator) Close() error {
	sourceErr, databaseErr := m.migrate.Close()

	return errors.Join(sourceErr, databaseErr)
}

// ignoreShortLimit accepts a rollback of more steps than were applied: every
// applied migration has been rolled back, which is what was asked.
func ignoreShortLimit(err error) error {
	var short migrate.ErrShortLimit
	if errors.As(err, &short) {
		return nil
	}

	return err
}

func ignoreNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}

	return err
}
