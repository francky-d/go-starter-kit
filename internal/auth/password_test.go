package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/francky-d/go-starter-kit/internal/auth"
)

func newHasher(t *testing.T) *auth.PasswordHasher {
	t.Helper()
	hasher, err := auth.NewPasswordHasher(bcrypt.MinCost)
	require.NoError(t, err)

	return hasher
}

func TestPasswordHasher_VerifiesTheHashedPassword(t *testing.T) {
	hasher := newHasher(t)

	hash, err := hasher.Hash("correct horse battery staple")

	require.NoError(t, err)
	assert.NotContains(t, hash, "correct horse")
	assert.NoError(t, hasher.Verify(hash, "correct horse battery staple"))
}

func TestPasswordHasher_RejectsAWrongPassword(t *testing.T) {
	hasher := newHasher(t)
	hash, err := hasher.Hash("correct horse battery staple")
	require.NoError(t, err)

	assert.ErrorIs(t, hasher.Verify(hash, "wrong password"), auth.ErrPasswordMismatch)
}

func TestPasswordHasher_SaltsEveryHash(t *testing.T) {
	hasher := newHasher(t)

	first, err := hasher.Hash("same password")
	require.NoError(t, err)
	second, err := hasher.Hash("same password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestPasswordHasher_RejectsPasswordsOverSeventyTwoBytes(t *testing.T) {
	hasher := newHasher(t)
	accented := strings.Repeat("é", 37) // 37 characters, 74 bytes

	_, err := hasher.Hash(accented)

	assert.ErrorIs(t, err, auth.ErrPasswordTooLong)
}

func TestPasswordHasher_DoesNotVerifyAPasswordExtendedPastSeventyTwoBytes(t *testing.T) {
	hasher := newHasher(t)
	password := strings.Repeat("a", auth.MaxPasswordBytes)
	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	err = hasher.Verify(hash, password+"suffix")

	assert.ErrorIs(t, err, auth.ErrPasswordMismatch)
}

func TestPasswordHasher_UsesTheConfiguredCost(t *testing.T) {
	hasher, err := auth.NewPasswordHasher(bcrypt.MinCost + 1)
	require.NoError(t, err)

	hash, err := hasher.Hash("password")
	require.NoError(t, err)

	cost, err := bcrypt.Cost([]byte(hash))
	require.NoError(t, err)
	assert.Equal(t, bcrypt.MinCost+1, cost)
}

func TestPasswordHasher_NeedsRehashOnlyForAnotherCost(t *testing.T) {
	older := newHasher(t)
	hash, err := older.Hash("password")
	require.NoError(t, err)
	newer, err := auth.NewPasswordHasher(bcrypt.MinCost + 1)
	require.NoError(t, err)

	assert.False(t, older.NeedsRehash(hash))
	assert.True(t, newer.NeedsRehash(hash))
}

func TestPasswordHasher_SimulateVerifyTakesAsLongAsAVerification(t *testing.T) {
	hasher, err := auth.NewPasswordHasher(10)
	require.NoError(t, err)
	hash, err := hasher.Hash("password")
	require.NoError(t, err)

	verify := measure(func() { _ = hasher.Verify(hash, "wrong") })
	simulate := measure(func() { hasher.SimulateVerify("wrong") })

	assert.Greater(t, simulate, verify/2, "the unknown-email path must cost a real bcrypt comparison")
}

func measure(run func()) time.Duration {
	started := time.Now()
	run()

	return time.Since(started)
}
