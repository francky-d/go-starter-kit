package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/config"
	"github.com/francky-d/go-starter-kit/internal/logging"
)

func TestNew_WritesJSONRecords(t *testing.T) {
	var output bytes.Buffer
	logger := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatJSON}, &output)

	logger.Info("user registered", "user_id", "42")

	var record map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &record))
	assert.Equal(t, "INFO", record["level"])
	assert.Equal(t, "user registered", record["msg"])
	assert.Equal(t, "42", record["user_id"])
}

func TestNew_WritesTextRecords(t *testing.T) {
	var output bytes.Buffer
	logger := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatText}, &output)

	logger.Info("user registered", "user_id", "42")

	assert.Contains(t, output.String(), `msg="user registered"`)
	assert.Contains(t, output.String(), "user_id=42")
}

func TestNew_DropsRecordsBelowTheConfiguredLevel(t *testing.T) {
	var output bytes.Buffer
	logger := logging.New(config.Log{Level: slog.LevelWarn, Format: config.LogFormatJSON}, &output)

	logger.Info("ignored")
	logger.Warn("kept")

	assert.NotContains(t, output.String(), "ignored")
	assert.Contains(t, output.String(), "kept")
}

func TestNew_AddsTheRequestIDCarriedByTheContext(t *testing.T) {
	var output bytes.Buffer
	logger := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatJSON}, &output)
	ctx := logging.ContextWithRequestID(t.Context(), "req-42")

	logger.With("app", "api").InfoContext(ctx, "within a request")
	logger.Info("outside a request")

	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)
	var within, outside map[string]any
	require.NoError(t, json.Unmarshal(lines[0], &within))
	require.NoError(t, json.Unmarshal(lines[1], &outside))
	assert.Equal(t, "req-42", within["request_id"])
	assert.Equal(t, "api", within["app"])
	assert.NotContains(t, outside, "request_id")
}

func TestDiscard_WritesNothing(t *testing.T) {
	logger := logging.Discard()

	assert.False(t, logger.Enabled(t.Context(), slog.LevelError))
}
