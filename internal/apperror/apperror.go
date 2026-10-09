// Package apperror defines the errors the application answers clients with.
//
// An *Error carries the HTTP status and a message that is safe to show to
// any client. The internal cause, when there is one, is kept for the logs
// and never serialised. Services and middleware return these errors;
// controllers pass them through unchanged; the central HTTP error handler
// turns them into responses.
package apperror

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
)

// Error is an application error with a client-safe message.
type Error struct {
	// Status is the HTTP status code of the response.
	Status int
	// Message is shown to the client. It must never contain internal details.
	Message string
	// Fields holds validation messages keyed by input field name.
	Fields map[string][]string

	cause error
}

// New returns an error answering status with message.
func New(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// Error describes the error for logs, including the internal cause.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%d %s: %v", e.Status, e.Message, e.cause)
	}

	return fmt.Sprintf("%d %s", e.Status, e.Message)
}

// Unwrap returns the internal cause, so errors.Is and errors.As see it.
func (e *Error) Unwrap() error {
	return e.cause
}

// Wrap returns a copy of the error that records cause for the logs. The
// client still only sees Message.
func (e *Error) Wrap(cause error) *Error {
	wrapped := *e
	wrapped.Fields = maps.Clone(e.Fields)
	wrapped.cause = cause

	return &wrapped
}

// BadRequest answers 400 with message.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message)
}

// Unauthorized answers 401 with a generic message.
func Unauthorized() *Error {
	return New(http.StatusUnauthorized, "Unauthenticated.")
}

// Forbidden answers 403 with a generic message that names neither the owner
// of the resource nor the permission that was missing.
func Forbidden() *Error {
	return New(http.StatusForbidden, "This action is forbidden.")
}

// NotFound answers 404 with a generic message. Use it both for records that
// do not exist and for records that belong to someone else.
func NotFound() *Error {
	return New(http.StatusNotFound, "Resource not found.")
}

// Conflict answers 409 with message.
func Conflict(message string) *Error {
	return New(http.StatusConflict, message)
}

// Validation answers 422 with one or more messages per invalid field.
func Validation(fields map[string][]string) *Error {
	err := New(http.StatusUnprocessableEntity, "The given data was invalid.")
	err.Fields = fields

	return err
}

// TooManyRequests answers 429 with a generic message.
func TooManyRequests() *Error {
	return New(http.StatusTooManyRequests, "Too many requests.")
}

// Internal answers 500 with a generic message and records cause for the
// logs.
func Internal(cause error) *Error {
	return New(http.StatusInternalServerError, "Internal server error.").Wrap(cause)
}

// ServiceUnavailable answers 503 with a generic message.
func ServiceUnavailable() *Error {
	return New(http.StatusServiceUnavailable, "Service unavailable.")
}

// FromStatus returns the generic error for status. Statuses without a
// dedicated constructor use the standard HTTP reason phrase.
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

	if status < 400 || status > 599 {
		status = http.StatusInternalServerError
	}
	if status == http.StatusInternalServerError {
		return New(status, "Internal server error.")
	}

	if text := http.StatusText(status); text != "" {
		return New(status, text+".")
	}

	return New(status, "Request failed.")
}

// As returns err as an *Error when it is one, or wraps one.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}

	return nil, false
}
