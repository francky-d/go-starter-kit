package repositories_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func TestNormalizePage_DefaultsAndCapsThePageSize(t *testing.T) {
	cases := []struct {
		page, perPage, wantPage, wantPerPage int
	}{
		{0, 0, 1, repositories.DefaultPerPage},
		{-3, -1, 1, repositories.DefaultPerPage},
		{2, 50, 2, 50},
		{1, 10_000, 1, repositories.MaxPerPage},
	}

	for _, tc := range cases {
		page, perPage := repositories.NormalizePage(tc.page, tc.perPage)

		assert.Equal(t, tc.wantPage, page)
		assert.Equal(t, tc.wantPerPage, perPage)
	}
}

func TestPaginate_ReturnsTheRequestedPage(t *testing.T) {
	db := testutil.DB(t)
	users := repositories.NewUserRepository(db)
	for i := range 5 {
		require.NoError(t, users.Create(t.Context(), newUser(fmt.Sprintf("user%d@example.com", i))))
	}

	var page []models.User
	require.NoError(t, db.Scopes(repositories.Paginate(2, 2)).Order("email").Find(&page).Error)

	require.Len(t, page, 2)
	assert.Equal(t, "user2@example.com", page[0].Email)
	assert.Equal(t, "user3@example.com", page[1].Email)
}
