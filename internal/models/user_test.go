package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/francky-d/go-starter-kit/internal/models"
)

func TestNormalizeEmail_TrimsAndLowercases(t *testing.T) {
	assert.Equal(t, "ada@example.com", models.NormalizeEmail("  Ada@Example.COM "))
}
