// Package logger provides structured logging functionality for the application
// using Go's standard library log/slog package.
//
// This package implements a simple, yet flexible structured logging system that:
// - Supports multiple log levels (debug, info, warn, error)
// - Outputs logs in JSON format for easy parsing and integration with log aggregators
// - Configures logging based on provided configuration
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
	"runtime"
	"strings"
	"time"
)

// LoggerConfig contains configuration settings for the logger.
// This is a focused configuration type that only contains the
// settings needed for the logger, allowing for decoupling from
// the application's main configuration.
type LoggerConfig struct {
	// Level controls the verbosity of application logging.
	// Accepts "debug", "info", "warn", "error" in order
	// of increasing severity. Default is "info" if not specified or invalid.
	Level string

	// EnableSourceInfo determines whether source code information (file/line) is included
	// in log entries. Default is false for performance reasons, but may be enabled
	// in CI or test environments to improve diagnostics.
	EnableSourceInfo bool
}

// loggerKey is an unexported type used as a key for storing and retrieving
// logger instances from a context.Context. Using a custom type for context
// keys prevents key collisions with other packages.
type loggerKey struct{}

// isInCIEnvironment returns true if the code is running in a CI environment.
// This is determined by checking common CI environment variables.
func isInCIEnvironment() bool {
	ciVars := []string{
		"CI",             // Generic
		"GITHUB_ACTIONS", // GitHub Actions
		"GITLAB_CI",      // GitLab CI
		"JENKINS_URL",    // Jenkins
		"TRAVIS",         // Travis CI
		"CIRCLECI",       // Circle CI
	}

	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// getCIMetadata returns a map of CI-specific metadata for enhanced logging.
// This includes information about the CI environment, build, and runner.
func getCIMetadata() map[string]string {
	metadata := make(map[string]string)

	// Common CI variables across different systems
	ciVarMap := map[string]string{
		// GitHub Actions
		"GITHUB_WORKFLOW":   "ci_workflow",
		"GITHUB_RUN_ID":     "ci_run_id",
		"GITHUB_RUN_NUMBER": "ci_run_number",
		"GITHUB_JOB":        "ci_job",
		"GITHUB_REPOSITORY": "ci_repository",
		"GITHUB_REF":        "ci_ref",
		"GITHUB_SHA":        "ci_commit",
		"GITHUB_ACTOR":      "ci_actor",
		"GITHUB_EVENT_NAME": "ci_event",

		// GitLab CI
		"CI_PROJECT_ID":      "ci_project_id",
		"CI_PIPELINE_ID":     "ci_pipeline_id",
		"CI_JOB_ID":          "ci_job_id",
		"CI_COMMIT_SHA":      "ci_commit",
		"CI_COMMIT_REF_NAME": "ci_ref",
		"CI_PROJECT_PATH":    "ci_repository",

		// Travis CI
		"TRAVIS_JOB_ID":    "ci_job_id",
		"TRAVIS_BUILD_ID":  "ci_build_id",
		"TRAVIS_REPO_SLUG": "ci_repository",
		"TRAVIS_COMMIT":    "ci_commit",
		"TRAVIS_BRANCH":    "ci_ref",

		// Jenkins
		"BUILD_ID":     "ci_build_id",
		"BUILD_NUMBER": "ci_build_number",
		"JOB_NAME":     "ci_job",
		"GIT_COMMIT":   "ci_commit",
		"GIT_BRANCH":   "ci_ref",
	}

	// Collect known CI variables
	for envVar, metadataKey := range ciVarMap {
		if value := os.Getenv(envVar); value != "" {
			metadata[metadataKey] = value
		}
	}

	// Add generic CI identifier if we're in a CI environment
	if len(metadata) > 0 {
		metadata["ci"] = "true"
	}

	// Add basic system info
	metadata["go_version"] = runtime.Version()
	metadata["os"] = runtime.GOOS
	metadata["arch"] = runtime.GOARCH

	return metadata
}

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
//  3. Configures handler options based on the environment (regular or CI)
//  4. Creates the appropriate handler (regular JSON or CI-enhanced)
//  5. Creates a new logger with the handler
//  6. Sets the new logger as the default for the application
//
// Parameters:
//   - cfg: A LoggerConfig struct containing logger configuration,
//     including the Level string to determine verbosity
//
// Returns:
//   - *slog.Logger: The configured structured logger instance
//   - error: Always returns nil in current implementation, but included in the signature
//     for future extensions (e.g., adding file logging which might fail)
//
// Default Behavior:
//   - JSON output is directed to stdout
//   - Invalid log levels default to "info" with a warning message to stderr
//   - Source location tracking is enabled in CI environments but disabled in production for performance
//   - When in CI mode, additional metadata is automatically added to logs
func Setup(cfg LoggerConfig) (*slog.Logger, error) {
	// Parse the log level from configuration (case-insensitive)
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
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
			"configured_level", cfg.Level,
			"default_level", "info")
	}

	// Configure the handler options with the parsed level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Enable source information if explicitly requested or if in CI environment
	inCI := isInCIEnvironment()
	if cfg.EnableSourceInfo || inCI {
		opts.AddSource = true
	}

	var handler slog.Handler

	// Use the appropriate handler based on the environment
	if inCI {
		// In CI, use the enhanced handler with additional metadata
		handler = NewCIHandler(os.Stdout, opts)

		// Log that we're using CI enhanced logging
		startTime := time.Now().Format(time.RFC3339)

		// These are slog.Attr instances that will be included in the first log message
		ciAttrs := []any{
			"environment", "ci",
			"log_format", "json",
			"source_enabled", opts.AddSource,
			"start_time", startTime,
		}

		// Add basic CI build information that's useful at the start of logs
		metadata := getCIMetadata()
		for key, value := range metadata {
			ciAttrs = append(ciAttrs, key, value)
		}

		// Create and use a temporary logger to avoid circular dependency
		// when setting up the default logger
		tmpLogger := slog.New(handler)
		tmpLogger.Info("initializing logger in CI environment", ciAttrs...)
	} else {
		// In non-CI environments, use the standard JSON handler
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	// Create the main logger with the configured handler
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
//     or if ctx is nil
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
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

// FromContextOrDefault retrieves a logger from the context, or returns the provided default logger
// if no logger is found in the context.
//
// Parameters:
//   - ctx: The context that may contain a logger
//   - defaultLogger: The logger to return if no logger is found in the context
//
// Returns:
//   - *slog.Logger: The logger from the context or the default logger if none is found
//     or if ctx is nil
func FromContextOrDefault(ctx context.Context, defaultLogger *slog.Logger) *slog.Logger {
	if ctx == nil {
		return defaultLogger
	}
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return defaultLogger
}

// WithLogger adds a logger to the context.
//
// Parameters:
//   - ctx: The parent context
//   - logger: The logger to add to the context
//
// Returns:
//   - context.Context: A new context containing the logger
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		// ALLOW-PANIC: Enforcing required dependency for context enrichment
		panic("logger cannot be nil for WithLogger")
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}
