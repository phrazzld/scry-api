package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken(t *testing.T) {
	t.Parallel()

	// Create test user data
	userID := uuid.New()
	testRefreshToken := "test-refresh-token"
	newAccessToken := "new-access-token"
	newRefreshToken := "new-refresh-token"

	// Create test auth config
	authConfig := &config.AuthConfig{
		TokenLifetimeMinutes:        60,   // 1 hour token lifetime for tests
		RefreshTokenLifetimeMinutes: 1440, // 24 hours for refresh token
	}

	// Test cases
	tests := []struct {
		name          string
		payload       map[string]interface{}
		setupMock     func() *mocks.MockJWTService
		wantStatus    int
		wantNewTokens bool
	}{
		{
			name: "valid refresh token",
			payload: map[string]interface{}{
				"refresh_token": testRefreshToken,
			},
			setupMock: func() *mocks.MockJWTService {
				// Setup mock to validate the refresh token and return user claims
				return &mocks.MockJWTService{
					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
						if tokenString == testRefreshToken {
							return &auth.Claims{
								UserID:    userID,
								TokenType: "refresh",
							}, nil
						}
						return nil, auth.ErrInvalidRefreshToken
					},
					// Setup mock to generate new tokens
					Token:        newAccessToken,
					RefreshToken: newRefreshToken,
					Err:          nil,
				}
			},
			wantStatus:    http.StatusOK,
			wantNewTokens: true,
		},
		{
			name:    "missing refresh token",
			payload: map[string]interface{}{
				// Empty payload, missing refresh_token
			},
			setupMock: func() *mocks.MockJWTService {
				return &mocks.MockJWTService{
					// No validation should be called if token is missing
					ValidateErr: nil,
				}
			},
			wantStatus:    http.StatusBadRequest,
			wantNewTokens: false,
		},
		{
			name: "invalid refresh token",
			payload: map[string]interface{}{
				"refresh_token": "invalid-token",
			},
			setupMock: func() *mocks.MockJWTService {
				return &mocks.MockJWTService{
					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
						return nil, auth.ErrInvalidRefreshToken
					},
				}
			},
			wantStatus:    http.StatusUnauthorized,
			wantNewTokens: false,
		},
		{
			name: "expired refresh token",
			payload: map[string]interface{}{
				"refresh_token": "expired-token",
			},
			setupMock: func() *mocks.MockJWTService {
				return &mocks.MockJWTService{
					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
						return nil, auth.ErrExpiredRefreshToken
					},
				}
			},
			wantStatus:    http.StatusUnauthorized,
			wantNewTokens: false,
		},
		{
			name: "wrong token type",
			payload: map[string]interface{}{
				"refresh_token": "access-token-not-refresh",
			},
			setupMock: func() *mocks.MockJWTService {
				return &mocks.MockJWTService{
					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
						return nil, auth.ErrWrongTokenType
					},
				}
			},
			wantStatus:    http.StatusUnauthorized,
			wantNewTokens: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			jwtService := tt.setupMock()
			userStore := mocks.NewMockUserStore()                                // Not used in refresh token flow
			passwordVerifier := &mocks.MockPasswordVerifier{ShouldSucceed: true} // Not used in refresh token flow

			// Create handler
			handler := NewAuthHandler(userStore, jwtService, passwordVerifier, authConfig)

			// Create request
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Call handler
			handler.RefreshToken(recorder, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, recorder.Code)

			// Check response for successful cases
			if tt.wantNewTokens {
				var resp RefreshTokenResponse
				err = json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, newAccessToken, resp.AccessToken)
				assert.Equal(t, newRefreshToken, resp.RefreshToken)
				assert.NotEmpty(t, resp.ExpiresAt, "ExpiresAt should be populated")
			}
		})
	}
}
