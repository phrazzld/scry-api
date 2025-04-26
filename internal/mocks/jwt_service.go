package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/service/auth"
)

// MockJWTService implements auth.JWTService for testing
type MockJWTService struct {
	// GenerateTokenFn allows test cases to mock the GenerateToken behavior
	GenerateTokenFn func(ctx context.Context, userID uuid.UUID) (string, error)

	// ValidateTokenFn allows test cases to mock the ValidateToken behavior
	ValidateTokenFn func(ctx context.Context, tokenString string) (*auth.Claims, error)

	// GenerateRefreshTokenFn allows test cases to mock the GenerateRefreshToken behavior
	GenerateRefreshTokenFn func(ctx context.Context, userID uuid.UUID) (string, error)

	// ValidateRefreshTokenFn allows test cases to mock the ValidateRefreshToken behavior
	ValidateRefreshTokenFn func(ctx context.Context, tokenString string) (*auth.Claims, error)

	// Default values used when functions aren't explicitly defined
	Token        string
	RefreshToken string
	Err          error
	ValidateErr  error
	Claims       *auth.Claims
}

// GenerateToken implements the auth.JWTService interface
func (m *MockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	// If a custom function is provided, use it
	if m.GenerateTokenFn != nil {
		return m.GenerateTokenFn(ctx, userID)
	}

	// Otherwise use the default values
	return m.Token, m.Err
}

// ValidateToken implements the auth.JWTService interface
func (m *MockJWTService) ValidateToken(
	ctx context.Context,
	tokenString string,
) (*auth.Claims, error) {
	// If a custom function is provided, use it
	if m.ValidateTokenFn != nil {
		return m.ValidateTokenFn(ctx, tokenString)
	}

	// Otherwise use the default values
	return m.Claims, m.ValidateErr
}

// GenerateRefreshToken implements the auth.JWTService interface
func (m *MockJWTService) GenerateRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
) (string, error) {
	// If a custom function is provided, use it
	if m.GenerateRefreshTokenFn != nil {
		return m.GenerateRefreshTokenFn(ctx, userID)
	}

	// Otherwise use the default values
	return m.RefreshToken, m.Err
}

// ValidateRefreshToken implements the auth.JWTService interface
func (m *MockJWTService) ValidateRefreshToken(
	ctx context.Context,
	tokenString string,
) (*auth.Claims, error) {
	// If a custom function is provided, use it
	if m.ValidateRefreshTokenFn != nil {
		return m.ValidateRefreshTokenFn(ctx, tokenString)
	}

	// Otherwise use the default values
	return m.Claims, m.ValidateErr
}
