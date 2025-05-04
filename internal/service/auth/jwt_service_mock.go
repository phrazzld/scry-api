//go:build integration

package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MockJWTService is a mock implementation of the JWTService interface for testing.
type MockJWTService struct {
	GenerateTokenFunc                  func(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateTokenFunc                  func(ctx context.Context, tokenString string) (*Claims, error)
	GenerateRefreshTokenFunc           func(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateRefreshTokenFunc           func(ctx context.Context, tokenString string) (*Claims, error)
	GenerateRefreshTokenWithExpiryFunc func(ctx context.Context, userID uuid.UUID, expiryTime time.Time) (string, error)
}

// GenerateToken creates a mock JWT token.
func (m *MockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(ctx, userID)
	}
	return "mock-token", nil
}

// ValidateToken validates a mock JWT token.
func (m *MockJWTService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, tokenString)
	}
	return &Claims{UserID: uuid.New(), TokenType: "access"}, nil
}

// GenerateRefreshToken creates a mock refresh token.
func (m *MockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.GenerateRefreshTokenFunc != nil {
		return m.GenerateRefreshTokenFunc(ctx, userID)
	}
	return "mock-refresh-token", nil
}

// ValidateRefreshToken validates a mock refresh token.
func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error) {
	if m.ValidateRefreshTokenFunc != nil {
		return m.ValidateRefreshTokenFunc(ctx, tokenString)
	}
	return &Claims{UserID: uuid.New(), TokenType: "refresh"}, nil
}

// GenerateRefreshTokenWithExpiry creates a mock refresh token with a custom expiry time.
func (m *MockJWTService) GenerateRefreshTokenWithExpiry(
	ctx context.Context,
	userID uuid.UUID,
	expiryTime time.Time,
) (string, error) {
	if m.GenerateRefreshTokenWithExpiryFunc != nil {
		return m.GenerateRefreshTokenWithExpiryFunc(ctx, userID, expiryTime)
	}
	return "mock-refresh-token-with-custom-expiry", nil
}
