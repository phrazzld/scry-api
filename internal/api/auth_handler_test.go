//go:build test || integration || test_without_external_deps

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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
)

// Test fixtures and common setup
var (
	fixedTime        = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	testUserID       = uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	testAccessToken  = "test-access-token"
	testRefreshToken = "test-refresh-token"
)

func TestNewAuthHandler(t *testing.T) {
	tests := []struct {
		name        string
		userStore   store.UserStore
		jwtService  auth.JWTService
		passwordVer auth.PasswordVerifier
		authConfig  *config.AuthConfig
		logger      *slog.Logger
		wantPanic   bool
	}{
		{
			name:        "successful creation",
			userStore:   mocks.NewMockUserStore(),
			jwtService:  auth.NewMockJWTService(),
			passwordVer: &mocks.MockPasswordVerifier{},
			authConfig:  &config.AuthConfig{},
			logger:      slog.Default(),
			wantPanic:   false,
		},
		{
			name:        "nil logger panics",
			userStore:   mocks.NewMockUserStore(),
			jwtService:  auth.NewMockJWTService(),
			passwordVer: &mocks.MockPasswordVerifier{},
			authConfig:  &config.AuthConfig{},
			logger:      nil,
			wantPanic:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() {
					NewAuthHandler(tt.userStore, tt.jwtService, tt.passwordVer, tt.authConfig, tt.logger)
				})
				return
			}

			handler := NewAuthHandler(tt.userStore, tt.jwtService, tt.passwordVer, tt.authConfig, tt.logger)
			assert.NotNil(t, handler)
			assert.Equal(t, tt.userStore, handler.userStore)
			assert.Equal(t, tt.jwtService, handler.jwtService)
			assert.Equal(t, tt.passwordVer, handler.passwordVerifier)
			assert.Equal(t, tt.authConfig, handler.authConfig)
			assert.NotNil(t, handler.timeFunc)
			assert.NotNil(t, handler.logger)
		})
	}
}

func TestAuthHandler_WithTimeFunc(t *testing.T) {
	originalHandler := NewAuthHandler(
		mocks.NewMockUserStore(),
		auth.NewMockJWTService(),
		&mocks.MockPasswordVerifier{},
		&config.AuthConfig{},
		slog.Default(),
	)

	customTimeFunc := func() time.Time { return fixedTime }
	newHandler := originalHandler.WithTimeFunc(customTimeFunc)

	// Should create a new handler instance
	assert.NotEqual(t, originalHandler, newHandler)

	// Original handler should be unchanged
	assert.NotEqual(t, fixedTime, originalHandler.timeFunc())

	// New handler should use custom time function
	assert.Equal(t, fixedTime, newHandler.timeFunc())

	// Other fields should be copied correctly
	assert.Equal(t, originalHandler.userStore, newHandler.userStore)
	assert.Equal(t, originalHandler.jwtService, newHandler.jwtService)
	assert.Equal(t, originalHandler.passwordVerifier, newHandler.passwordVerifier)
	assert.Equal(t, originalHandler.authConfig, newHandler.authConfig)
	assert.Equal(t, originalHandler.logger, newHandler.logger)
}

func TestAuthHandler_generateTokenResponse(t *testing.T) {
	tests := []struct {
		name                 string
		setupMocks           func(*auth.MockJWTService)
		authConfig           *config.AuthConfig
		userID               uuid.UUID
		expectError          bool
		expectedErrorMsg     string
		expectedExpiresAt    string
		expectedAccessToken  string
		expectedRefreshToken string
	}{
		{
			name: "successful token generation",
			setupMocks: func(jwt *auth.MockJWTService) {
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testAccessToken, nil
				}
				jwt.GenerateRefreshTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testRefreshToken, nil
				}
			},
			authConfig: &config.AuthConfig{
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			},
			userID:               testUserID,
			expectError:          false,
			expectedExpiresAt:    "2024-01-15T13:00:00Z",
			expectedAccessToken:  testAccessToken,
			expectedRefreshToken: testRefreshToken,
		},
		{
			name: "access token generation fails",
			setupMocks: func(jwt *auth.MockJWTService) {
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return "", errors.New("token generation failed")
				}
			},
			authConfig: &config.AuthConfig{
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			},
			userID:           testUserID,
			expectError:      true,
			expectedErrorMsg: "failed to generate access token",
		},
		{
			name: "refresh token generation fails",
			setupMocks: func(jwt *auth.MockJWTService) {
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testAccessToken, nil
				}
				jwt.GenerateRefreshTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return "", errors.New("refresh token generation failed")
				}
			},
			authConfig: &config.AuthConfig{
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			},
			userID:           testUserID,
			expectError:      true,
			expectedErrorMsg: "failed to generate refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockJWT := auth.NewMockJWTService()
			tt.setupMocks(mockJWT)

			handler := NewAuthHandler(
				mocks.NewMockUserStore(),
				mockJWT,
				&mocks.MockPasswordVerifier{},
				tt.authConfig,
				slog.Default(),
			).WithTimeFunc(func() time.Time { return fixedTime })

			accessToken, refreshToken, expiresAt, err := handler.generateTokenResponse(context.Background(), tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedAccessToken, accessToken)
			assert.Equal(t, tt.expectedRefreshToken, refreshToken)
			assert.Equal(t, tt.expectedExpiresAt, expiresAt)
		})
	}
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name             string
		requestBody      interface{}
		setupMocks       func(*mocks.MockUserStore, *auth.MockJWTService)
		expectedStatus   int
		expectError      bool
		validateResponse func(t *testing.T, body []byte)
	}{
		{
			name: "successful registration",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "validpassword123",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService) {
				userStore.CreateFn = func(ctx context.Context, user *domain.User) error {
					return nil
				}
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testAccessToken, nil
				}
				jwt.GenerateRefreshTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testRefreshToken, nil
				}
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
			validateResponse: func(t *testing.T, body []byte) {
				var resp AuthResponse
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp.UserID)
				assert.Equal(t, testAccessToken, resp.AccessToken)
				assert.Equal(t, testRefreshToken, resp.RefreshToken)
				assert.Equal(t, "2024-01-15T13:00:00Z", resp.ExpiresAt)
			},
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			setupMocks:     func(*mocks.MockUserStore, *auth.MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "validation fails - invalid email",
			requestBody: RegisterRequest{
				Email:    "invalid-email",
				Password: "validpassword123",
			},
			setupMocks:     func(*mocks.MockUserStore, *auth.MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "validation fails - short password",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
			},
			setupMocks:     func(*mocks.MockUserStore, *auth.MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "user creation fails - duplicate email",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "validpassword123",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService) {
				userStore.CreateError = store.ErrEmailExists
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
		{
			name: "user creation fails - database error",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "validpassword123",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService) {
				userStore.CreateError = errors.New("database connection failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "token generation fails",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "validpassword123",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService) {
				userStore.CreateFn = func(ctx context.Context, user *domain.User) error {
					return nil
				}
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return "", errors.New("token generation failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserStore := mocks.NewMockUserStore()
			mockJWT := auth.NewMockJWTService()
			tt.setupMocks(mockUserStore, mockJWT)

			handler := NewAuthHandler(
				mockUserStore,
				mockJWT,
				&mocks.MockPasswordVerifier{},
				&config.AuthConfig{
					TokenLifetimeMinutes:        60,
					RefreshTokenLifetimeMinutes: 1440,
				},
				slog.Default(),
			).WithTimeFunc(func() time.Time { return fixedTime })

			// Create request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Use context from request (no special correlation ID needed for unit tests)

			rr := httptest.NewRecorder()

			// Execute
			handler.Register(rr, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError && tt.validateResponse != nil {
				tt.validateResponse(t, rr.Body.Bytes())
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	testUser := &domain.User{
		ID:             testUserID,
		Email:          "test@example.com",
		HashedPassword: "hashed-password",
	}

	tests := []struct {
		name             string
		requestBody      interface{}
		setupMocks       func(*mocks.MockUserStore, *auth.MockJWTService, *mocks.MockPasswordVerifier)
		expectedStatus   int
		expectError      bool
		validateResponse func(t *testing.T, body []byte)
	}{
		{
			name: "successful login",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "correctpassword",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService, passVer *mocks.MockPasswordVerifier) {
				userStore.GetByEmailFn = func(ctx context.Context, email string) (*domain.User, error) {
					return testUser, nil
				}
				passVer.CompareFn = func(hashedPassword, password string) error {
					return nil
				}
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testAccessToken, nil
				}
				jwt.GenerateRefreshTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testRefreshToken, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
			validateResponse: func(t *testing.T, body []byte) {
				var resp AuthResponse
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Equal(t, testUserID, resp.UserID)
				assert.Equal(t, testAccessToken, resp.AccessToken)
				assert.Equal(t, testRefreshToken, resp.RefreshToken)
				assert.Equal(t, "2024-01-15T13:00:00Z", resp.ExpiresAt)
			},
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			setupMocks:     func(*mocks.MockUserStore, *auth.MockJWTService, *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "validation fails - invalid email",
			requestBody: LoginRequest{
				Email:    "invalid-email",
				Password: "password",
			},
			setupMocks:     func(*mocks.MockUserStore, *auth.MockJWTService, *mocks.MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "user not found",
			requestBody: LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService, passVer *mocks.MockPasswordVerifier) {
				userStore.GetByEmailError = store.ErrUserNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "incorrect password",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService, passVer *mocks.MockPasswordVerifier) {
				userStore.GetByEmailFn = func(ctx context.Context, email string) (*domain.User, error) {
					return testUser, nil
				}
				passVer.CompareFn = func(hashedPassword, password string) error {
					return errors.New("password mismatch")
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name: "database error during user lookup",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "password",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService, passVer *mocks.MockPasswordVerifier) {
				userStore.GetByEmailError = errors.New("database connection failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "token generation fails",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "correctpassword",
			},
			setupMocks: func(userStore *mocks.MockUserStore, jwt *auth.MockJWTService, passVer *mocks.MockPasswordVerifier) {
				userStore.GetByEmailFn = func(ctx context.Context, email string) (*domain.User, error) {
					return testUser, nil
				}
				passVer.CompareFn = func(hashedPassword, password string) error {
					return nil
				}
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return "", errors.New("token generation failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserStore := mocks.NewMockUserStore()
			mockJWT := auth.NewMockJWTService()
			mockPasswordVerifier := &mocks.MockPasswordVerifier{}
			tt.setupMocks(mockUserStore, mockJWT, mockPasswordVerifier)

			handler := NewAuthHandler(
				mockUserStore,
				mockJWT,
				mockPasswordVerifier,
				&config.AuthConfig{
					TokenLifetimeMinutes:        60,
					RefreshTokenLifetimeMinutes: 1440,
				},
				slog.Default(),
			).WithTimeFunc(func() time.Time { return fixedTime })

			// Create request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Use context from request (no special correlation ID needed for unit tests)

			rr := httptest.NewRecorder()

			// Execute
			handler.Login(rr, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError && tt.validateResponse != nil {
				tt.validateResponse(t, rr.Body.Bytes())
			}
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	testClaims := &auth.Claims{
		UserID: testUserID,
		ID:     "refresh-token-id",
	}

	tests := []struct {
		name             string
		requestBody      interface{}
		setupMocks       func(*auth.MockJWTService)
		expectedStatus   int
		expectError      bool
		validateResponse func(t *testing.T, body []byte)
	}{
		{
			name: "successful token refresh",
			requestBody: RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMocks: func(jwt *auth.MockJWTService) {
				jwt.ValidateRefreshTokenFunc = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return testClaims, nil
				}
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testAccessToken, nil
				}
				jwt.GenerateRefreshTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return testRefreshToken, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
			validateResponse: func(t *testing.T, body []byte) {
				var resp RefreshTokenResponse
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Equal(t, testAccessToken, resp.AccessToken)
				assert.Equal(t, testRefreshToken, resp.RefreshToken)
				assert.Equal(t, "2024-01-15T13:00:00Z", resp.ExpiresAt)
			},
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			setupMocks:     func(*auth.MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "validation fails - missing refresh token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "",
			},
			setupMocks:     func(*auth.MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid refresh token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "invalid-refresh-token",
			},
			setupMocks: func(jwt *auth.MockJWTService) {
				jwt.ValidateRefreshTokenFunc = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return nil, auth.ErrInvalidToken
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name: "token generation fails during refresh",
			requestBody: RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMocks: func(jwt *auth.MockJWTService) {
				jwt.ValidateRefreshTokenFunc = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
					return testClaims, nil
				}
				jwt.GenerateTokenFunc = func(ctx context.Context, userID uuid.UUID) (string, error) {
					return "", errors.New("token generation failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockJWT := auth.NewMockJWTService()
			tt.setupMocks(mockJWT)

			handler := NewAuthHandler(
				mocks.NewMockUserStore(),
				mockJWT,
				&mocks.MockPasswordVerifier{},
				&config.AuthConfig{
					TokenLifetimeMinutes:        60,
					RefreshTokenLifetimeMinutes: 1440,
				},
				slog.Default(),
			).WithTimeFunc(func() time.Time { return fixedTime })

			// Create request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Use context from request (no special correlation ID needed for unit tests)

			rr := httptest.NewRecorder()

			// Execute
			handler.RefreshToken(rr, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError && tt.validateResponse != nil {
				tt.validateResponse(t, rr.Body.Bytes())
			}
		})
	}
}
