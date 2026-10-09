// Package logging builds the structured logger shared by the application.
package logging

import (
	"io"
	"log/slog"

	"github.com/francky-d/go-starter-kit/internal/config"
)

// New returns a logger that writes to output in the configured format and
// drops records below the configured level.
func New(cfg config.Log, output io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.Level}

	switch cfg.Format {
	case config.LogFormatText:
		return slog.New(slog.NewTextHandler(output, options))
	default: // config.LogFormatJSON; config.Validate rejects any other value
		return slog.New(slog.NewJSONHandler(output, options))
	}
}

// Discard returns a logger that drops every record. Tests use it to keep
// their output readable.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
