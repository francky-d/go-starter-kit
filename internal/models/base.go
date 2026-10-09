package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BeforeCreate assigns a UUID v7. Its time-ordered prefix keeps new rows
// together in the primary key index, unlike random UUIDs.
func (b *Base) BeforeCreate(*gorm.DB) error {
	if b.ID != uuid.Nil {
		return nil
	}

	id, err := uuid.NewV7()
	if err != nil {
		return err
	}

	b.ID = id

	return nil
}
