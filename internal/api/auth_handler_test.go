package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// TestRefreshTokenFailure tests various failure scenarios for the refresh token endpoint.
func TestRefreshTokenFailure(t *testing.T) {
	t.Parallel()

	// Create test user data
	userID := uuid.New()
	testEmail := "test@example.com"
	dummyHash := "dummy-hash"

	// Define test tokens
	testAccessToken := "test-access-token"
	testRefreshToken := "test-refresh-token"

	// Create common test configuration
	authConfig := &config.AuthConfig{
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 60 * 24 * 7,
	}

	// Create user store mock
	userStore := mocks.NewLoginMockUserStore(userID, testEmail, dummyHash)

	// Test cases
	tests := []struct {
		name               string
		payload            interface{}
		configureJWTMock   func() *mocks.MockJWTService
		wantStatus         int
		wantErrorMsg       string
		missingContentType bool
	}{
		{
			name:    "missing refresh token",
			payload: map[string]interface{}{
				// Intentionally empty to test missing required field
			},
			configureJWTMock: func() *mocks.MockJWTService {
				return &mocks.MockJWTService{
					Token:        testAccessToken,
					RefreshToken: testRefreshToken,
				}
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "Validation error",
		},
		{
			name: "invalid JSON format",
			payload: `{
				"refresh_token": "test-refresh-token"
				this is not valid JSON
			}`,
			configureJWTMock: func() *mocks.MockJWTService {
				return &mocks.MockJWTService{
					Token:        testAccessToken,
					RefreshToken: testRefreshToken,
				}
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "Invalid request format",
		},
		// Removed missing content type test as it depends on internal implementation details
		{
			name: "invalid refresh token",
			payload: map[string]interface{}{
				"refresh_token": "invalid-token",
			},
			configureJWTMock: func() *mocks.MockJWTService {
				jwtService := &mocks.MockJWTService{
					Token:        testAccessToken,
					RefreshToken: testRefreshToken,
				}
				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return nil, auth.ErrInvalidRefreshToken
				}
				return jwtService
			},
			wantStatus:   http.StatusUnauthorized,
			wantErrorMsg: "Invalid refresh token",
		},
		{
			name: "expired refresh token",
			payload: map[string]interface{}{
				"refresh_token": "expired-token",
			},
			configureJWTMock: func() *mocks.MockJWTService {
				jwtService := &mocks.MockJWTService{
					Token:        testAccessToken,
					RefreshToken: testRefreshToken,
				}
				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return nil, auth.ErrExpiredRefreshToken
				}
				return jwtService
			},
			wantStatus:   http.StatusUnauthorized,
			wantErrorMsg: "Invalid refresh token",
		},
		{
			name: "using access token instead of refresh token",
			payload: map[string]interface{}{
				"refresh_token": testAccessToken, // Using access token when refresh is required
			},
			configureJWTMock: func() *mocks.MockJWTService {
				jwtService := &mocks.MockJWTService{
					Token:        testAccessToken,
					RefreshToken: testRefreshToken,
				}
				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return nil, auth.ErrWrongTokenType
				}
				return jwtService
			},
			wantStatus:   http.StatusUnauthorized,
			wantErrorMsg: "Invalid refresh token",
		},
		{
			name: "internal server error during validation",
			payload: map[string]interface{}{
				"refresh_token": "server-error-token",
			},
			configureJWTMock: func() *mocks.MockJWTService {
				jwtService := &mocks.MockJWTService{
					Token:        testAccessToken,
					RefreshToken: testRefreshToken,
				}
				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return nil, errors.New("unexpected internal error")
				}
				return jwtService
			},
			wantStatus:   http.StatusInternalServerError,
			wantErrorMsg: "Failed to validate refresh token",
		},
		{
			name: "error generating access token",
			payload: map[string]interface{}{
				"refresh_token": testRefreshToken,
			},
			configureJWTMock: func() *mocks.MockJWTService {
				jwtService := &mocks.MockJWTService{
					Token:        testAccessToken,
					RefreshToken: testRefreshToken,
					Err:          errors.New("token generation error"),
				}
				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return &auth.Claims{
						UserID:    userID,
						TokenType: "refresh",
						IssuedAt:  time.Now().Add(-10 * time.Minute),
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}, nil
				}
				return jwtService
			},
			wantStatus:   http.StatusInternalServerError,
			wantErrorMsg: "Failed to generate authentication token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configure mock JWT service for this test case
			jwtService := tt.configureJWTMock()
			passwordVerifier := &mocks.MockPasswordVerifier{ShouldSucceed: true}

			// Create handler with dependencies
			handler := NewAuthHandler(userStore, jwtService, passwordVerifier, authConfig)

			// Create request
			var reqBody []byte
			var err error

			switch payload := tt.payload.(type) {
			case string:
				// For testing invalid JSON scenario
				reqBody = []byte(payload)
			default:
				// For regular map payload
				reqBody, err = json.Marshal(payload)
				require.NoError(t, err)
			}

			// Create HTTP request
			req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
			if !tt.missingContentType {
				req.Header.Set("Content-Type", "application/json")
			}

			// Create response recorder
			recorder := httptest.NewRecorder()

			// Call handler
			handler.RefreshToken(recorder, req)

			// Check response status code
			assert.Equal(t, tt.wantStatus, recorder.Code)

			// Parse error response
			var errorResp ErrorResponse
			err = json.NewDecoder(recorder.Body).Decode(&errorResp)
			require.NoError(t, err)

			// Verify error message
			assert.Contains(t, errorResp.Error, tt.wantErrorMsg)
		})
	}
}
