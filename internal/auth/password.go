// Package auth holds the authentication primitives: password hashing and
// tokens. It knows nothing about HTTP or the database.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MaxPasswordBytes is bcrypt's limit: it ignores anything beyond 72 bytes,
// so longer passwords are rejected instead of being silently truncated.
const MaxPasswordBytes = 72

var (
	ErrPasswordTooLong  = errors.New("auth: password is longer than 72 bytes")
	ErrPasswordMismatch = errors.New("auth: password does not match")
)

type PasswordHasher struct {
	cost      int
	dummyHash []byte
}

func NewPasswordHasher(cost int) (*PasswordHasher, error) {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("dummy password"), cost)
	if err != nil {
		return nil, fmt.Errorf("auth: prepare password hasher: %w", err)
	}

	return &PasswordHasher{cost: cost, dummyHash: dummyHash}, nil
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}

	return string(hash), nil
}

// Verify rejects a password over 72 bytes, which bcrypt would otherwise
// compare on its first 72 bytes only. It still spends the bcrypt time, so the
// rejection takes as long as for an unknown email.
func (h *PasswordHasher) Verify(hash, password string) error {
	if len(password) > MaxPasswordBytes {
		h.SimulateVerify(password)
		return ErrPasswordMismatch
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordMismatch
	}

	return err
}

// SimulateVerify spends the time of a real verification. Login calls it when
// the email is unknown, so response times do not reveal which emails exist.
func (h *PasswordHasher) SimulateVerify(password string) {
	_ = bcrypt.CompareHashAndPassword(h.dummyHash, []byte(password))
}
