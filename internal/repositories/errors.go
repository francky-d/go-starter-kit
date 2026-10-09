// Package repositories holds the database queries. Repositories translate
// GORM errors into the sentinels below, so services never depend on GORM
// error values. Services still use GORM to open transactions, which they
// hand to repositories through WithTx.
package repositories

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("repositories: record not found")
	ErrDuplicate = errors.New("repositories: duplicate record")
	// ErrInvalid reports a value a database constraint rejected, such as an
	// email that is not lowercase.
	ErrInvalid = errors.New("repositories: value rejected by a constraint")
)

func translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return fmt.Errorf("%w: %w", ErrDuplicate, err)
	case errors.Is(err, gorm.ErrCheckConstraintViolated):
		return fmt.Errorf("%w: %w", ErrInvalid, err)
	default:
		return err
	}
}
