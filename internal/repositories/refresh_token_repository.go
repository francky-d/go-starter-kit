package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/models"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) WithTx(tx *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: tx}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	return translate(r.db.WithContext(ctx).Create(token).Error)
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).Take(&token).Error; err != nil {
		return nil, translate(err)
	}

	return &token, nil
}

// RevokeIfActive revokes the token only if nobody revoked it first, and
// reports whether it did. Two requests racing to use the same token cannot
// both succeed: the second one sees false.
func (r *RefreshTokenRepository) RevokeIfActive(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID, at time.Time) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{"revoked_at": at, "replaced_by_id": replacedBy, "updated_at": at})

	return result.RowsAffected == 1, translate(result.Error)
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return translate(r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{"revoked_at": at, "updated_at": at}).Error)
}
