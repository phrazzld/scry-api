// Package logger provides structured logging functionality for the application
// using Go's standard library log/slog package.
//
// This package implements a simple, yet flexible structured logging system that:
// - Supports multiple log levels (debug, info, warn, error)
// - Outputs logs in JSON format for easy parsing and integration with log aggregators
// - Configures logging based on application configuration
// - Provides a consistent logging interface throughout the application
//
// The primary entry point is the Setup function, which initializes the logger
// based on the provided configuration and sets it as the default logger for
// the application.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/phrazzld/scry-api/internal/config"
)

// loggerKey is an unexported type used as a key for storing and retrieving
// logger instances from a context.Context. Using a custom type for context
// keys prevents key collisions with other packages.
type loggerKey struct{} // nolint:unused // Will be used in upcoming contextual logging implementation

// Setup initializes and configures the application's logging system based on
// the provided configuration. It creates a structured JSON logger with the
// appropriate log level and sets it as the default logger for the application.
//
// Supported log levels (case-insensitive):
//   - "debug": Most verbose level, includes all log messages
//   - "info": Standard level for informational messages (default if invalid)
//   - "warn": Only warning and error messages
//   - "error": Only error messages
//
// The function performs the following:
//  1. Parses the log level from configuration (case-insensitive)
//  2. Handles invalid log levels by defaulting to "info" and logging a warning
//  3. Configures a JSON handler with the appropriate level
//  4. Creates a new logger with the JSON handler
//  5. Sets the new logger as the default for the application
//
// Parameters:
//   - cfg: A ServerConfig struct containing application configuration,
//     including the LogLevel string to determine verbosity
//
// Returns:
//   - *slog.Logger: The configured structured logger instance
//   - error: Always returns nil in current implementation, but included for
//     future extensibility (e.g., if file output is added with potential errors)
//
// Default Behavior:
//   - JSON output is directed to stdout
//   - Invalid log levels default to "info" with a warning message to stderr
//   - Source location tracking is disabled for performance reasons
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

	// Create the main logger with the configured JSON handler
	logger := slog.New(handler)

	// Set this logger as the default for the application
	// This allows using the slog package functions directly (slog.Info, slog.Error, etc.)
	slog.SetDefault(logger)

	// Return the configured logger and nil error to indicate success
	return logger, nil
}

// WithRequestID adds a request ID to the logger in the context.
// It creates a new logger with the request ID added as a structured field
// and returns a new context containing this enhanced logger.
//
// Parameters:
//   - ctx: The parent context
//   - requestID: A unique identifier for the request
//
// Returns:
//   - context.Context: A new context containing the logger with request ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := slog.Default().With(slog.String("request_id", requestID))
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext retrieves a logger from the context, or returns the default logger
// if no logger is found in the context.
//
// Parameters:
//   - ctx: The context that may contain a logger
//
// Returns:
//   - *slog.Logger: The logger from the context or the default logger if none is found
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// LogWithContext logs a message using the logger from the context.
// It retrieves the logger using FromContext and logs the message with the specified
// level and arguments.
//
// Parameters:
//   - ctx: The context that may contain a logger
//   - level: The severity level of the log message
//   - msg: The message to log
//   - args: Optional structured logging attributes as key-value pairs
func LogWithContext(ctx context.Context, level slog.Level, msg string, args ...any) {
	FromContext(ctx).Log(ctx, level, msg, args...)
}
