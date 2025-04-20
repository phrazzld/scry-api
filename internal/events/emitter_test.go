package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryEventEmitter(t *testing.T) {
	// Create a minimal logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("emit event with no handlers", func(t *testing.T) {
		emitter := NewInMemoryEventEmitter(logger)
		event, err := NewTaskRequestEvent("test-event", map[string]string{"key": "value"})
		require.NoError(t, err)

		// Should not error even with no handlers
		err = emitter.EmitEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("emit event with successful handlers", func(t *testing.T) {
		emitter := NewInMemoryEventEmitter(logger)

		// Create a few mock handlers
		handler1 := &MockEventHandler{}
		handler2 := &MockEventHandler{}

		// Register the handlers
		emitter.RegisterHandler(handler1)
		emitter.RegisterHandler(handler2)

		// Create and emit an event
		event, err := NewTaskRequestEvent("test-event", map[string]string{"key": "value"})
		require.NoError(t, err)

		err = emitter.EmitEvent(context.Background(), event)
		assert.NoError(t, err)

		// Verify both handlers received the event
		assert.Equal(t, 1, handler1.HandledCount)
		assert.Equal(t, 1, handler2.HandledCount)
		assert.Equal(t, event, handler1.LastEvent)
		assert.Equal(t, event, handler2.LastEvent)
	})

	t.Run("emit event with failing handler", func(t *testing.T) {
		emitter := NewInMemoryEventEmitter(logger)

		// Create handlers - one successful, one that fails
		successHandler := &MockEventHandler{}
		failingHandler := &MockEventHandler{
			HandlerError: errors.New("handler error"),
		}

		// Register both handlers
		emitter.RegisterHandler(successHandler)
		emitter.RegisterHandler(failingHandler)

		// Create and emit an event
		event, err := NewTaskRequestEvent("test-event", map[string]string{"key": "value"})
		require.NoError(t, err)

		// Should return an error from the failing handler
		err = emitter.EmitEvent(context.Background(), event)
		assert.Error(t, err)
		assert.Equal(t, "handler error", err.Error())

		// Both handlers should still have received the event
		assert.Equal(t, 1, successHandler.HandledCount)
		assert.Equal(t, 1, failingHandler.HandledCount)
	})
}
