package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/service/auth"
)

// MockJWTService provides a mock implementation of auth.JWTService for testing
type MockJWTService struct {
	ValidateTokenFunc                  func(ctx context.Context, token string) (*auth.Claims, error)
	GenerateTokenFunc                  func(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateRefreshTokenFunc           func(ctx context.Context, token string) (*auth.Claims, error)
	GenerateRefreshTokenFunc           func(ctx context.Context, userID uuid.UUID) (string, error)
	GenerateRefreshTokenWithExpiryFunc func(ctx context.Context, userID uuid.UUID, expiryTime time.Time) (string, error)
}

// ValidateToken implements the auth.JWTService interface for testing
func (m *MockJWTService) ValidateToken(ctx context.Context, token string) (*auth.Claims, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, token)
	}
	// Default implementation returns an error
	return nil, fmt.Errorf("mock ValidateToken not implemented")
}

// GenerateToken implements the auth.JWTService interface for testing
func (m *MockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(ctx, userID)
	}
	// Default implementation returns a test token
	return "mock-test-token", nil
}

// ValidateRefreshToken implements the auth.JWTService interface for testing
func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, token string) (*auth.Claims, error) {
	if m.ValidateRefreshTokenFunc != nil {
		return m.ValidateRefreshTokenFunc(ctx, token)
	}
	// Default implementation returns an error
	return nil, auth.ErrInvalidRefreshToken
}

// GenerateRefreshToken implements the auth.JWTService interface for testing
func (m *MockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.GenerateRefreshTokenFunc != nil {
		return m.GenerateRefreshTokenFunc(ctx, userID)
	}
	// Default implementation returns a test refresh token
	return "mock-refresh-token", nil
}

// GenerateRefreshTokenWithExpiry implements the auth.JWTService interface for testing
func (m *MockJWTService) GenerateRefreshTokenWithExpiry(
	ctx context.Context,
	userID uuid.UUID,
	expiryTime time.Time,
) (string, error) {
	if m.GenerateRefreshTokenWithExpiryFunc != nil {
		return m.GenerateRefreshTokenWithExpiryFunc(ctx, userID, expiryTime)
	}
	// Default implementation returns a test refresh token
	return "mock-refresh-token-with-expiry", nil
}
