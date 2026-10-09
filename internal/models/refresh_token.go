package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	Base
	UserID       uuid.UUID `gorm:"type:uuid"`
	TokenHash    string    `json:"-"`
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	ReplacedByID *uuid.UUID `gorm:"type:uuid"`
}

func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t *RefreshToken) HasExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}
