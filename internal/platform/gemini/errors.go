package gemini

import "errors"

// Error definitions for the gemini package.
var (
	// ErrEmptyMemoText is returned when a memo text is empty.
	ErrEmptyMemoText = errors.New("memo text cannot be empty")
)
