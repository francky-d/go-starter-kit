// Package resources shapes models into the JSON clients receive. A model is
// never serialised directly, so a new column never leaks by accident.
package resources

import (
	"time"

	"github.com/francky-d/go-starter-kit/internal/models"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUser(user *models.User) User {
	return User{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}
