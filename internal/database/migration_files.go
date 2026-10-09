package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
)

var (
	migrationName   = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	migrationPrefix = regexp.MustCompile(`^(\d+)_.+\.up\.sql$`)
)

// CreateMigration writes an empty up/down pair in directory, numbered after
// the highest existing migration, and returns the two paths.
func CreateMigration(directory, name string) (up, down string, err error) {
	if !migrationName.MatchString(name) {
		return "", "", fmt.Errorf("database: migration name must be snake_case, got %q", name)
	}

	next, err := nextMigrationNumber(directory)
	if err != nil {
		return "", "", err
	}

	base := filepath.Join(directory, fmt.Sprintf("%06d_%s", next, name))
	up, down = base+".up.sql", base+".down.sql"

	if err := createEmptyFiles(up, down); err != nil {
		return "", "", err
	}

	return up, down, nil
}

func nextMigrationNumber(directory string) (int, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0, fmt.Errorf("database: read migrations directory: %w", err)
	}

	numbers := []int{0}
	for _, entry := range entries {
		if match := migrationPrefix.FindStringSubmatch(entry.Name()); match != nil {
			number, _ := strconv.Atoi(match[1]) // the pattern guarantees digits
			numbers = append(numbers, number)
		}
	}

	return slices.Max(numbers) + 1, nil
}

func createEmptyFiles(paths ...string) error {
	for _, path := range paths {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("database: create migration: %w", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("database: create migration: %w", err)
		}
	}

	return nil
}
