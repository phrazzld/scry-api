package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			jwtService := &mocks.MockJWTService{
				ValidateErr: tt.validateErr,
				Claims:      tt.claims,
			}

			// Create middleware
			middleware := NewAuthMiddleware(jwtService)

			// Create test handler
			var capturedUserID uuid.UUID
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID, ok := GetUserID(r)
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
			middleware.Authenticate(nextHandler).ServeHTTP(recorder, req)

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
		userID, ok := GetUserID(req)

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
		userID, ok := GetUserID(req)

		// Check results
		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, userID)
	})
}
