// Package logger provides structured logging functionality for the application
// using Go's standard library log/slog package.
package logger

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"time"
)

// CIHandler is a custom slog.Handler that adds CI environment metadata
// and source code location to log records.
type CIHandler struct {
	// The underlying handler (usually JSON)
	handler slog.Handler
	// CI metadata to add to every log record
	metadata map[string]string
	// Whether to add source location info
	addSource bool
}

// NewCIHandler creates a new CIHandler that wraps the provided handler,
// adding CI metadata and source information to each log record.
func NewCIHandler(out io.Writer, opts *slog.HandlerOptions) *CIHandler {
	// Get common CI metadata
	metadata := getCIMetadata()

	// Create the base JSON handler
	var handlerOpts *slog.HandlerOptions
	if opts != nil {
		// Clone the options to avoid modifying the caller's options
		handlerOptsCopy := *opts
		handlerOpts = &handlerOptsCopy
	} else {
		handlerOpts = &slog.HandlerOptions{}
	}

	// Create the handler
	jsonHandler := slog.NewJSONHandler(out, handlerOpts)

	return &CIHandler{
		handler:   jsonHandler,
		metadata:  metadata,
		addSource: handlerOpts.AddSource,
	}
}

// Enabled implements the slog.Handler interface.
func (h *CIHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// WithAttrs implements the slog.Handler interface.
func (h *CIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CIHandler{
		handler:   h.handler.WithAttrs(attrs),
		metadata:  h.metadata,
		addSource: h.addSource,
	}
}

// WithGroup implements the slog.Handler interface.
func (h *CIHandler) WithGroup(name string) slog.Handler {
	return &CIHandler{
		handler:   h.handler.WithGroup(name),
		metadata:  h.metadata,
		addSource: h.addSource,
	}
}

// Handle implements the slog.Handler interface.
func (h *CIHandler) Handle(ctx context.Context, record slog.Record) error {
	// Clone the record to avoid modifying the original
	enhanced := record.Clone()

	// Add source information if enabled
	if h.addSource {
		pc, file, line, ok := runtime.Caller(4) // Adjust frames as needed
		if ok {
			// Get function name
			funcName := runtime.FuncForPC(pc).Name()

			// Add source info as structured attributes
			enhanced.AddAttrs(
				slog.String("source_file", file),
				slog.Int("source_line", line),
				slog.String("source_func", funcName),
			)
		}
	}

	// Add CI metadata as attributes
	for key, value := range h.metadata {
		enhanced.AddAttrs(slog.String(key, value))
	}

	// Add timestamp precision enhancement for test failure debugging
	nanoseconds := enhanced.Time.UnixNano() % int64(time.Second)
	enhanced.AddAttrs(slog.Int64("timestamp_nano", nanoseconds))

	// Forward the enhanced record to the underlying handler
	return h.handler.Handle(ctx, enhanced)
}

// TestFailureLogger provides specialized logging for test failures.
// It adds test-specific context and formats errors appropriately for CI environments.
type TestFailureLogger struct {
	logger *slog.Logger
}

// NewTestFailureLogger creates a new test failure logger.
func NewTestFailureLogger(baseLogger *slog.Logger) *TestFailureLogger {
	return &TestFailureLogger{
		logger: baseLogger,
	}
}

// LogTestFailure logs a test failure with detailed diagnostic information.
// It structures the information in a way that's easy to parse in CI logs.
func (tfl *TestFailureLogger) LogTestFailure(
	ctx context.Context,
	testName string,
	err error,
	details map[string]interface{},
) {
	// Create attributes for the test failure
	var attrs []any
	attrs = append(attrs,
		"test_name", testName,
		"test_status", "failed",
	)

	// Add the error if provided
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	// Add details if provided (no need to check for nil - ranging over nil map is safe)
	for k, v := range details {
		attrs = append(attrs, k, v)
	}

	// Source location is already added by the CIHandler if configured
	// Log the test failure at ERROR level for visibility
	tfl.logger.ErrorContext(ctx, "TEST FAILURE", attrs...)
}

// LogTestSkip logs when a test is skipped.
func (tfl *TestFailureLogger) LogTestSkip(ctx context.Context, testName string, reason string) {
	tfl.logger.WarnContext(ctx, "TEST SKIPPED",
		"test_name", testName,
		"test_status", "skipped",
		"reason", reason,
	)
}

// End of TestFailureLogger implementation
