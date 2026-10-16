package apperror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

func TestConstructors_UseTheExpectedStatus(t *testing.T) {
	cases := map[string]struct {
		err    *apperror.Error
		status int
	}{
		"bad request":         {apperror.BadRequest("Malformed request body."), http.StatusBadRequest},
		"unauthorized":        {apperror.Unauthorized(), http.StatusUnauthorized},
		"forbidden":           {apperror.Forbidden(), http.StatusForbidden},
		"not found":           {apperror.NotFound(), http.StatusNotFound},
		"conflict":            {apperror.Conflict("Already taken."), http.StatusConflict},
		"validation":          {apperror.Validation(nil), http.StatusUnprocessableEntity},
		"too many requests":   {apperror.TooManyRequests(), http.StatusTooManyRequests},
		"internal":            {apperror.Internal(errors.New("boom")), http.StatusInternalServerError},
		"service unavailable": {apperror.ServiceUnavailable(), http.StatusServiceUnavailable},
		"client closed":       {apperror.ClientClosedRequest(), apperror.StatusClientClosedRequest},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.status, tc.err.Status)
			assert.NotEmpty(t, tc.err.Message)
		})
	}
}

func TestWrap_KeepsTheCauseOutOfTheMessage(t *testing.T) {
	cause := errors.New("pq: relation \"users\" does not exist")

	err := apperror.Internal(cause)

	assert.Equal(t, "Internal server error.", err.Message)
	assert.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), "relation")
}

func TestWrap_DoesNotModifyTheOriginal(t *testing.T) {
	original := apperror.Validation(map[string][]string{"email": {"is required"}})

	wrapped := original.Wrap(errors.New("cause"))
	wrapped.Fields["email"] = append(wrapped.Fields["email"], "changed")

	assert.NoError(t, original.Unwrap())
	assert.Equal(t, []string{"is required"}, original.Fields["email"])
}

func TestFromStatus_ReturnsGenericMessages(t *testing.T) {
	assert.Equal(t, "Resource not found.", apperror.FromStatus(http.StatusNotFound).Message)
	assert.Equal(t, "Method Not Allowed.", apperror.FromStatus(http.StatusMethodNotAllowed).Message)
	assert.Equal(t, "Request Entity Too Large.", apperror.FromStatus(http.StatusRequestEntityTooLarge).Message)
	assert.Equal(t, http.StatusInternalServerError, apperror.FromStatus(http.StatusOK).Status)
	assert.Equal(t, "Internal server error.", apperror.FromStatus(http.StatusBadGateway+100).Message)
	assert.Equal(t, "Request failed.", apperror.FromStatus(499).Message)
}

func TestAs_FindsWrappedErrors(t *testing.T) {
	err := fmt.Errorf("register: %w", apperror.Conflict("Already taken."))

	appErr, ok := apperror.As(err)

	require.True(t, ok)
	assert.Equal(t, http.StatusConflict, appErr.Status)

	_, ok = apperror.As(errors.New("plain"))
	assert.False(t, ok)
}
