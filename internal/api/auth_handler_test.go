package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	// Create dependencies
	userStore := mocks.NewMockUserStore()
	jwtService := &mocks.MockJWTService{Token: "test-token", Err: nil}
	passwordVerifier := &mocks.MockPasswordVerifier{ShouldSucceed: true}

	// Create test auth config
	authConfig := &config.AuthConfig{
		TokenLifetimeMinutes: 60, // 1 hour token lifetime for tests
	}

	// Create handler
	handler := NewAuthHandler(userStore, jwtService, passwordVerifier, authConfig)

	// Test cases
	tests := []struct {
		name       string
		payload    map[string]interface{}
		wantStatus int
		wantToken  bool
	}{
		{
			name: "valid registration",
			payload: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password1234567",
			},
			wantStatus: http.StatusCreated,
			wantToken:  true,
		},
		{
			name: "invalid email",
			payload: map[string]interface{}{
				"email":    "invalid-email",
				"password": "password1234567",
			},
			wantStatus: http.StatusBadRequest,
			wantToken:  false,
		},
		{
			name: "password too short",
			payload: map[string]interface{}{
				"email":    "test2@example.com",
				"password": "short",
			},
			wantStatus: http.StatusBadRequest,
			wantToken:  false,
		},
		{
			name: "missing email",
			payload: map[string]interface{}{
				"password": "password1234567",
			},
			wantStatus: http.StatusBadRequest,
			wantToken:  false,
		},
		{
			name: "missing password",
			payload: map[string]interface{}{
				"email": "test3@example.com",
			},
			wantStatus: http.StatusBadRequest,
			wantToken:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Call handler
			handler.Register(recorder, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, recorder.Code)

			// Check response
			if tt.wantToken {
				var authResp AuthResponse
				err = json.NewDecoder(recorder.Body).Decode(&authResp)
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, authResp.UserID)
				assert.Equal(t, "test-token", authResp.AccessToken)
				assert.NotEmpty(t, authResp.ExpiresAt, "ExpiresAt should be populated")
			}
		})
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()

	// Create test user data
	userID := uuid.New()
	testEmail := "test@example.com"
	testPassword := "password1234567"
	dummyHash := "dummy-hash" // The actual hash value doesn't matter anymore

	// Create common dependencies
	jwtService := &mocks.MockJWTService{Token: "test-token", Err: nil}
	userStore := mocks.NewLoginMockUserStore(userID, testEmail, dummyHash)

	// Test cases
	tests := []struct {
		name             string
		payload          map[string]interface{}
		passwordVerifier *mocks.MockPasswordVerifier
		wantStatus       int
		wantToken        bool
	}{
		{
			name: "valid login",
			payload: map[string]interface{}{
				"email":    testEmail,
				"password": testPassword,
			},
			passwordVerifier: &mocks.MockPasswordVerifier{ShouldSucceed: true},
			wantStatus:       http.StatusOK,
			wantToken:        true,
		},
		{
			name: "invalid email",
			payload: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": testPassword,
			},
			passwordVerifier: &mocks.MockPasswordVerifier{ShouldSucceed: false},
			wantStatus:       http.StatusUnauthorized,
			wantToken:        false,
		},
		{
			name: "invalid password",
			payload: map[string]interface{}{
				"email":    testEmail,
				"password": "wrongpassword",
			},
			passwordVerifier: &mocks.MockPasswordVerifier{ShouldSucceed: false},
			wantStatus:       http.StatusUnauthorized,
			wantToken:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with appropriate password verifier
			// Create test auth config
			authConfig := &config.AuthConfig{
				TokenLifetimeMinutes: 60, // 1 hour token lifetime for tests
			}

			handler := NewAuthHandler(userStore, jwtService, tt.passwordVerifier, authConfig)

			// Create request
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Call handler
			handler.Login(recorder, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, recorder.Code)

			// Check response
			if tt.wantToken {
				var authResp AuthResponse
				err = json.NewDecoder(recorder.Body).Decode(&authResp)
				require.NoError(t, err)
				assert.Equal(t, userID, authResp.UserID)
				assert.Equal(t, "test-token", authResp.AccessToken)
				// We haven't implemented ExpiresAt in Login yet, so we don't check it here
			}
		})
	}
}

// TestRefreshTokenSuccess tests the complete flow of obtaining a refresh token
// via login and then using it to get a new token pair.
func TestRefreshTokenSuccess(t *testing.T) {
	t.Parallel()

	// Create test user data
	userID := uuid.New()
	testEmail := "test@example.com"
	testPassword := "password1234567"
	dummyHash := "dummy-hash"

	// Define test tokens
	initialAccessToken := "initial-access-token"
	initialRefreshToken := "initial-refresh-token"
	newAccessToken := "new-access-token"
	newRefreshToken := "new-refresh-token"

	// Configure the JWT service mock with both token types
	jwtService := &mocks.MockJWTService{
		Token:        initialAccessToken,
		RefreshToken: initialRefreshToken,
		Err:          nil,
	}

	// Set up mock behavior for ValidateRefreshToken
	jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
		// Verify that the token being validated is the one we expect
		if tokenString != initialRefreshToken {
			t.Errorf("Expected refresh token %s, got %s", initialRefreshToken, tokenString)
			return nil, auth.ErrInvalidRefreshToken
		}

		// Return valid claims
		return &auth.Claims{
			UserID:    userID,
			TokenType: "refresh",
			IssuedAt:  time.Now().Add(-10 * time.Minute),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}, nil
	}

	// Set up mock behavior for token generation after refresh
	tokenGenerationCount := 0
	refreshTokenGenerationCount := 0

	jwtService.GenerateTokenFn = func(ctx context.Context, uid uuid.UUID) (string, error) {
		tokenGenerationCount++
		// For the second call (after refresh), return new access token
		if tokenGenerationCount > 1 {
			return newAccessToken, nil
		}
		return initialAccessToken, nil
	}

	jwtService.GenerateRefreshTokenFn = func(ctx context.Context, uid uuid.UUID) (string, error) {
		refreshTokenGenerationCount++
		// For the second call (after refresh), return new refresh token
		if refreshTokenGenerationCount > 1 {
			return newRefreshToken, nil
		}
		return initialRefreshToken, nil
	}

	// Create user store mock
	userStore := mocks.NewLoginMockUserStore(userID, testEmail, dummyHash)

	// Create password verifier mock that will succeed
	passwordVerifier := &mocks.MockPasswordVerifier{ShouldSucceed: true}

	// Create test auth config
	authConfig := &config.AuthConfig{
		TokenLifetimeMinutes:        60,          // 1 hour access token lifetime
		RefreshTokenLifetimeMinutes: 60 * 24 * 7, // 7 days refresh token lifetime
	}

	// Create handler with dependencies
	handler := NewAuthHandler(userStore, jwtService, passwordVerifier, authConfig)

	// STEP 1: Login to get initial tokens
	loginPayload := map[string]interface{}{
		"email":    testEmail,
		"password": testPassword,
	}

	loginPayloadBytes, err := json.Marshal(loginPayload)
	require.NoError(t, err)

	loginReq := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginPayloadBytes))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	// Call login handler
	handler.Login(loginRecorder, loginReq)

	// Check login response
	require.Equal(t, http.StatusOK, loginRecorder.Code)

	var loginResp AuthResponse
	err = json.NewDecoder(loginRecorder.Body).Decode(&loginResp)
	require.NoError(t, err)

	// Verify login response contains expected tokens
	assert.Equal(t, userID, loginResp.UserID)
	assert.Equal(t, initialAccessToken, loginResp.AccessToken)
	assert.Equal(t, initialRefreshToken, loginResp.RefreshToken)
	assert.NotEmpty(t, loginResp.ExpiresAt)

	// STEP 2: Use refresh token to get new tokens
	refreshPayload := RefreshTokenRequest{
		RefreshToken: initialRefreshToken,
	}

	refreshPayloadBytes, err := json.Marshal(refreshPayload)
	require.NoError(t, err)

	refreshReq := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(refreshPayloadBytes))
	refreshReq.Header.Set("Content-Type", "application/json")

	refreshRecorder := httptest.NewRecorder()

	// Call refresh token handler
	handler.RefreshToken(refreshRecorder, refreshReq)

	// Check refresh response
	require.Equal(t, http.StatusOK, refreshRecorder.Code)

	var refreshResp RefreshTokenResponse
	err = json.NewDecoder(refreshRecorder.Body).Decode(&refreshResp)
	require.NoError(t, err)

	// Verify refresh response contains new tokens
	assert.Equal(t, newAccessToken, refreshResp.AccessToken)
	assert.Equal(t, newRefreshToken, refreshResp.RefreshToken)
	assert.NotEmpty(t, refreshResp.ExpiresAt)

	// Verify token generation functions were called the expected number of times
	assert.Equal(t, 2, tokenGenerationCount, "GenerateToken should be called twice: once for login, once for refresh")
	assert.Equal(
		t,
		2,
		refreshTokenGenerationCount,
		"GenerateRefreshToken should be called twice: once for login, once for refresh",
	)
}
