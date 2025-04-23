// Package domain defines the core business entities and errors.
package domain

import "errors"

// Common domain errors used across the application.
var (
	// ErrValidation is returned when a domain entity fails validation.
	// This is often wrapped with a more specific error message.
	ErrValidation = errors.New("validation failed")

	// ErrInvalidFormat is returned when data is not in the expected format.
	ErrInvalidFormat = errors.New("invalid format")

	// ErrInvalidID is returned when an ID is malformed or invalid.
	ErrInvalidID = errors.New("invalid ID")

	// ErrInvalidEmail is returned when an email address is malformed.
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidPassword is returned when a password doesn't meet requirements.
	ErrInvalidPassword = errors.New("invalid password")

	// ErrEmptyContent is returned when required content is empty.
	ErrEmptyContent = errors.New("content cannot be empty")

	// ErrInvalidReviewOutcome is returned when a review outcome is not valid.
	ErrInvalidReviewOutcome = errors.New("invalid review outcome")

	// ErrInvalidCardContent is returned when card content is not valid JSON.
	ErrInvalidCardContent = errors.New("invalid card content")

	// ErrInvalidMemoStatus is returned when a memo status is not valid.
	ErrInvalidMemoStatus = errors.New("invalid memo status")

	// ErrUnauthorized is returned when an operation is not permitted.
	ErrUnauthorized = errors.New("unauthorized operation")
)
