package testutils

import (
	"context"
	"log/slog"
	"sync"
)

// LogEntry represents a simplified log record for testing
type LogEntry map[string]interface{}

// TestSlogHandler is a memory-backed slog.Handler for testing
type TestSlogHandler struct {
	mu      sync.Mutex
	entries []LogEntry
}

// NewTestSlogHandler creates a new memory-backed slog handler
func NewTestSlogHandler() *TestSlogHandler {
	return &TestSlogHandler{
		entries: make([]LogEntry, 0),
	}
}

// Enabled satisfies slog.Handler interface
func (h *TestSlogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle satisfies slog.Handler interface
func (h *TestSlogHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry := make(LogEntry)
	entry["level"] = r.Level.String()
	entry["message"] = r.Message

	r.Attrs(func(attr slog.Attr) bool {
		entry[attr.Key] = attr.Value.Any()
		return true
	})

	h.entries = append(h.entries, entry)
	return nil
}

// WithAttrs satisfies slog.Handler interface
func (h *TestSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup satisfies slog.Handler interface
func (h *TestSlogHandler) WithGroup(name string) slog.Handler {
	return h
}

// Entries returns all captured log entries
func (h *TestSlogHandler) Entries() []LogEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	result := make([]LogEntry, len(h.entries))
	copy(result, h.entries)
	return result
}

// Clear resets the captured log entries
func (h *TestSlogHandler) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = make([]LogEntry, 0)
}
