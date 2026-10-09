package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/database"
)

func TestCreateMigration_StartsAtOne(t *testing.T) {
	directory := t.TempDir()

	up, down, err := database.CreateMigration(directory, "create_posts_table")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(directory, "000001_create_posts_table.up.sql"), up)
	assert.Equal(t, filepath.Join(directory, "000001_create_posts_table.down.sql"), down)
	assert.FileExists(t, up)
	assert.FileExists(t, down)
}

func TestCreateMigration_FollowsTheHighestNumber(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "000007_create_users_table.up.sql"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "000002_create_teams_table.up.sql"), nil, 0o644))

	up, _, err := database.CreateMigration(directory, "add_avatar_to_users")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(directory, "000008_add_avatar_to_users.up.sql"), up)
}

func TestCreateMigration_RejectsNamesThatAreNotSnakeCase(t *testing.T) {
	for _, name := range []string{"", "CreatePosts", "create-posts", "1_posts", "../escape"} {
		_, _, err := database.CreateMigration(t.TempDir(), name)

		assert.Error(t, err, name)
	}
}
