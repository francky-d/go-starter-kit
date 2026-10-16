// Package requests binds and validates request input. Each endpoint declares
// its own input struct, so a client can never set a field the endpoint does
// not expect.
package requests

import (
	"context"
	"errors"
	"io"
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

	if err := drainBody(c.Request()); err != nil {
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

// drainBody reads what the JSON decoder left after the value. Without a
// Content-Length, the body limit counts bytes as they are read, so a body
// over HTTP_BODY_LIMIT fails here even when the value itself fits. Without
// the baseline body limit, only HTTP_READ_TIMEOUT would bound this read.
func drainBody(request *http.Request) error {
	if request.ContentLength == 0 {
		return nil
	}

	_, err := io.Copy(io.Discard, request.Body)

	return err
}

// malformedRequest keeps the decoder's message out of the response: it can
// quote the request body. A body over HTTP_BODY_LIMIT keeps its 413.
func malformedRequest(err error) error {
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) && httpErr.Code == http.StatusRequestEntityTooLarge {
		return apperror.FromStatus(http.StatusRequestEntityTooLarge).Wrap(err)
	}

	return apperror.BadRequest("Malformed request body.").Wrap(err)
}
