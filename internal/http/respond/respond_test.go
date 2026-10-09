package respond_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/http/respond"
)

func newContext(method string) (echo.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "/", nil)

	return echo.New().NewContext(request, recorder), recorder
}

func TestOK_WrapsDataInTheEnvelope(t *testing.T) {
	c, recorder := newContext(http.MethodGet)

	require.NoError(t, respond.OK(c, map[string]string{"name": "Ada"}))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"data":{"name":"Ada"}}`, recorder.Body.String())
}

func TestCreated_Answers201(t *testing.T) {
	c, recorder := newContext(http.MethodPost)

	require.NoError(t, respond.Created(c, map[string]string{"id": "1"}))

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.JSONEq(t, `{"data":{"id":"1"}}`, recorder.Body.String())
}

func TestNoContent_AnswersWithoutBody(t *testing.T) {
	c, recorder := newContext(http.MethodDelete)

	require.NoError(t, respond.NoContent(c))

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func TestPaginated_AddsPaginationMeta(t *testing.T) {
	c, recorder := newContext(http.MethodGet)

	err := respond.Paginated(c, []string{"a", "b"}, respond.Pagination{Page: 2, PerPage: 2, Total: 5})

	require.NoError(t, err)
	assert.JSONEq(t,
		`{"data":["a","b"],"meta":{"page":2,"per_page":2,"total":5,"last_page":3}}`,
		recorder.Body.String())
}

func TestPagination_LastPageIsAtLeastOne(t *testing.T) {
	assert.Equal(t, 1, respond.Pagination{Page: 1, PerPage: 20, Total: 0}.LastPage())
	assert.Equal(t, 1, respond.Pagination{Page: 1, PerPage: 0, Total: 10}.LastPage())
	assert.Equal(t, 1, respond.Pagination{Page: 1, PerPage: 20, Total: 20}.LastPage())
	assert.Equal(t, 2, respond.Pagination{Page: 1, PerPage: 20, Total: 21}.LastPage())
}
