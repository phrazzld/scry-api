//go:build ignore

// This file is maintained for backward compatibility only.
// Please use internal/service/auth/jwt_service_mock.go instead.

package mocks

import (
	"github.com/phrazzld/scry-api/internal/service/auth"
)

// MockJWTService is a deprecated mock implementation.
// Please use auth.NewMockJWTService() from internal/service/auth/jwt_service_mock.go instead.
type MockJWTService struct {
	// Embed the new mock to forward all calls
	*auth.MockJWTService
}

// NewMockJWTService creates a new mock JWT service from the canonical mock implementation.
// This function is deprecated and is provided for backward compatibility only.
// Please use auth.NewMockJWTService() from internal/service/auth/jwt_service_mock.go instead.
func NewMockJWTService() *MockJWTService {
	return &MockJWTService{
		MockJWTService: auth.NewMockJWTService(),
	}
}
