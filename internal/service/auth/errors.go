package auth

import "errors"

// Common authentication service errors
var (
	// ErrInvalidToken indicates the token format is invalid or signature doesn't match
	ErrInvalidToken = errors.New("invalid authentication token")

	// ErrExpiredToken indicates the token has expired
	ErrExpiredToken = errors.New("authentication token has expired")

	// ErrTokenNotYetValid indicates the token is not yet valid (nbf claim in the future)
	ErrTokenNotYetValid = errors.New("authentication token not yet valid")

	// ErrMissingToken indicates a token was expected but not provided
	ErrMissingToken = errors.New("authentication token is missing")
)
