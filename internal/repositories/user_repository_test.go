package repositories_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func newUser(email string) *models.User {
	return &models.User{Name: "Ada Lovelace", Email: email, PasswordHash: "hash"}
}

func TestUserRepository_CreateAssignsATimeOrderedID(t *testing.T) {
	users := repositories.NewUserRepository(testutil.DB(t))
	user := newUser("ada@example.com")

	require.NoError(t, users.Create(t.Context(), user))

	assert.Equal(t, uuid.Version(7), user.ID.Version())
	assert.False(t, user.CreatedAt.IsZero())
}

func TestUserRepository_FindsAUserByIDAndByEmail(t *testing.T) {
	users := repositories.NewUserRepository(testutil.DB(t))
	created := newUser("ada@example.com")
	require.NoError(t, users.Create(t.Context(), created))

	byID, err := users.FindByID(t.Context(), created.ID)
	require.NoError(t, err)
	byEmail, err := users.FindByEmail(t.Context(), "ada@example.com")
	require.NoError(t, err)

	assert.Equal(t, created.ID, byID.ID)
	assert.Equal(t, created.ID, byEmail.ID)
	assert.Equal(t, "hash", byEmail.PasswordHash)
}

func TestUserRepository_ReportsMissingUsersAsNotFound(t *testing.T) {
	users := repositories.NewUserRepository(testutil.DB(t))

	_, byID := users.FindByID(t.Context(), uuid.Must(uuid.NewV7()))
	_, byEmail := users.FindByEmail(t.Context(), "nobody@example.com")

	assert.ErrorIs(t, byID, repositories.ErrNotFound)
	assert.ErrorIs(t, byEmail, repositories.ErrNotFound)
}

func TestUserRepository_ReportsATakenEmailAsDuplicate(t *testing.T) {
	users := repositories.NewUserRepository(testutil.DB(t))
	require.NoError(t, users.Create(t.Context(), newUser("ada@example.com")))

	err := users.Create(t.Context(), newUser("ada@example.com"))

	assert.ErrorIs(t, err, repositories.ErrDuplicate)
}

func TestUserRepository_NormalizesEmailsOnCreateAndLookup(t *testing.T) {
	users := repositories.NewUserRepository(testutil.DB(t))
	user := newUser("  Ada@Example.COM ")

	require.NoError(t, users.Create(t.Context(), user))
	found, err := users.FindByEmail(t.Context(), "ADA@example.com")

	require.NoError(t, err)
	assert.Equal(t, "ada@example.com", found.Email)
	assert.ErrorIs(t, users.Create(t.Context(), newUser("ada@EXAMPLE.com")), repositories.ErrDuplicate)
}

func TestUserRepository_UpdatesThePasswordHash(t *testing.T) {
	users := repositories.NewUserRepository(testutil.DB(t))
	user := newUser("ada@example.com")
	require.NoError(t, users.Create(t.Context(), user))

	require.NoError(t, users.UpdatePasswordHash(t.Context(), user.ID, "new hash"))

	updated, err := users.FindByID(t.Context(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "new hash", updated.PasswordHash)
	assert.True(t, updated.UpdatedAt.After(user.UpdatedAt))
	assert.ErrorIs(t, users.UpdatePasswordHash(t.Context(), uuid.Must(uuid.NewV7()), "hash"), repositories.ErrNotFound)
}

func TestUserRepository_WithTxWritesInsideTheTransaction(t *testing.T) {
	db := testutil.DB(t)
	users := repositories.NewUserRepository(db)

	tx := db.Begin()
	require.NoError(t, users.WithTx(tx).Create(t.Context(), newUser("ada@example.com")))
	require.NoError(t, tx.Rollback().Error)

	_, err := users.FindByEmail(t.Context(), "ada@example.com")
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}
