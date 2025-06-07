//go:build test_without_external_deps

package logger_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/stretchr/testify/assert"
)

func TestFromContextOrDefault(t *testing.T) {
	defaultLogger := slog.Default()
	customLogger := slog.New(slog.NewTextHandler(nil, nil))

	tests := []struct {
		name     string
		ctx      context.Context
		expected *slog.Logger
	}{
		{
			name:     "nil_context_returns_default",
			ctx:      nil,
			expected: defaultLogger,
		},
		{
			name:     "context_without_logger_returns_default",
			ctx:      context.Background(),
			expected: defaultLogger,
		},
		{
			name:     "context_with_logger_returns_context_logger",
			ctx:      logger.WithLogger(context.Background(), customLogger),
			expected: customLogger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.FromContextOrDefault(tt.ctx, defaultLogger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithLogger(t *testing.T) {
	t.Run("valid_logger", func(t *testing.T) {
		customLogger := slog.New(slog.NewTextHandler(nil, nil))
		ctx := logger.WithLogger(context.Background(), customLogger)

		// Verify the logger was stored in the context
		retrievedLogger := logger.FromContext(ctx)
		assert.Equal(t, customLogger, retrievedLogger)
	})

	t.Run("nil_logger_panics", func(t *testing.T) {
		assert.Panics(t, func() {
			logger.WithLogger(context.Background(), nil)
		})
	})
}
