//go:build test_without_external_deps

package logger_test

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIHandler_WithAttrs(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	handler := logger.NewCIHandler(&buf, &slog.HandlerOptions{AddSource: true})

	// Add attributes to the handler
	attrs := []slog.Attr{
		slog.String("test_attr", "test_value"),
		slog.Int("count", 42),
	}

	handlerWithAttrs := handler.WithAttrs(attrs)
	assert.NotNil(t, handlerWithAttrs)

	// Create a logger with the new handler
	log := slog.New(handlerWithAttrs)
	log.Info("test message")

	// Verify the attributes are included in the output
	output := buf.String()
	assert.Contains(t, output, "test_attr")
	assert.Contains(t, output, "test_value")
	assert.Contains(t, output, "count")
	assert.Contains(t, output, "42")
}

func TestCIHandler_WithGroup(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	handler := logger.NewCIHandler(&buf, &slog.HandlerOptions{AddSource: true})

	// Add a group to the handler
	handlerWithGroup := handler.WithGroup("test_group")
	assert.NotNil(t, handlerWithGroup)

	// Create a logger with the new handler
	log := slog.New(handlerWithGroup)
	log.Info("test message", "nested_attr", "nested_value")

	// Verify the group is included in the output
	output := buf.String()
	assert.Contains(t, output, "test_group")
	assert.Contains(t, output, "nested_attr")
	assert.Contains(t, output, "nested_value")
}

func TestNewCIHandler_NewDestinationBranch(t *testing.T) {
	// Test the branch of NewCIHandler that handles when a new bytes.Buffer is needed
	// This happens when the destination is not already a bytes.Buffer

	// Create a file to write to (different from bytes.Buffer)
	tmpFile, err := os.CreateTemp("", "test_ci_handler")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	// This should trigger the creation of a new destination
	handler := logger.NewCIHandler(tmpFile, &slog.HandlerOptions{AddSource: true})
	assert.NotNil(t, handler)

	// Test that it can handle logging
	log := slog.New(handler)
	log.Info("test message")

	// Verify the file contains the logged message
	_, _ = tmpFile.Seek(0, 0)
	content := make([]byte, 1024)
	n, _ := tmpFile.Read(content)
	output := string(content[:n])
	assert.Contains(t, output, "test message")
}
