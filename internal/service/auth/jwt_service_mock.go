//go:build test || integration

package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MockJWTService is a mock implementation of the JWTService interface for testing.
// This is the single canonical mock implementation to be used in all tests.
type MockJWTService struct {
	// Function fields for custom behaviors
	GenerateTokenFunc                  func(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateTokenFunc                  func(ctx context.Context, tokenString string) (*Claims, error)
	GenerateRefreshTokenFunc           func(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateRefreshTokenFunc           func(ctx context.Context, tokenString string) (*Claims, error)
	GenerateRefreshTokenWithExpiryFunc func(ctx context.Context, userID uuid.UUID, expiryTime time.Time) (string, error)

	// Fixed fields for simple cases
	Token           string        // Default token to return
	RefreshToken    string        // Default refresh token to return
	TokenError      error         // Default error for token generation
	ValidationError error         // Default error for token validation
	Claims          *Claims       // Default claims to return
	TokenLifetime   time.Duration // Custom token lifetime for testing
}

// NewMockJWTService creates a new mock JWT service with default values.
// By default, it returns minimal values that will pass simple validation.
func NewMockJWTService() *MockJWTService {
	now := time.Now()
	userID := uuid.New()

	return &MockJWTService{
		Token:        "mock-jwt-token",
		RefreshToken: "mock-refresh-token",
		Claims: &Claims{
			UserID:    userID,
			TokenType: "access",
			Subject:   userID.String(),
			IssuedAt:  now,
			ExpiresAt: now.Add(1 * time.Hour),
			ID:        uuid.New().String(),
		},
	}
}

// GenerateToken implements the JWTService.GenerateToken method.
func (m *MockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(ctx, userID)
	}
	return m.Token, m.TokenError
}

// ValidateToken implements the JWTService.ValidateToken method.
func (m *MockJWTService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, tokenString)
	}
	return m.Claims, m.ValidationError
}

// GenerateRefreshToken implements the JWTService.GenerateRefreshToken method.
func (m *MockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.GenerateRefreshTokenFunc != nil {
		return m.GenerateRefreshTokenFunc(ctx, userID)
	}
	return m.RefreshToken, m.TokenError
}

// ValidateRefreshToken implements the JWTService.ValidateRefreshToken method.
func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error) {
	if m.ValidateRefreshTokenFunc != nil {
		return m.ValidateRefreshTokenFunc(ctx, tokenString)
	}
	if m.Claims != nil && m.Claims.TokenType == "access" {
		// Create a copy with correct token type
		refreshClaims := *m.Claims
		refreshClaims.TokenType = "refresh"
		return &refreshClaims, m.ValidationError
	}
	return m.Claims, m.ValidationError
}

// GenerateRefreshTokenWithExpiry implements the JWTService.GenerateRefreshTokenWithExpiry method.
func (m *MockJWTService) GenerateRefreshTokenWithExpiry(
	ctx context.Context,
	userID uuid.UUID,
	expiryTime time.Time,
) (string, error) {
	if m.GenerateRefreshTokenWithExpiryFunc != nil {
		return m.GenerateRefreshTokenWithExpiryFunc(ctx, userID, expiryTime)
	}
	return m.RefreshToken, m.TokenError
}

// WithGenerateTokenFunc sets a custom GenerateToken function and returns the mock.
func (m *MockJWTService) WithGenerateTokenFunc(
	fn func(ctx context.Context, userID uuid.UUID) (string, error),
) *MockJWTService {
	m.GenerateTokenFunc = fn
	return m
}

// WithValidateTokenFunc sets a custom ValidateToken function and returns the mock.
func (m *MockJWTService) WithValidateTokenFunc(
	fn func(ctx context.Context, tokenString string) (*Claims, error),
) *MockJWTService {
	m.ValidateTokenFunc = fn
	return m
}

// WithGenerateRefreshTokenFunc sets a custom GenerateRefreshToken function and returns the mock.
func (m *MockJWTService) WithGenerateRefreshTokenFunc(
	fn func(ctx context.Context, userID uuid.UUID) (string, error),
) *MockJWTService {
	m.GenerateRefreshTokenFunc = fn
	return m
}

// WithValidateRefreshTokenFunc sets a custom ValidateRefreshToken function and returns the mock.
func (m *MockJWTService) WithValidateRefreshTokenFunc(
	fn func(ctx context.Context, tokenString string) (*Claims, error),
) *MockJWTService {
	m.ValidateRefreshTokenFunc = fn
	return m
}

// WithGenerateRefreshTokenWithExpiryFunc sets a custom GenerateRefreshTokenWithExpiry function and returns the mock.
func (m *MockJWTService) WithGenerateRefreshTokenWithExpiryFunc(
	fn func(ctx context.Context, userID uuid.UUID, expiryTime time.Time) (string, error),
) *MockJWTService {
	m.GenerateRefreshTokenWithExpiryFunc = fn
	return m
}

// WithToken sets a custom token value and returns the mock.
func (m *MockJWTService) WithToken(token string) *MockJWTService {
	m.Token = token
	return m
}

// WithRefreshToken sets a custom refresh token value and returns the mock.
func (m *MockJWTService) WithRefreshToken(token string) *MockJWTService {
	m.RefreshToken = token
	return m
}

// WithTokenError sets a custom token generation error and returns the mock.
func (m *MockJWTService) WithTokenError(err error) *MockJWTService {
	m.TokenError = err
	return m
}

// WithValidationError sets a custom token validation error and returns the mock.
func (m *MockJWTService) WithValidationError(err error) *MockJWTService {
	m.ValidationError = err
	return m
}

// WithClaims sets custom claims and returns the mock.
func (m *MockJWTService) WithClaims(claims *Claims) *MockJWTService {
	m.Claims = claims
	return m
}

// WithTokenLifetime sets a custom token lifetime and returns the mock.
func (m *MockJWTService) WithTokenLifetime(lifetime time.Duration) *MockJWTService {
	m.TokenLifetime = lifetime
	return m
}
