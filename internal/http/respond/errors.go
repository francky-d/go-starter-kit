package respond

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

type ErrorBody struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

// ErrorHandler is the single place where error bodies are written. Whatever
// the error, the client only receives a status and a safe message; the
// cause goes to the logs.
func ErrorHandler(log *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		appErr := toAppError(err)
		logError(log, c, appErr, err)

		if writeErr := writeError(c, appErr); writeErr != nil {
			log.Error("write error response", "error", writeErr)
		}
	}
}

// toAppError looks for an *apperror.Error first, so an *echo.HTTPError whose
// Internal field holds one answers the inner status. Echo never builds such
// a chain itself.
func toAppError(err error) *apperror.Error {
	if appErr, ok := apperror.As(err); ok {
		return appErr
	}

	// Echo may put parser details in its message, so only the status is kept.
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return apperror.FromStatus(httpErr.Code).Wrap(err)
	}

	return apperror.Internal(err)
}

func writeError(c echo.Context, appErr *apperror.Error) error {
	if c.Request().Method == http.MethodHead {
		return c.NoContent(appErr.Status)
	}

	return c.JSON(appErr.Status, ErrorBody{Message: appErr.Message, Errors: appErr.Fields})
}

func logError(log *slog.Logger, c echo.Context, appErr *apperror.Error, err error) {
	message := "request rejected"
	if appErr.Status >= http.StatusInternalServerError {
		message = "request failed"
	}

	log.Log(c.Request().Context(), levelFor(appErr.Status), message, errorAttributes(c, appErr, err)...)
}

// levelFor keeps rejected authentication, forbidden actions and rate-limit
// hits visible in production: their cause (an expired or forged token, a
// scripted client) matters. Other client errors are debug noise.
func levelFor(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status == http.StatusUnauthorized, status == http.StatusForbidden, status == http.StatusTooManyRequests:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

func errorAttributes(c echo.Context, appErr *apperror.Error, err error) []any {
	attributes := []any{
		"method", c.Request().Method,
		"path", c.Request().URL.Path,
		"status", appErr.Status,
		"error", err,
	}

	if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
		attributes = append(attributes, "request_id", requestID)
	}

	return attributes
}
