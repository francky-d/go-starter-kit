package logging

import (
	"context"
	"log/slog"
)

type requestIDKey struct{}

// ContextWithRequestID returns ctx carrying the request id: every record
// logged with that context then includes it as request_id.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// requestIDHandler adds the request id of the context to each record, so
// the records of GORM and of the services name their request too.
type requestIDHandler struct {
	slog.Handler
}

func (h requestIDHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		record = record.Clone()
		record.AddAttrs(slog.String("request_id", requestID))
	}

	return h.Handler.Handle(ctx, record)
}

func (h requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestIDHandler{h.Handler.WithAttrs(attrs)}
}

// WithGroup puts request_id inside the group of the record; the kit never
// opens a group.
func (h requestIDHandler) WithGroup(name string) slog.Handler {
	return requestIDHandler{h.Handler.WithGroup(name)}
}
