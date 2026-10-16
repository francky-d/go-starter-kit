// Package requests binds and validates request input. Each endpoint declares
// its own input struct, so a client can never set a field the endpoint does
// not expect.
package requests

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(jsonFieldName)

	return v
}

// jsonFieldName makes validation errors name the field the client sent, not
// the Go struct field.
// An empty name makes the validator fall back to the Go field name.
func jsonFieldName(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}

	return name
}

func Bind(c echo.Context, input any) error {
	if err := requireJSONBody(c.Request()); err != nil {
		return err
	}

	if err := c.Bind(input); err != nil {
		return malformedRequest(err)
	}

	return Validate(c.Request().Context(), input)
}

// requireJSONBody rejects a body in any other format: Echo's binder would
// also bind XML, matched by Go field names, and forms.
func requireJSONBody(request *http.Request) error {
	if request.ContentLength == 0 {
		return nil
	}

	mediaType, _, err := mime.ParseMediaType(request.Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return apperror.FromStatus(http.StatusUnsupportedMediaType)
	}

	return nil
}

func Validate(ctx context.Context, input any) error {
	err := validate.StructCtx(ctx, input)

	var invalidFields validator.ValidationErrors
	if errors.As(err, &invalidFields) {
		return apperror.Validation(messagesByField(invalidFields))
	}

	if err != nil {
		return apperror.Internal(err)
	}

	return nil
}

// malformedRequest keeps the decoder's message out of the response: it can
// quote the request body.
func malformedRequest(err error) error {
	return apperror.BadRequest("Malformed request body.").Wrap(err)
}
