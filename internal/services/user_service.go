package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
)

type UserService struct {
	users *repositories.UserRepository
}

func NewUserService(users *repositories.UserRepository) *UserService {
	return &UserService{users: users}
}

// CurrentUser returns the user behind an access token. An access token
// outlives the deletion of its user, so a missing user is a 401, not a 404.
func (s *UserService) CurrentUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, apperror.Unauthorized()
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return user, nil
}
