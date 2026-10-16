package requests_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/http/requests"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/logging"
)

type address struct {
	City string `json:"city" validate:"required"`
}

type signUp struct {
	Name     string    `json:"name" validate:"required,max=10"`
	Email    string    `json:"email" validate:"required,email"`
	Password string    `json:"password" validate:"required,min=8"`
	Role     string    `json:"role" validate:"omitempty,oneof=admin member"`
	Address  *address  `json:"address" validate:"omitempty"`
	Tags     []string  `json:"tags" validate:"max=2"`
	Items    []address `json:"items" validate:"dive"`
	Nickname *string   `json:"nickname" validate:"omitnil,min=1"`
	Labels   []string  `json:"labels" validate:"max=1"`
}

// post sends body through a real Echo instance, so binding, validation and
// the error handler are exercised together.
func post(t *testing.T, contentType, body string) (*httptest.ResponseRecorder, signUp) {
	t.Helper()
	var bound signUp

	e := echo.New()
	e.HTTPErrorHandler = respond.ErrorHandler(logging.Discard())
	e.POST("/", func(c echo.Context) error {
		if err := requests.Bind(c, &bound); err != nil {
			return err
		}
		return c.NoContent(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, contentType)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)

	return recorder, bound
}

func postJSON(t *testing.T, body string) (*httptest.ResponseRecorder, signUp) {
	t.Helper()

	return post(t, echo.MIMEApplicationJSON, body)
}

func TestBind_FillsTheInputWhenValid(t *testing.T) {
	response, bound := postJSON(t, `{"name":"Ada","email":"ada@example.com","password":"long-enough","unknown":"ignored"}`)

	require.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, "Ada", bound.Name)
	assert.Equal(t, "ada@example.com", bound.Email)
}

func TestBind_ReportsMissingFieldsByTheirJSONName(t *testing.T) {
	response, _ := postJSON(t, `{}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.JSONEq(t, `{
		"message": "The given data was invalid.",
		"errors": {
			"name": ["is required"],
			"email": ["is required"],
			"password": ["is required"]
		}
	}`, response.Body.String())
}

func TestBind_TreatsAnEmptyBodyAsMissingFields(t *testing.T) {
	response, _ := postJSON(t, ``)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

func TestBind_DescribesEachRuleWithoutEchoingTheValue(t *testing.T) {
	response, _ := postJSON(t, `{
		"name": "A name that is far too long",
		"email": "not-an-email",
		"password": "short",
		"role": "owner",
		"tags": ["a", "b", "c"]
	}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.JSONEq(t, `{
		"message": "The given data was invalid.",
		"errors": {
			"name": ["must be at most 10 characters"],
			"email": ["must be a valid email address"],
			"password": ["must be at least 8 characters"],
			"role": ["must be one of: admin member"],
			"tags": ["must be at most 2 items"]
		}
	}`, response.Body.String())
	assert.NotContains(t, response.Body.String(), "not-an-email")
}

func TestBind_UsesTheSingularForALimitOfOne(t *testing.T) {
	response, _ := postJSON(t, `{
		"name": "Ada",
		"email": "ada@example.com",
		"password": "long-enough",
		"nickname": "",
		"labels": ["a", "b"]
	}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.JSONEq(t, `{
		"message": "The given data was invalid.",
		"errors": {
			"nickname": ["must be at least 1 character"],
			"labels": ["must be at most 1 item"]
		}
	}`, response.Body.String())
}

func TestBind_NamesNestedFieldsByTheirPath(t *testing.T) {
	response, _ := postJSON(t, `{
		"name": "Ada", "email": "ada@example.com", "password": "long-enough",
		"address": {}, "items": [{"city": "Paris"}, {}]
	}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.JSONEq(t, `{
		"message": "The given data was invalid.",
		"errors": {
			"address.city": ["is required"],
			"items[1].city": ["is required"]
		}
	}`, response.Body.String())
}

func TestBind_RejectsMalformedJSONWithoutParserDetails(t *testing.T) {
	response, _ := postJSON(t, `{"name": "Ada", "secret-fragment`)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"message":"Malformed request body."}`, response.Body.String())
}

func TestBind_RejectsWrongTypesWithoutParserDetails(t *testing.T) {
	response, _ := postJSON(t, `{"name": 42}`)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"message":"Malformed request body."}`, response.Body.String())
}

func TestBind_RejectsUnsupportedContentTypes(t *testing.T) {
	response, _ := post(t, "text/plain", `name=Ada`)

	require.Equal(t, http.StatusUnsupportedMediaType, response.Code)
	assert.JSONEq(t, `{"message":"Unsupported Media Type."}`, response.Body.String())
}

// Echo's binder alone would bind XML by Go field name, and forms.
func TestBind_RejectsBodiesInFormatsOtherThanJSON(t *testing.T) {
	bodies := map[string]string{
		echo.MIMEApplicationXML:  `<signUp><Name>Ada</Name></signUp>`,
		echo.MIMEApplicationForm: `name=Ada`,
		"":                       `{"name":"Ada"}`,
	}

	for contentType, body := range bodies {
		t.Run(contentType, func(t *testing.T) {
			response, bound := post(t, contentType, body)

			require.Equal(t, http.StatusUnsupportedMediaType, response.Code)
			assert.Empty(t, bound.Name)
		})
	}
}

func TestBind_AcceptsJSONWithACharset(t *testing.T) {
	response, bound := post(t, "application/json; charset=utf-8",
		`{"name":"Ada","email":"ada@example.com","password":"long-enough"}`)

	require.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, "Ada", bound.Name)
}

func TestValidate_TreatsANonStructInputAsAProgrammingError(t *testing.T) {
	err := requests.Validate(t.Context(), "not a struct")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
