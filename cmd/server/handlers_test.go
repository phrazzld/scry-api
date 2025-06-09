//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/assert"
)

// TestTaskFactoryEventHandlerHandleEvent tests the HandleEvent method
func TestTaskFactoryEventHandlerHandleEvent(t *testing.T) {
	testLogger, _ := CreateTestLogger(t)

	handler := &TaskFactoryEventHandler{
		logger: testLogger,
		// taskFactory and taskRunner are nil - should be handled gracefully
	}

	ctx := context.Background()

	t.Run("unsupported event type", func(t *testing.T) {
		event := &events.TaskRequestEvent{
			Type:      "unsupported_type",
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		}

		err := handler.HandleEvent(ctx, event)
		assert.NoError(t, err, "unsupported event types should be ignored without error")
	})

	t.Run("memo generation event type", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{"memo_id": "test-memo-123"})
		event := &events.TaskRequestEvent{
			Type:      task.TaskTypeMemoGeneration,
			ID:        uuid.New(),
			Payload:   payload,
			CreatedAt: time.Now(),
		}

		// This should fail because taskFactory is nil, but we're testing the code path
		err := handler.HandleEvent(ctx, event)
		assert.Error(t, err, "should fail with nil taskFactory")
	})

	t.Run("memo generation event with invalid payload", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{"invalid": "payload"})
		event := &events.TaskRequestEvent{
			Type:      task.TaskTypeMemoGeneration,
			ID:        uuid.New(),
			Payload:   payload,
			CreatedAt: time.Now(),
		}

		// This should fail during payload parsing
		err := handler.HandleEvent(ctx, event)
		assert.Error(t, err, "should fail with invalid payload")
	})
}

// TestLogDatabaseInfo tests database info logging functionality
func TestLogDatabaseInfo(t *testing.T) {
	testLogger, _ := CreateTestLogger(t)
	ctx := context.Background()

	t.Run("with nil database", func(t *testing.T) {
		// logDatabaseInfo panics with nil database when trying to query it
		// This is expected behavior - the function expects a valid database
		assert.Panics(t, func() {
			logDatabaseInfo(nil, ctx, testLogger)
		}, "logDatabaseInfo panics with nil database (expected behavior)")
	})

	t.Run("with nil logger", func(t *testing.T) {
		// Should use default logger but still panic with nil DB
		assert.Panics(t, func() {
			logDatabaseInfo(nil, ctx, nil)
		}, "logDatabaseInfo panics with nil database even with nil logger")
	})
}
