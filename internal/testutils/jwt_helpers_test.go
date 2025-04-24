package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestJWTService(t *testing.T) {
	// Create the service
	service := NewTestJWTService()

	// Verify we can call methods on the service
	assert.NotNil(t, service, "TestJWTService should not be nil")
}

func TestNewTestJWTServiceWithOptions(t *testing.T) {
	// Create with custom options
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	timeFn := func() time.Time { return fixedTime }

	service := NewTestJWTServiceWithOptions(
		"custom-secret-that-is-at-least-32-chars",
		30*time.Minute,
		2*time.Hour,
		timeFn,
	)

	// Verify we can call methods on the service
	assert.NotNil(t, service, "TestJWTService should not be nil")

	// Create with defaults for empty values
	serviceWithDefaults := NewTestJWTServiceWithOptions("", 0, 0, nil)
	assert.NotNil(t, serviceWithDefaults, "TestJWTService with defaults should not be nil")
}

func TestGenerateAndValidateToken(t *testing.T) {
	// Create the service
	service := NewTestJWTService()
	ctx := context.Background()
	userID := uuid.New()

	// Generate a token
	token, err := service.GenerateToken(ctx, userID)
	require.NoError(t, err, "GenerateToken should not return an error")
	require.NotEmpty(t, token, "Generated token should not be empty")

	// Validate the token
	claims, err := service.ValidateToken(ctx, token)
	require.NoError(t, err, "ValidateToken should not return an error")
	require.NotNil(t, claims, "Claims should not be nil")

	// Check the claims
	assert.Equal(t, userID, claims.UserID, "UserID claim should match")
	assert.Equal(t, "access", claims.TokenType, "TokenType claim should be 'access'")
	assert.Equal(t, userID.String(), claims.Subject, "Subject claim should match userID")
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	// Create the service
	service := NewTestJWTService()
	ctx := context.Background()
	userID := uuid.New()

	// Generate a refresh token
	token, err := service.GenerateRefreshToken(ctx, userID)
	require.NoError(t, err, "GenerateRefreshToken should not return an error")
	require.NotEmpty(t, token, "Generated refresh token should not be empty")

	// Validate the refresh token
	claims, err := service.ValidateRefreshToken(ctx, token)
	require.NoError(t, err, "ValidateRefreshToken should not return an error")
	require.NotNil(t, claims, "Claims should not be nil")

	// Check the claims
	assert.Equal(t, userID, claims.UserID, "UserID claim should match")
	assert.Equal(t, "refresh", claims.TokenType, "TokenType claim should be 'refresh'")
	assert.Equal(t, userID.String(), claims.Subject, "Subject claim should match userID")
}

func TestTokenValidationFailures(t *testing.T) {
	// Create the service
	service := NewTestJWTService()
	ctx := context.Background()
	userID := uuid.New()

	// Invalid token format
	_, err := service.ValidateToken(ctx, "invalid.token.format")
	assert.Error(t, err, "Should error with invalid token format")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)

	// Empty token
	_, err = service.ValidateToken(ctx, "")
	assert.Error(t, err, "Should error with empty token")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)

	// Generate a valid token
	token, err := service.GenerateToken(ctx, userID)
	require.NoError(t, err)

	// Tamper with the token - change a character in the signature
	tamperedToken := token[:len(token)-2] + "xx"
	_, err = service.ValidateToken(ctx, tamperedToken)
	assert.Error(t, err, "Should error with tampered token")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)

	// Test wrong token type - validate a refresh token as an access token
	refreshToken, err := service.GenerateRefreshToken(ctx, userID)
	require.NoError(t, err)
	_, err = service.ValidateToken(ctx, refreshToken)
	assert.Error(t, err, "Should error when validating refresh token as access token")
	assert.ErrorIs(t, err, auth.ErrWrongTokenType)

	// Test expired token
	fixedPastTime := time.Now().Add(-2 * time.Hour)
	expiredService := CreateFixedTimeJWTService(fixedPastTime)
	expiredToken, err := expiredService.GenerateToken(ctx, userID)
	require.NoError(t, err)

	_, err = service.ValidateToken(ctx, expiredToken)
	assert.Error(t, err, "Should error with expired token")
	assert.ErrorIs(t, err, auth.ErrExpiredToken)
}

func TestFixedTimeJWTService(t *testing.T) {
	// Define a fixed time in the future so the token doesn't expire during validation
	futureTime := time.Now().Add(24 * time.Hour)

	// Create a service with fixed time
	service := CreateFixedTimeJWTService(futureTime)
	ctx := context.Background()
	userID := uuid.New()

	// Generate a token
	token, err := service.GenerateToken(ctx, userID)
	require.NoError(t, err)

	// Parse the token manually to check timestamps
	parser := jwt.NewParser(jwt.WithTimeFunc(func() time.Time {
		return futureTime // Use our fixed time for validation
	}))

	parsedToken, err := parser.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(TestJWTSecret), nil
	})
	require.NoError(t, err)

	// Extract claims
	mapClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)

	// Check issued at time
	iat, ok := mapClaims["iat"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(futureTime.Unix()), iat, "IssuedAt time should match fixed time")

	// Check expiry time
	exp, ok := mapClaims["exp"].(float64)
	require.True(t, ok)
	expectedExp := futureTime.Add(TestTokenLifetime).Unix()
	assert.Equal(t, float64(expectedExp), exp, "ExpiresAt time should match fixed time + token lifetime")

	// Use the same service for validation as for generation
	claims, err := service.ValidateToken(ctx, token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
}

func TestGenerateAuthHeader(t *testing.T) {
	// Generate an auth header
	userID := uuid.New()
	header, err := GenerateAuthHeader(userID)
	require.NoError(t, err)
	require.NotEmpty(t, header)

	// Check the header format
	assert.Contains(t, header, "Bearer ", "Auth header should start with 'Bearer '")

	// Extract and validate the token
	token := header[7:] // Remove "Bearer " prefix
	service := NewTestJWTService()
	claims, err := service.ValidateToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestGenerateTokenWithClaims(t *testing.T) {
	// Generate a token with custom claims
	userID := uuid.New()
	token, err := GenerateTokenWithClaims(userID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Validate the token
	service := NewTestJWTService()
	claims, err := service.ValidateToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}
