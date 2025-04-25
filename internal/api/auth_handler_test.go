package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthHandler_Register tests the Register handler functionality.
func TestAuthHandler_Register(t *testing.T) {
	// Define fixed values for consistent testing
	fixedTime := time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC)
	fixedToken := "test-access-token"
	fixedRefreshToken := "test-refresh-token"
	expiresAt := fixedTime.Add(time.Hour).Format(time.RFC3339)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.MockUserStore, *mocks.MockJWTService, *mocks.MockPasswordVerifier)
		expectedStatus int
		expectedBody   string
		wantTokens     bool
	}{
		{
			name: "successful_registration",
			requestBody: RegisterRequest{
				Email:    "newuser@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// UserStore will successfully create the user
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					// Simulate storing the user
					us.Users[user.Email] = user
					return nil
				}
				// JWTService will return fixed tokens
				js.Token = fixedToken
				js.RefreshToken = fixedRefreshToken
				js.Err = nil
			},
			expectedStatus: http.StatusCreated,
			wantTokens:     true,
		},
		{
			name: "invalid_request_format",
			requestBody: `{
				"email": "invalid-json
			}`,
			setupMocks:     func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation error",
			wantTokens:     false,
		},
		{
			name: "missing_required_field",
			requestBody: RegisterRequest{
				Email: "missing@password.com",
				// Password field intentionally omitted
			},
			setupMocks:     func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "required field",
			wantTokens:     false,
		},
		{
			name: "invalid_email_format",
			requestBody: RegisterRequest{
				Email:    "not-an-email",
				Password: "securePassword123",
			},
			setupMocks:     func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid email format",
			wantTokens:     false,
		},
		{
			name: "password_too_short",
			requestBody: RegisterRequest{
				Email:    "valid@example.com",
				Password: "short",
			},
			setupMocks:     func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "too short",
			wantTokens:     false,
		},
		{
			name: "email_already_exists",
			requestBody: RegisterRequest{
				Email:    "existing@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Simulate existing email error
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					return store.ErrEmailExists
				}
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "Email already exists",
			wantTokens:     false,
		},
		{
			name: "database_error",
			requestBody: RegisterRequest{
				Email:    "valid@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Simulate database error
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					return errors.New("database connection error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "unexpected error",
			wantTokens:     false,
		},
		{
			name: "token_generation_error",
			requestBody: RegisterRequest{
				Email:    "valid@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Store user successfully
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					us.Users[user.Email] = user
					return nil
				}
				// But fail when generating token
				js.Err = errors.New("token generation failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to generate authentication tokens",
			wantTokens:     false,
		},
		{
			name: "domain_user_creation_error",
			requestBody: RegisterRequest{
				Email:    "valid@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *mocks.MockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Mock domain error by making the userStore.Create function return a validation error
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					return &domain.ValidationError{
						Field:   "email",
						Message: "already exists",
					}
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid email: already exists",
			wantTokens:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockUserStore := mocks.NewMockUserStore()
			mockJWTService := &mocks.MockJWTService{}
			mockPasswordVerifier := &mocks.MockPasswordVerifier{}

			// Configure mocks based on test case
			tc.setupMocks(mockUserStore, mockJWTService, mockPasswordVerifier)

			// Create auth config with fixed times for testing
			authConfig := &config.AuthConfig{
				JWTSecret:                   "test-secret",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			}

			// Create auth handler with fixed time function for predictable expirations
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewAuthHandler(
				mockUserStore,
				mockJWTService,
				mockPasswordVerifier,
				authConfig,
				logger,
			)
			handler = handler.WithTimeFunc(func() time.Time {
				return fixedTime
			})

			// Create request body
			var reqBody []byte
			var err error
			if str, ok := tc.requestBody.(string); ok {
				// Handle raw JSON string for invalid format tests
				reqBody = []byte(str)
			} else {
				// Handle structured request object
				reqBody, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			}

			// Create request and response recorder
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/register",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call the handler directly
			handler.Register(w, req)

			// Verify response status code
			assert.Equal(t, tc.expectedStatus, w.Code)

			// Parse response
			var respBody map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &respBody)
			require.NoError(t, err)

			// Check for error message if expected
			if tc.expectedBody != "" {
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tc.expectedBody)
			}

			// Check for tokens if successful
			if tc.wantTokens {
				// Check response structure
				assert.Contains(t, respBody, "token", "Response should contain access token")
				assert.Contains(
					t,
					respBody,
					"refresh_token",
					"Response should contain refresh token",
				)
				assert.Contains(
					t,
					respBody,
					"expires_at",
					"Response should contain expiration time",
				)
				assert.Contains(t, respBody, "user_id", "Response should contain user ID")

				// Verify token values
				assert.Equal(t, fixedToken, respBody["token"])
				assert.Equal(t, fixedRefreshToken, respBody["refresh_token"])
				assert.Equal(t, expiresAt, respBody["expires_at"])
			}
		})
	}
}

// TestAuthHandler_Login tests the Login handler functionality.
func TestAuthHandler_Login(t *testing.T) {
	// Define fixed values for consistent testing
	fixedTime := time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC)
	fixedUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedToken := "test-access-token"
	fixedRefreshToken := "test-refresh-token"
	expiresAt := fixedTime.Add(time.Hour).Format(time.RFC3339)
	testEmail := "user@example.com"
	testPassword := "securePassword123"
	hashedPassword := "$2a$10$vdA.EZOiPg3BRwKobGbkjOrZzZcyHXw44D0SyaSKNgdyA6c/J94Py" // Hashed version of "securePassword123"

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.LoginMockUserStore, *mocks.MockJWTService, *mocks.MockPasswordVerifier)
		expectedStatus int
		expectedBody   string
		wantTokens     bool
	}{
		{
			name: "successful_login",
			requestBody: LoginRequest{
				Email:    testEmail,
				Password: testPassword,
			},
			setupMocks: func(us *mocks.LoginMockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Password comparison will succeed
				pv.ShouldSucceed = true
				// Tokens will be generated successfully
				js.Token = fixedToken
				js.RefreshToken = fixedRefreshToken
				js.Err = nil
			},
			expectedStatus: http.StatusOK,
			wantTokens:     true,
		},
		{
			name: "invalid_request_format",
			requestBody: `{
				"email": "invalid-json
			}`,
			setupMocks:     func(us *mocks.LoginMockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation error",
			wantTokens:     false,
		},
		{
			name: "missing_required_field",
			requestBody: LoginRequest{
				Email: testEmail,
				// Password field intentionally omitted
			},
			setupMocks:     func(us *mocks.LoginMockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "required field",
			wantTokens:     false,
		},
		{
			name: "user_not_found",
			requestBody: LoginRequest{
				Email:    "nonexistent@example.com",
				Password: testPassword,
			},
			setupMocks: func(us *mocks.LoginMockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Override the GetByEmail function to simulate user not found
				us.GetByEmailFn = func(ctx context.Context, email string) (*domain.User, error) {
					return nil, store.ErrUserNotFound
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid credentials",
			wantTokens:     false,
		},
		{
			name: "database_error",
			requestBody: LoginRequest{
				Email:    testEmail,
				Password: testPassword,
			},
			setupMocks: func(us *mocks.LoginMockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Simulate database error
				us.GetByEmailError = errors.New("database connection error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to authenticate user",
			wantTokens:     false,
		},
		{
			name: "invalid_password",
			requestBody: LoginRequest{
				Email:    testEmail,
				Password: "wrongPassword",
			},
			setupMocks: func(us *mocks.LoginMockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Password comparison will fail
				pv.ShouldSucceed = false
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid credentials",
			wantTokens:     false,
		},
		{
			name: "token_generation_error",
			requestBody: LoginRequest{
				Email:    testEmail,
				Password: testPassword,
			},
			setupMocks: func(us *mocks.LoginMockUserStore, js *mocks.MockJWTService, pv *mocks.MockPasswordVerifier) {
				// Password check will succeed
				pv.ShouldSucceed = true
				// But token generation will fail
				js.Err = errors.New("token generation failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to generate authentication tokens",
			wantTokens:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create login-specific user store mock
			mockUserStore := mocks.NewLoginMockUserStore(fixedUserID, testEmail, hashedPassword)
			mockJWTService := &mocks.MockJWTService{}
			mockPasswordVerifier := &mocks.MockPasswordVerifier{}

			// Configure mocks based on test case
			tc.setupMocks(mockUserStore, mockJWTService, mockPasswordVerifier)

			// Create auth config
			authConfig := &config.AuthConfig{
				JWTSecret:                   "test-secret",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			}

			// Create auth handler with fixed time function
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewAuthHandler(
				mockUserStore,
				mockJWTService,
				mockPasswordVerifier,
				authConfig,
				logger,
			)
			handler = handler.WithTimeFunc(func() time.Time {
				return fixedTime
			})

			// Create request body
			var reqBody []byte
			var err error
			if str, ok := tc.requestBody.(string); ok {
				// Handle raw JSON string for invalid format tests
				reqBody = []byte(str)
			} else {
				// Handle structured request object
				reqBody, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			}

			// Create request and response recorder
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call the handler directly
			handler.Login(w, req)

			// Verify response status code
			assert.Equal(t, tc.expectedStatus, w.Code)

			// Parse response
			var respBody map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &respBody)
			require.NoError(t, err)

			// Check for error message if expected
			if tc.expectedBody != "" {
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tc.expectedBody)
			}

			// Check for tokens if successful
			if tc.wantTokens {
				// Check response structure
				assert.Contains(t, respBody, "token", "Response should contain access token")
				assert.Contains(
					t,
					respBody,
					"refresh_token",
					"Response should contain refresh token",
				)
				assert.Contains(
					t,
					respBody,
					"expires_at",
					"Response should contain expiration time",
				)
				assert.Contains(t, respBody, "user_id", "Response should contain user ID")

				// Verify token values
				assert.Equal(t, fixedToken, respBody["token"])
				assert.Equal(t, fixedRefreshToken, respBody["refresh_token"])
				assert.Equal(t, expiresAt, respBody["expires_at"])
			}
		})
	}
}

// TestAuthHandler_RefreshToken tests the RefreshToken handler functionality.
func TestAuthHandler_RefreshToken(t *testing.T) {
	// Define fixed values for consistent testing
	fixedTime := time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC)
	fixedUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedToken := "test-access-token"
	fixedRefreshToken := "test-refresh-token"
	expiresAt := fixedTime.Add(time.Hour).Format(time.RFC3339)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.MockJWTService)
		expectedStatus int
		expectedBody   string
		wantTokens     bool
	}{
		{
			name: "successful_token_refresh",
			requestBody: RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMocks: func(js *mocks.MockJWTService) {
				// ValidateRefreshToken will succeed
				js.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return &auth.Claims{UserID: fixedUserID}, nil
				}
				// Token generation will succeed
				js.Token = fixedToken
				js.RefreshToken = fixedRefreshToken
				js.Err = nil
			},
			expectedStatus: http.StatusOK,
			wantTokens:     true,
		},
		{
			name: "invalid_request_format",
			requestBody: `{
				"refresh_token": "invalid-json
			}`,
			setupMocks:     func(js *mocks.MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation error",
			wantTokens:     false,
		},
		{
			name:        "missing_refresh_token",
			requestBody: RefreshTokenRequest{
				// RefreshToken field intentionally omitted
			},
			setupMocks:     func(js *mocks.MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "required field",
			wantTokens:     false,
		},
		{
			name: "invalid_refresh_token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "invalid-refresh-token",
			},
			setupMocks: func(js *mocks.MockJWTService) {
				// ValidateRefreshToken will fail
				js.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return nil, auth.ErrInvalidRefreshToken
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid refresh token",
			wantTokens:     false,
		},
		{
			name: "expired_refresh_token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "expired-refresh-token",
			},
			setupMocks: func(js *mocks.MockJWTService) {
				// ValidateRefreshToken will fail
				js.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return nil, auth.ErrExpiredRefreshToken
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid refresh token",
			wantTokens:     false,
		},
		{
			name: "token_generation_error",
			requestBody: RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMocks: func(js *mocks.MockJWTService) {
				// ValidateRefreshToken will succeed
				js.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return &auth.Claims{UserID: fixedUserID}, nil
				}
				// But token generation will fail
				js.Err = errors.New("token generation failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to generate new authentication tokens",
			wantTokens:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks - only need JWT service for this endpoint
			mockJWTService := &mocks.MockJWTService{}
			mockUserStore := mocks.NewMockUserStore()
			mockPasswordVerifier := &mocks.MockPasswordVerifier{}

			// Configure mocks based on test case
			tc.setupMocks(mockJWTService)

			// Create auth config
			authConfig := &config.AuthConfig{
				JWTSecret:                   "test-secret",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			}

			// Create auth handler with fixed time function
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewAuthHandler(
				mockUserStore,
				mockJWTService,
				mockPasswordVerifier,
				authConfig,
				logger,
			)
			handler = handler.WithTimeFunc(func() time.Time {
				return fixedTime
			})

			// Create request body
			var reqBody []byte
			var err error
			if str, ok := tc.requestBody.(string); ok {
				// Handle raw JSON string for invalid format tests
				reqBody = []byte(str)
			} else {
				// Handle structured request object
				reqBody, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			}

			// Create request and response recorder
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/refresh",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call the handler directly
			handler.RefreshToken(w, req)

			// Verify response status code
			assert.Equal(t, tc.expectedStatus, w.Code)

			// Parse response
			var respBody map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &respBody)
			require.NoError(t, err)

			// Check for error message if expected
			if tc.expectedBody != "" {
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tc.expectedBody)
			}

			// Check for tokens if successful
			if tc.wantTokens {
				// Check response structure
				assert.Contains(t, respBody, "access_token", "Response should contain access token")
				assert.Contains(
					t,
					respBody,
					"refresh_token",
					"Response should contain refresh token",
				)
				assert.Contains(
					t,
					respBody,
					"expires_at",
					"Response should contain expiration time",
				)

				// Verify token values
				assert.Equal(t, fixedToken, respBody["access_token"])
				assert.Equal(t, fixedRefreshToken, respBody["refresh_token"])
				assert.Equal(t, expiresAt, respBody["expires_at"])
			}
		})
	}
}

// TestAuthHandler_GenerateTokenResponse tests the generateTokenResponse helper function.
func TestAuthHandler_GenerateTokenResponse(t *testing.T) {
	// Define fixed values for consistent testing
	fixedTime := time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC)
	fixedUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedToken := "test-access-token"
	fixedRefreshToken := "test-refresh-token"
	expectedExpiresAt := fixedTime.Add(time.Hour).Format(time.RFC3339)

	tests := []struct {
		name          string
		setupMocks    func(*mocks.MockJWTService)
		expectError   bool
		expectedToken string
	}{
		{
			name: "successful_token_generation",
			setupMocks: func(js *mocks.MockJWTService) {
				js.Token = fixedToken
				js.RefreshToken = fixedRefreshToken
				js.Err = nil
			},
			expectError:   false,
			expectedToken: fixedToken,
		},
		{
			name: "access_token_generation_error",
			setupMocks: func(js *mocks.MockJWTService) {
				// GenerateToken will fail
				js.Token = ""
				js.Err = errors.New("failed to generate access token")
			},
			expectError: true,
		},
		{
			name: "refresh_token_generation_error",
			setupMocks: func(js *mocks.MockJWTService) {
				// GenerateToken will succeed, but GenerateRefreshToken will fail
				js.GenerateTokenFn = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return fixedToken, nil
				}
				js.GenerateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return "", errors.New("failed to generate refresh token")
				}
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockJWTService := &mocks.MockJWTService{}
			mockUserStore := mocks.NewMockUserStore()
			mockPasswordVerifier := &mocks.MockPasswordVerifier{}

			// Configure mocks based on test case
			tc.setupMocks(mockJWTService)

			// Create auth config
			authConfig := &config.AuthConfig{
				JWTSecret:                   "test-secret",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			}

			// Create auth handler with fixed time function
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewAuthHandler(
				mockUserStore,
				mockJWTService,
				mockPasswordVerifier,
				authConfig,
				logger,
			)
			handler = handler.WithTimeFunc(func() time.Time {
				return fixedTime
			})

			// Call the helper function
			accessToken, refreshToken, expiresAt, err := handler.generateTokenResponse(
				context.Background(),
				fixedUserID,
			)

			// Verify results
			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
				assert.Empty(t, expiresAt)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedToken, accessToken)
				assert.Equal(t, fixedRefreshToken, refreshToken)
				assert.Equal(t, expectedExpiresAt, expiresAt)
			}
		})
	}
}

// TestAuthHandler_SanitizeValidationError verifies the error sanitization functionality.
func TestAuthHandler_SanitizeValidationError(t *testing.T) {
	tests := []struct {
		name          string
		inputErr      error
		expectedMsg   string
		shouldContain bool
	}{
		{
			name:          "domain_validation_error_with_field",
			inputErr:      &domain.ValidationError{Field: "email", Message: "invalid format"},
			expectedMsg:   "Invalid email: invalid format",
			shouldContain: true,
		},
		{
			name:          "domain_validation_error_without_field",
			inputErr:      &domain.ValidationError{Message: "general validation error"},
			expectedMsg:   "general validation error",
			shouldContain: true,
		},
		{
			name: "validator_error_required",
			inputErr: errors.New(
				"Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag",
			),
			expectedMsg:   "Invalid Email: required field",
			shouldContain: true,
		},
		{
			name: "validator_error_email_format",
			inputErr: errors.New(
				"Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag",
			),
			expectedMsg:   "Invalid Email: invalid email format",
			shouldContain: true,
		},
		{
			name: "validator_error_min_length",
			inputErr: errors.New(
				"Key: 'RegisterRequest.Password' Error:Field validation for 'Password' failed on the 'min' tag",
			),
			expectedMsg:   "Invalid Password: too short",
			shouldContain: true,
		},
		{
			name:          "unknown_error_format",
			inputErr:      errors.New("unexpected error format"),
			expectedMsg:   "Validation error",
			shouldContain: true,
		},
		// Removed the nil error test case since SanitizeValidationError doesn't handle nil errors
		// The function is always called with an already validated non-nil error
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeValidationError(tc.inputErr)
			if tc.shouldContain {
				assert.Contains(t, result, tc.expectedMsg)
			} else {
				assert.NotContains(t, result, tc.expectedMsg)
			}
		})
	}
}

// TestAuthHandler_Integration tests the auth handler's integration with Chi router.
func TestAuthHandler_Integration(t *testing.T) {
	// Create router and set up routes
	r := chi.NewRouter()

	// Define fixed values
	fixedUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedToken := "test-access-token"
	fixedRefreshToken := "test-refresh-token"

	// Set up mocks
	mockUserStore := mocks.NewMockUserStore()
	mockJWTService := &mocks.MockJWTService{
		Token:        fixedToken,
		RefreshToken: fixedRefreshToken,
	}
	mockPasswordVerifier := &mocks.MockPasswordVerifier{
		ShouldSucceed: true,
	}

	// Create auth config
	authConfig := &config.AuthConfig{
		JWTSecret:                   "test-secret",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}

	// Create auth handler
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewAuthHandler(
		mockUserStore,
		mockJWTService,
		mockPasswordVerifier,
		authConfig,
		logger,
	)

	// Set up routes
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
		r.Post("/refresh", handler.RefreshToken)
	})

	// Test registration endpoint with router
	t.Run("router_registration", func(t *testing.T) {
		reqBody := RegisterRequest{
			Email:    "newuser@example.com",
			Password: "securePassword123",
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		r.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), fixedToken)
	})

	// Configure mock for login test
	mockUserStore.Users["user@example.com"] = &domain.User{
		ID:             fixedUserID,
		Email:          "user@example.com",
		HashedPassword: "hashed-password", // MockPasswordVerifier ignores this
	}

	// Test login endpoint with router
	t.Run("router_login", func(t *testing.T) {
		reqBody := LoginRequest{
			Email:    "user@example.com",
			Password: "securePassword123",
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		r.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), fixedToken)
	})

	// Configure mock for refresh token test
	mockJWTService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
		if tokenString == "valid-refresh-token" {
			return &auth.Claims{UserID: fixedUserID}, nil
		}
		return nil, auth.ErrInvalidRefreshToken
	}

	// Test refresh token endpoint with router
	t.Run("router_refresh_token", func(t *testing.T) {
		reqBody := RefreshTokenRequest{
			RefreshToken: "valid-refresh-token",
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		r.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), fixedToken)
	})
}

// We don't need this helper function as we're parsing the JSON directly in the tests

// TestAuthHandler_NewAuthHandler tests the constructor function.
func TestAuthHandler_NewAuthHandler(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore()
	mockJWTService := &mocks.MockJWTService{}
	mockPasswordVerifier := &mocks.MockPasswordVerifier{}
	authConfig := &config.AuthConfig{
		JWTSecret:                   "test-secret",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}

	t.Run("with_logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
		handler := NewAuthHandler(
			mockUserStore,
			mockJWTService,
			mockPasswordVerifier,
			authConfig,
			logger,
		)

		assert.NotNil(t, handler)
		assert.Equal(t, mockUserStore, handler.userStore)
		assert.Equal(t, mockJWTService, handler.jwtService)
		assert.Equal(t, mockPasswordVerifier, handler.passwordVerifier)
		assert.Equal(t, authConfig, handler.authConfig)
		// Validator now uses shared.Validate singleton
		assert.NotNil(t, handler.logger)
		// Check that timeFunc is set (can't compare functions directly)
		assert.NotNil(t, handler.timeFunc) // Default time function should be set
	})

	t.Run("without_logger", func(t *testing.T) {
		// Test for panic with nil logger
		assert.Panics(t, func() {
			NewAuthHandler(mockUserStore, mockJWTService, mockPasswordVerifier, authConfig, nil)
		})

	})
}
