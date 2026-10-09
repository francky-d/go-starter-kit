package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/francky-d/go-starter-kit/internal/models"
)

func TestRefreshToken_HasExpiredFromItsExpiryOn(t *testing.T) {
	expiresAt := time.Now()
	token := models.RefreshToken{ExpiresAt: expiresAt}

	assert.False(t, token.HasExpired(expiresAt.Add(-time.Second)))
	assert.True(t, token.HasExpired(expiresAt))
}

func TestRefreshToken_IsRevokedOnceRevokedAtIsSet(t *testing.T) {
	token := models.RefreshToken{}
	assert.False(t, token.IsRevoked())

	revokedAt := time.Now()
	token.RevokedAt = &revokedAt
	assert.True(t, token.IsRevoked())
}

func TestRefreshToken_WasReplacedOnceItHasAReplacement(t *testing.T) {
	token := models.RefreshToken{}
	assert.False(t, token.WasReplaced())

	replacement := uuid.Must(uuid.NewV7())
	token.ReplacedByID = &replacement
	assert.True(t, token.WasReplaced())
}
