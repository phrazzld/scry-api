package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/service/auth"
)

// GenerateAuthHeader creates an Authorization header value (Bearer token) for tests.
func GenerateAuthHeader(userID uuid.UUID) (string, error) {
	// Create a test JWT service
	authConfig := &auth.JWTConfig{
		Secret:               "test-jwt-secret",
		TokenLifetimeMinutes: 60,
	}
	jwtService, err := auth.NewJWTService(authConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT service: %w", err)
	}

	// Generate a token
	token, err := jwtService.GenerateToken(context.Background(), userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return "Bearer " + token, nil
}

// CreateTestJWTService creates a JWT service for testing with a pre-configured secret and expiration.
func CreateTestJWTService() (auth.JWTService, error) {
	authConfig := &auth.JWTConfig{
		Secret:               "test-jwt-secret",
		TokenLifetimeMinutes: 60,
	}
	return auth.NewJWTService(authConfig)
}

// AssertErrorResponse checks that a response contains an error with the expected status code.
func AssertErrorResponse(resp *http.Response, expectedStatus int, expectedMsg string) error {
	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the error response
	var errorResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
		return fmt.Errorf("failed to unmarshal error response: %w", err)
	}

	// Verify the error message
	if expectedMsg != "" && errorResp.Error != expectedMsg {
		return fmt.Errorf("expected error message %q, got %q", expectedMsg, errorResp.Error)
	}

	return nil
}