package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskRequestEvent(t *testing.T) {
	// Define a sample payload
	type testPayload struct {
		ID     uuid.UUID `json:"id"`
		Action string    `json:"action"`
	}

	payload := testPayload{
		ID:     uuid.New(),
		Action: "test_action",
	}

	// Create a new event
	eventType := "test_event"
	event, err := NewTaskRequestEvent(eventType, payload)

	// Assert creation was successful
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.Equal(t, eventType, event.Type)
	assert.NotNil(t, event.Payload)
	assert.WithinDuration(t, time.Now(), event.CreatedAt, 2*time.Second)

	// Verify payload was correctly serialized
	var decodedPayload testPayload
	err = json.Unmarshal(event.Payload, &decodedPayload)
	require.NoError(t, err)
	assert.Equal(t, payload.ID, decodedPayload.ID)
	assert.Equal(t, payload.Action, decodedPayload.Action)
}

// MockEventHandler implements the EventHandler interface for testing
type MockEventHandler struct {
	// The last event received by this handler
	LastEvent *TaskRequestEvent
	// Error to return from HandleEvent
	HandlerError error
	// Count of events handled
	HandledCount int
}

// HandleEvent implements the EventHandler interface
func (h *MockEventHandler) HandleEvent(ctx context.Context, event *TaskRequestEvent) error {
	h.LastEvent = event
	h.HandledCount++
	return h.HandlerError
}

func TestEventHandler(t *testing.T) {
	// Create a mock handler
	handler := &MockEventHandler{}

	// Create a test event
	event, err := NewTaskRequestEvent("test_type", map[string]string{"key": "value"})
	require.NoError(t, err)

	// Handle the event
	err = handler.HandleEvent(context.Background(), event)
	assert.NoError(t, err)
	assert.Equal(t, 1, handler.HandledCount)
	assert.Equal(t, event, handler.LastEvent)

	// Test error case
	expectedErr := errors.New("handler error")
	handler.HandlerError = expectedErr
	err = handler.HandleEvent(context.Background(), event)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 2, handler.HandledCount)
}
