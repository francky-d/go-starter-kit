package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/francky-d/go-starter-kit/internal/auth"
)

func TestNewRefreshToken_IsRandomAndStoredAsAHash(t *testing.T) {
	first, firstHash := auth.NewRefreshToken()
	second, _ := auth.NewRefreshToken()

	assert.NotEqual(t, first, second)
	assert.Len(t, first, 43) // 32 bytes, base64url without padding
	assert.NotEqual(t, first, firstHash)
	assert.Equal(t, firstHash, auth.HashRefreshToken(first))
}
