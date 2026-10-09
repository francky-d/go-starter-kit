// Package apperror defines the errors clients are answered with. An Error
// carries an HTTP status and a message safe for any client; its internal
// cause is kept for the logs and never serialised.
package apperror

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
)

const internalServerErrorMessage = "Internal server error."

type Error struct {
	Status int
	// Message is shown to the client: it must never contain internal details.
	Message string
	// Fields maps each invalid input field to its validation messages.
	Fields map[string][]string

	cause error
}

func New(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// Error includes the internal cause: use it in logs, never in a response.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%d %s: %v", e.Status, e.Message, e.cause)
	}

	return fmt.Sprintf("%d %s", e.Status, e.Message)
}

func (e *Error) Unwrap() error {
	return e.cause
}

// Wrap returns a copy that records cause for the logs; the client still only
// sees Message.
func (e *Error) Wrap(cause error) *Error {
	wrapped := *e
	wrapped.Fields = maps.Clone(e.Fields)
	wrapped.cause = cause

	return &wrapped
}

func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message)
}

func Unauthorized() *Error {
	return New(http.StatusUnauthorized, "Unauthenticated.")
}

// Forbidden names neither the owner of the resource nor the missing
// permission.
func Forbidden() *Error {
	return New(http.StatusForbidden, "This action is forbidden.")
}

// NotFound is also the answer for a record that belongs to someone else, so
// a client cannot tell whether it exists.
func NotFound() *Error {
	return New(http.StatusNotFound, "Resource not found.")
}

func Conflict(message string) *Error {
	return New(http.StatusConflict, message)
}

func Validation(fields map[string][]string) *Error {
	err := New(http.StatusUnprocessableEntity, "The given data was invalid.")
	err.Fields = fields

	return err
}

func TooManyRequests() *Error {
	return New(http.StatusTooManyRequests, "Too many requests.")
}

func Internal(cause error) *Error {
	return New(http.StatusInternalServerError, internalServerErrorMessage).Wrap(cause)
}

func ServiceUnavailable() *Error {
	return New(http.StatusServiceUnavailable, "Service unavailable.")
}

// FromStatus returns the generic error for status, falling back to the
// standard reason phrase. A status outside 4xx and 5xx becomes a 500.
func FromStatus(status int) *Error {
	switch status {
	case http.StatusUnauthorized:
		return Unauthorized()
	case http.StatusForbidden:
		return Forbidden()
	case http.StatusNotFound:
		return NotFound()
	case http.StatusTooManyRequests:
		return TooManyRequests()
	case http.StatusServiceUnavailable:
		return ServiceUnavailable()
	}

	return withReasonPhrase(status)
}

func withReasonPhrase(status int) *Error {
	if status < 400 || status > 599 || status == http.StatusInternalServerError {
		return New(http.StatusInternalServerError, internalServerErrorMessage)
	}

	if text := http.StatusText(status); text != "" {
		return New(status, text+".")
	}

	return New(status, "Request failed.")
}

func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}

	return nil, false
}
