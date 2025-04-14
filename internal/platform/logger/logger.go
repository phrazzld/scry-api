// Package logger provides structured logging functionality for the application.
package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/phrazzld/scry-api/internal/config"
)

// Setup initializes and configures the application's logging system based on
// the provided configuration. It creates a structured JSON logger with the
// appropriate log level and sets it as the default logger for the application.
//
// It accepts a ServerConfig containing the log level setting and returns the
// configured logger and any error encountered during setup.
func Setup(cfg config.ServerConfig) (*slog.Logger, error) {
	// Parse the log level from configuration (case-insensitive)
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		// If the log level is invalid, use info level as default and log a warning
		level = slog.LevelInfo

		// Create a temporary logger to output the warning
		// This will use the default handler (text output to stderr)
		tmpLogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		tmpLogger.Warn("invalid log level configured, using default level",
			"configured_level", cfg.LogLevel,
			"default_level", "info")
	}

	// Configure the handler options with the parsed level
	opts := &slog.HandlerOptions{
		Level: level,
		// AddSource: true, // Commented out to avoid performance overhead
	}

	// Create a JSON handler that writes to stdout with the configured options
	handler := slog.NewJSONHandler(os.Stdout, opts)

	// handler will be used in the next task, but we're declaring it here
	// to ensure the JSON handler creation is in place
	_ = handler

	// The rest of the function will be implemented in subsequent tasks
	return nil, nil
}
