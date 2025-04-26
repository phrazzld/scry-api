package service

import "errors"

// Common service errors
var (
	// ErrNotOwned indicates a resource is owned by a different user than the one making the request
	ErrNotOwned = errors.New("resource is owned by another user")

	// ErrStatsNotFound indicates that user card statistics were not found
	ErrStatsNotFound = errors.New("user card statistics not found")
)
