package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/francky-d/go-starter-kit/internal/config"
	"github.com/francky-d/go-starter-kit/internal/database"
)

const usage = `Usage: migrate <command>

Commands:
  up                apply every pending migration
  down [steps]      roll back the last migration, or the last <steps> migrations
  version           print the current version
  force <version>   mark <version> as applied, after fixing a failed migration by hand
  create <name>     create an empty up/down pair in database/migrations`

const migrationsDirectory = "database/migrations"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("missing command\n\n%s", usage)
	}

	command, parameters := arguments[0], arguments[1:]
	if command == "create" {
		return create(parameters)
	}

	return migrate(command, parameters)
}

func create(parameters []string) error {
	if len(parameters) != 1 {
		return errors.New("usage: migrate create <name>")
	}

	up, down, err := database.CreateMigration(migrationsDirectory, parameters[0])
	if err != nil {
		return err
	}

	fmt.Printf("Created %s\nCreated %s\n", up, down)

	return nil
}

func migrate(command string, parameters []string) error {
	settings, err := config.LoadDatabase()
	if err != nil {
		return err
	}

	migrator, err := database.NewMigrator(context.Background(), settings)
	if err != nil {
		return err
	}

	return errors.Join(execute(migrator, command, parameters), migrator.Close())
}

func execute(migrator *database.Migrator, command string, parameters []string) error {
	switch command {
	case "up":
		return migrator.Up()
	case "down":
		return down(migrator, parameters)
	case "version":
		return printVersion(migrator)
	case "force":
		return force(migrator, parameters)
	default:
		return fmt.Errorf("unknown command %q\n\n%s", command, usage)
	}
}

func down(migrator *database.Migrator, parameters []string) error {
	steps, err := optionalNumber(parameters, 1)
	if err != nil {
		return err
	}

	return migrator.Down(steps)
}

func force(migrator *database.Migrator, parameters []string) error {
	if len(parameters) != 1 {
		return errors.New("usage: migrate force <version>")
	}

	version, err := strconv.Atoi(parameters[0])
	if err != nil {
		return fmt.Errorf("version must be a number, got %q", parameters[0])
	}

	return migrator.Force(version)
}

func printVersion(migrator *database.Migrator) error {
	version, dirty, err := migrator.Version()
	if err != nil {
		return err
	}

	if dirty {
		fmt.Printf("%d (dirty: fix the failed migration, then run force)\n", version)
		return nil
	}

	fmt.Println(version)

	return nil
}

func optionalNumber(parameters []string, fallback int) (int, error) {
	if len(parameters) == 0 {
		return fallback, nil
	}

	number, err := strconv.Atoi(parameters[0])
	if err != nil {
		return 0, fmt.Errorf("expected a number, got %q", parameters[0])
	}

	return number, nil
}
