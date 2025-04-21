package events

import (
	"context"
	"log/slog"
	"sync"
)

// InMemoryEventEmitter is a simple implementation of the EventEmitter interface
// that stores registered handlers in memory and dispatches events to them.
type InMemoryEventEmitter struct {
	handlers []EventHandler
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewInMemoryEventEmitter creates a new instance of InMemoryEventEmitter.
func NewInMemoryEventEmitter(logger *slog.Logger) *InMemoryEventEmitter {
	return &InMemoryEventEmitter{
		handlers: make([]EventHandler, 0),
		logger:   logger.With("component", "in_memory_event_emitter"),
	}
}

// RegisterHandler adds a new event handler to receive events.
func (e *InMemoryEventEmitter) RegisterHandler(handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handler)
	e.logger.Debug("registered new event handler", "handler_count", len(e.handlers))
}

// EmitEvent publishes the given event to all registered handlers.
// If any handler returns an error, the event will still be sent to all other handlers,
// and the first error encountered will be returned.
func (e *InMemoryEventEmitter) EmitEvent(ctx context.Context, event *TaskRequestEvent) error {
	e.mu.RLock()
	handlers := make([]EventHandler, len(e.handlers))
	copy(handlers, e.handlers)
	e.mu.RUnlock()

	e.logger.Debug("emitting event",
		"event_id", event.ID,
		"event_type", event.Type,
		"handler_count", len(handlers))

	if len(handlers) == 0 {
		e.logger.Warn("no handlers registered for event",
			"event_id", event.ID,
			"event_type", event.Type)
		return nil
	}

	var firstErr error
	for i, handler := range handlers {
		if err := handler.HandleEvent(ctx, event); err != nil {
			e.logger.Error("handler failed to process event",
				"error", err,
				"handler_index", i,
				"event_id", event.ID,
				"event_type", event.Type)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}
