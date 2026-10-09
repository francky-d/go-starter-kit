package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/models"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) WithTx(tx *gorm.DB) *UserRepository {
	return &UserRepository{db: tx}
}

// Create and FindByEmail normalise the email themselves, so a caller cannot
// store a mixed-case email or miss an account by case.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.Email = models.NormalizeEmail(user.Email)

	return translate(r.db.WithContext(ctx).Create(user).Error)
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return r.findOne(ctx, "id = ?", id)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.findOne(ctx, "email = ?", models.NormalizeEmail(email))
}

func (r *UserRepository) findOne(ctx context.Context, condition string, value any) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where(condition, value).Take(&user).Error; err != nil {
		return nil, translate(err)
	}

	return &user, nil
}
