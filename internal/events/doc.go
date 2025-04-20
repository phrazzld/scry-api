// Package events provides types and interfaces for an event-driven architecture.
//
// This package defines event types and handler interfaces that allow for loose coupling
// between components in the system. Services can emit events without knowing which
// handlers will process them, enabling better separation of concerns and reducing
// circular dependencies.
//
// The primary components are:
// - TaskRequestEvent: Represents a request to create a background task
// - EventHandler: Interface for components that can handle events
// - EventEmitter: Interface for components that can emit events
package events
