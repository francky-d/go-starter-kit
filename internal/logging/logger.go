package logging

import (
	"io"
	"log/slog"

	"github.com/francky-d/go-starter-kit/internal/config"
)

func New(cfg config.Log, output io.Writer) *slog.Logger {
	return slog.New(requestIDHandler{newHandler(cfg, output)})
}

func newHandler(cfg config.Log, output io.Writer) slog.Handler {
	options := &slog.HandlerOptions{Level: cfg.Level}

	switch cfg.Format {
	case config.LogFormatText:
		return slog.NewTextHandler(output, options)
	default: // config.LogFormatJSON; config.Validate rejects any other value
		return slog.NewJSONHandler(output, options)
	}
}

func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
