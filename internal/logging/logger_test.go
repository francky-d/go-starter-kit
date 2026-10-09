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

func TestDiscard_WritesNothing(t *testing.T) {
	logger := logging.Discard()

	assert.False(t, logger.Enabled(t.Context(), slog.LevelError))
}
