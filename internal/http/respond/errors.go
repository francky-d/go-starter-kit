package respond

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

// ErrorBody is the body of every error response.
type ErrorBody struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

// ErrorHandler returns the central Echo error handler. It turns any error
// returned by a handler or a middleware into the error envelope:
//
//   - an *apperror.Error answers its own status, message and field errors;
//   - an *echo.HTTPError (unknown route, wrong method, body too large...)
//     answers its status with a generic message, because Echo may put
//     internal details such as parser errors in its message;
//   - any other error answers 500 with a generic message.
//
// Server errors are logged at error level with their cause. Rejected
// authentication, forbidden actions and rate-limit hits are logged at info
// level, because their cause (an expired or forged token, a scripted
// client) matters in production. Other client errors are logged at debug
// level only.
func ErrorHandler(log *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		appErr := toAppError(err)
		logError(log, c, appErr, err)

		var writeErr error
		if c.Request().Method == http.MethodHead {
			writeErr = c.NoContent(appErr.Status)
		} else {
			writeErr = c.JSON(appErr.Status, ErrorBody{Message: appErr.Message, Errors: appErr.Fields})
		}

		if writeErr != nil {
			log.Error("write error response", "error", writeErr)
		}
	}
}

// toAppError finds the outermost *apperror.Error in the chain first. An
// *echo.HTTPError whose Internal field holds an *apperror.Error therefore
// answers the inner error's status; Echo never builds such a chain itself.
func toAppError(err error) *apperror.Error {
	if appErr, ok := apperror.As(err); ok {
		return appErr
	}

	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return apperror.FromStatus(httpErr.Code).Wrap(err)
	}

	return apperror.Internal(err)
}

func logError(log *slog.Logger, c echo.Context, appErr *apperror.Error, err error) {
	attributes := []any{
		"method", c.Request().Method,
		"path", c.Request().URL.Path,
		"status", appErr.Status,
		"error", err,
	}
	if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
		attributes = append(attributes, "request_id", requestID)
	}

	switch appErr.Status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		log.Info("request rejected", attributes...)
	default:
		if appErr.Status >= http.StatusInternalServerError {
			log.Error("request failed", attributes...)
			return
		}
		log.Debug("request rejected", attributes...)
	}
}
