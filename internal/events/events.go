package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskRequestEvent represents a request to create a background task.
// It contains the necessary information for task creation without
// direct dependencies on the task package.
type TaskRequestEvent struct {
	// ID is a unique identifier for this event
	ID uuid.UUID `json:"id"`

	// Type indicates the task type that should be created
	Type string `json:"type"`

	// Payload contains the task-specific data serialized as JSON
	Payload json.RawMessage `json:"payload"`

	// CreatedAt is the timestamp when the event was created
	CreatedAt time.Time `json:"created_at"`
}

// UnmarshalPayload decodes the event payload into the provided structure.
func (e *TaskRequestEvent) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

// NewTaskRequestEvent creates a new TaskRequestEvent with the specified type and payload.
func NewTaskRequestEvent(eventType string, payload interface{}) (*TaskRequestEvent, error) {
	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &TaskRequestEvent{
		ID:        uuid.New(),
		Type:      eventType,
		Payload:   payloadBytes,
		CreatedAt: time.Now(),
	}, nil
}

// EventHandler defines an interface for components that can handle events.
// Handlers are responsible for processing events and taking appropriate actions.
type EventHandler interface {
	// HandleEvent processes the given event within the provided context.
	// Returns an error if the event cannot be handled successfully.
	HandleEvent(ctx context.Context, event *TaskRequestEvent) error
}

// EventEmitter defines an interface for components that can emit events.
// This allows services to publish events without direct knowledge of handlers.
type EventEmitter interface {
	// EmitEvent publishes the given event to all registered handlers.
	// Returns an error if the event cannot be emitted.
	EmitEvent(ctx context.Context, event *TaskRequestEvent) error
}
