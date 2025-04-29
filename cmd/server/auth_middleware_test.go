//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockJWTService is a mock implementation of auth.JWTService
type MockJWTService struct {
	ValidateErr error
	Claims      *auth.Claims
}

// ValidateToken implements auth.JWTService.ValidateToken
func (m *MockJWTService) ValidateToken(ctx context.Context, tokenString string) (*auth.Claims, error) {
	return m.Claims, m.ValidateErr
}

// GenerateToken implements auth.JWTService.GenerateToken
func (m *MockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	return "mock-token", nil
}

// ValidateRefreshToken implements auth.JWTService.ValidateRefreshToken
func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, tokenString string) (*auth.Claims, error) {
	return m.Claims, m.ValidateErr
}

// GenerateRefreshToken implements auth.JWTService.GenerateRefreshToken
func (m *MockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	return "mock-refresh-token", nil
}

// Use the shared context key for testing
var UserIDKey = shared.UserIDContextKey

func TestAuthMiddleware_Authenticate(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name           string
		authHeader     string
		validateErr    error
		claims         *auth.Claims
		expectedStatus int
		expectedUserID uuid.UUID
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer valid-token",
			validateErr:    nil,
			claims:         &auth.Claims{UserID: userID},
			expectedStatus: http.StatusOK,
			expectedUserID: userID,
		},
		{
			name:           "missing auth header",
			authHeader:     "",
			validateErr:    nil,
			claims:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid auth format",
			authHeader:     "InvalidFormat",
			validateErr:    nil,
			claims:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "expired token",
			authHeader:     "Bearer expired-token",
			validateErr:    auth.ErrExpiredToken,
			claims:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			validateErr:    auth.ErrInvalidToken,
			claims:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock JWT service
			jwtService := &MockJWTService{
				ValidateErr: tt.validateErr,
				Claims:      tt.claims,
			}

			// Create middleware
			authMiddleware := middleware.NewAuthMiddleware(jwtService)

			// Create test handler
			var capturedUserID uuid.UUID
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID, ok := middleware.GetUserID(r)
				if ok {
					capturedUserID = userID
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create request
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Add("Authorization", tt.authHeader)
			}

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Run middleware
			authMiddleware.Authenticate(nextHandler).ServeHTTP(recorder, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			// Check user ID in context
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedUserID, capturedUserID)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	t.Parallel()

	testUserID := uuid.New()

	// Test case 1: Context with user ID
	t.Run("context with user ID", func(t *testing.T) {
		// Create request with user ID in context
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		ctx := context.WithValue(req.Context(), UserIDKey, testUserID)
		req = req.WithContext(ctx)

		// Get user ID from context
		userID, ok := middleware.GetUserID(req)

		// Check results
		assert.True(t, ok)
		assert.Equal(t, testUserID, userID)
	})

	// Test case 2: Context without user ID
	t.Run("context without user ID", func(t *testing.T) {
		// Create request without user ID in context
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		// Get user ID from context
		userID, ok := middleware.GetUserID(req)

		// Check results
		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, userID)
	})
}
