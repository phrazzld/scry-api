package generation

import "errors"

// Common errors returned by the generation package
var (
	// ErrGenerationFailed is returned when card generation fails for any general reason
	ErrGenerationFailed = errors.New("failed to generate cards from text")

	// ErrInvalidResponse is returned when the LLM response cannot be parsed or is malformed
	ErrInvalidResponse = errors.New("invalid response from language model")

	// ErrContentBlocked is returned when the LLM blocks the content due to safety filters
	ErrContentBlocked = errors.New("content blocked by language model safety filters")

	// ErrTransientFailure is returned for temporary errors that might resolve on retry
	ErrTransientFailure = errors.New("transient error during card generation")

	// ErrInvalidConfig is returned when the generator configuration is invalid
	ErrInvalidConfig = errors.New("invalid generator configuration")
)
