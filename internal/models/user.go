package models

import "strings"

type User struct {
	Base
	Name         string
	Email        string
	PasswordHash string `json:"-"`
}

// NormalizeEmail is applied before every write and lookup: the users table
// only accepts lowercase emails, so two accounts cannot differ by case.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
