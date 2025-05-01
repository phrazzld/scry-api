//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockUserStore is a mock implementation of the UserStore interface for tests
type MockUserStore struct {
	Users    map[string]*domain.User
	CreateFn func(ctx context.Context, user *domain.User) error
}

// Create implements the UserStore.Create method
func (m *MockUserStore) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

// GetByID implements the UserStore.GetByID method
func (m *MockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, user := range m.Users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, store.ErrUserNotFound
}

// GetByEmail implements the UserStore.GetByEmail method
func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, exists := m.Users[email]
	if exists {
		return user, nil
	}
	return nil, store.ErrUserNotFound
}

// Update implements the UserStore.Update method
func (m *MockUserStore) Update(ctx context.Context, user *domain.User) error {
	existingUser, err := m.GetByID(ctx, user.ID)
	if err != nil {
		return err
	}
	existingUser.Email = user.Email
	existingUser.HashedPassword = user.HashedPassword
	existingUser.UpdatedAt = user.UpdatedAt
	return nil
}

// Delete implements the UserStore.Delete method
func (m *MockUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	for email, user := range m.Users {
		if user.ID == id {
			delete(m.Users, email)
			return nil
		}
	}
	return store.ErrUserNotFound
}

// WithTx implements the UserStore.WithTx method
func (m *MockUserStore) WithTx(tx *sql.Tx) store.UserStore {
	// In mock implementations, typically just return self
	return m
}

// NewMockUserStore creates a new MockUserStore with empty users map
func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		Users: make(map[string]*domain.User),
	}
}

// MockPasswordVerifier is a mock implementation of auth.PasswordVerifier
type MockPasswordVerifier struct {
	ShouldSucceed bool
}

// VerifyPassword implements the PasswordVerifier.VerifyPassword method
func (m *MockPasswordVerifier) VerifyPassword(hashedPassword string, password string) error {
	if m.ShouldSucceed {
		return nil
	}
	return errors.New("invalid password")
}

// Compare implements the PasswordVerifier.Compare method
func (m *MockPasswordVerifier) Compare(hashedPassword string, password string) error {
	return m.VerifyPassword(hashedPassword, password)
}

// MockJWTService is a mock implementation of the auth.JWTService interface
type MockJWTService struct {
	ValidateErr error
	Claims      *auth.Claims
}

// ValidateToken implements the JWTService.ValidateToken method
func (m *MockJWTService) ValidateToken(ctx context.Context, token string) (*auth.Claims, error) {
	return m.Claims, m.ValidateErr
}

// GenerateToken implements the JWTService.GenerateToken method
func (m *MockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.ValidateErr != nil {
		return "", m.ValidateErr
	}
	return "mock-token", nil
}

// ValidateRefreshToken implements the JWTService.ValidateRefreshToken method
func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, token string) (*auth.Claims, error) {
	return m.Claims, m.ValidateErr
}

// GenerateRefreshToken implements the JWTService.GenerateRefreshToken method
func (m *MockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.ValidateErr != nil {
		return "", m.ValidateErr
	}
	return "mock-refresh-token", nil
}

// TestAuthHandler_Register tests the Register handler functionality.
func TestAuthHandler_Register(t *testing.T) {
	// Define fixed values for consistent testing
	_ = time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC) // This is used for reference but not directly in code

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockUserStore, *MockJWTService, *MockPasswordVerifier)
		expectedStatus int
		expectedBody   string
		wantTokens     bool
	}{
		{
			name: "successful_registration",
			requestBody: api.RegisterRequest{
				Email:    "newuser@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// UserStore will successfully create the user
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					// Simulate storing the user
					us.Users[user.Email] = user
					return nil
				}
				// JWTService will return fixed tokens
				js.Claims = &auth.Claims{UserID: uuid.New()}
				js.ValidateErr = nil
			},
			expectedStatus: http.StatusCreated,
			wantTokens:     true,
		},
		{
			name: "invalid_request_format",
			requestBody: `{
				"email": "invalid-json
			}`,
			setupMocks:     func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation error",
			wantTokens:     false,
		},
		{
			name: "missing_required_field",
			requestBody: api.RegisterRequest{
				Email: "missing@password.com",
				// Password field intentionally omitted
			},
			setupMocks:     func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "required field",
			wantTokens:     false,
		},
		{
			name: "invalid_email_format",
			requestBody: api.RegisterRequest{
				Email:    "not-an-email",
				Password: "securePassword123",
			},
			setupMocks:     func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid email format",
			wantTokens:     false,
		},
		{
			name: "password_too_short",
			requestBody: api.RegisterRequest{
				Email:    "valid@example.com",
				Password: "short",
			},
			setupMocks:     func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "too short",
			wantTokens:     false,
		},
		{
			name: "email_already_exists",
			requestBody: api.RegisterRequest{
				Email:    "existing@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
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
			requestBody: api.RegisterRequest{
				Email:    "valid@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// Simulate database error
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					return errors.New("database connection error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to create user",
			wantTokens:     false,
		},
		{
			name: "token_generation_error",
			requestBody: api.RegisterRequest{
				Email:    "valid@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// Store user successfully
				us.CreateFn = func(ctx context.Context, user *domain.User) error {
					us.Users[user.Email] = user
					return nil
				}
				// But fail when generating token
				js.ValidateErr = errors.New("token generation failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to generate authentication tokens",
			wantTokens:     false,
		},
		{
			name: "domain_user_creation_error",
			requestBody: api.RegisterRequest{
				Email:    "valid@example.com",
				Password: "securePassword123",
			},
			setupMocks: func(us *MockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
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
			mockUserStore := NewMockUserStore()
			mockJWTService := &MockJWTService{
				ValidateErr: nil,
				Claims: &auth.Claims{
					UserID: uuid.New(),
				},
			}
			mockPasswordVerifier := &MockPasswordVerifier{
				ShouldSucceed: true,
			}

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
			handler := api.NewAuthHandler(
				mockUserStore,
				mockJWTService,
				mockPasswordVerifier,
				authConfig,
				logger,
			)

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

			// Set up router for testing
			router := chi.NewRouter()
			router.Post("/api/auth/register", handler.Register)

			// Create request and response recorder
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/register",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call the handler through the router
			router.ServeHTTP(w, req)

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
			}
		})
	}
}

// MockLoginUserStore extends MockUserStore with specific functionality for login tests
type MockLoginUserStore struct {
	*MockUserStore
	UserID          uuid.UUID
	Email           string
	HashedPassword  string
	GetByEmailError error
}

// NewLoginMockUserStore creates a new MockLoginUserStore with a predefined user
func NewLoginMockUserStore(userID uuid.UUID, email, hashedPassword string) *MockLoginUserStore {
	mockStore := &MockLoginUserStore{
		MockUserStore:  NewMockUserStore(),
		UserID:         userID,
		Email:          email,
		HashedPassword: hashedPassword,
	}

	// Add the test user
	mockStore.Users[email] = &domain.User{
		ID:             userID,
		Email:          email,
		HashedPassword: hashedPassword,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return mockStore
}

// GetByEmail overrides MockUserStore.GetByEmail for login-specific behavior
func (m *MockLoginUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetByEmailError != nil {
		return nil, m.GetByEmailError
	}
	if email == m.Email {
		return m.Users[email], nil
	}
	return nil, store.ErrUserNotFound
}

// TestAuthHandler_Login tests the Login handler functionality.
func TestAuthHandler_Login(t *testing.T) {
	// Define fixed values for consistent testing
	_ = time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC) // This is used for reference but not directly in code
	fixedUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testEmail := "user@example.com"
	testPassword := "securePassword123"
	// Generate hash dynamically instead of using hardcoded hash
	testPasswordRaw := "securePassword123"
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(testPasswordRaw), bcrypt.MinCost)
	require.NoError(t, err, "Failed to hash test password")
	hashedPassword := string(hashedBytes)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockLoginUserStore, *MockJWTService, *MockPasswordVerifier)
		expectedStatus int
		expectedBody   string
		wantTokens     bool
	}{
		{
			name: "successful_login",
			requestBody: api.LoginRequest{
				Email:    testEmail,
				Password: testPassword,
			},
			setupMocks: func(us *LoginMockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// Password comparison will succeed
				pv.ShouldSucceed = true
				// JWT validation
				js.ValidateErr = nil
				js.Claims = &auth.Claims{UserID: fixedUserID}
			},
			expectedStatus: http.StatusOK,
			wantTokens:     true,
		},
		{
			name: "invalid_request_format",
			requestBody: `{
				"email": "invalid-json
			}`,
			setupMocks:     func(us *LoginMockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation error",
			wantTokens:     false,
		},
		{
			name: "missing_required_field",
			requestBody: api.LoginRequest{
				Email: testEmail,
				// Password field intentionally omitted
			},
			setupMocks:     func(us *LoginMockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "required field",
			wantTokens:     false,
		},
		{
			name: "user_not_found",
			requestBody: api.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: testPassword,
			},
			setupMocks: func(us *LoginMockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// Override the GetByEmail function to simulate user not found
				us.GetByEmailError = store.ErrUserNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
			wantTokens:     false,
		},
		{
			name: "database_error",
			requestBody: api.LoginRequest{
				Email:    testEmail,
				Password: testPassword,
			},
			setupMocks: func(us *LoginMockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// Simulate database error
				us.GetByEmailError = errors.New("database connection error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to authenticate user",
			wantTokens:     false,
		},
		{
			name: "invalid_password",
			requestBody: api.LoginRequest{
				Email:    testEmail,
				Password: "wrongPassword",
			},
			setupMocks: func(us *LoginMockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// Password comparison will fail
				pv.ShouldSucceed = false
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Invalid credentials",
			wantTokens:     false,
		},
		{
			name: "token_generation_error",
			requestBody: api.LoginRequest{
				Email:    testEmail,
				Password: testPassword,
			},
			setupMocks: func(us *LoginMockUserStore, js *MockJWTService, pv *MockPasswordVerifier) {
				// Password check will succeed
				pv.ShouldSucceed = true
				// But token generation will fail
				js.ValidateErr = errors.New("token generation failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to generate authentication tokens",
			wantTokens:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create login-specific user store mock
			mockUserStore := NewLoginMockUserStore(fixedUserID, testEmail, hashedPassword)
			mockJWTService := &MockJWTService{}
			mockPasswordVerifier := &MockPasswordVerifier{}

			// Configure mocks based on test case
			tc.setupMocks(mockUserStore, mockJWTService, mockPasswordVerifier)

			// Create auth config
			authConfig := &config.AuthConfig{
				JWTSecret:                   "test-secret",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			}

			// Create auth handler
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := api.NewAuthHandler(
				mockUserStore,
				mockJWTService,
				mockPasswordVerifier,
				authConfig,
				logger,
			)

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

			// Set up router for testing
			router := chi.NewRouter()
			router.Post("/api/auth/login", handler.Login)

			// Create request and response recorder
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/login",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call the handler through the router
			router.ServeHTTP(w, req)

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
			}
		})
	}
}

// TestAuthHandler_RefreshToken tests the RefreshToken handler functionality.
func TestAuthHandler_RefreshToken(t *testing.T) {
	// Define fixed values for consistent testing
	_ = time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC) // This is used for reference but not directly in code
	fixedUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockJWTService)
		expectedStatus int
		expectedBody   string
		wantTokens     bool
	}{
		{
			name: "successful_token_refresh",
			requestBody: api.RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMocks: func(js *MockJWTService) {
				// ValidateRefreshToken will succeed
				js.ValidateErr = nil
				js.Claims = &auth.Claims{UserID: fixedUserID}
			},
			expectedStatus: http.StatusOK,
			wantTokens:     true,
		},
		{
			name: "invalid_request_format",
			requestBody: `{
				"refresh_token": "invalid-json
			}`,
			setupMocks:     func(js *MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation error",
			wantTokens:     false,
		},
		{
			name:        "missing_refresh_token",
			requestBody: api.RefreshTokenRequest{
				// RefreshToken field intentionally omitted
			},
			setupMocks:     func(js *MockJWTService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "required field",
			wantTokens:     false,
		},
		{
			name: "invalid_refresh_token",
			requestBody: api.RefreshTokenRequest{
				RefreshToken: "invalid-refresh-token",
			},
			setupMocks: func(js *MockJWTService) {
				// ValidateRefreshToken will fail
				js.ValidateErr = auth.ErrInvalidRefreshToken
				js.Claims = nil
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid refresh token",
			wantTokens:     false,
		},
		{
			name: "expired_refresh_token",
			requestBody: api.RefreshTokenRequest{
				RefreshToken: "expired-refresh-token",
			},
			setupMocks: func(js *MockJWTService) {
				// ValidateRefreshToken will fail
				js.ValidateErr = auth.ErrExpiredRefreshToken
				js.Claims = nil
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid refresh token",
			wantTokens:     false,
		},
		{
			name: "token_generation_error",
			requestBody: api.RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMocks: func(js *MockJWTService) {
				// ValidateRefreshToken will succeed initially
				js.Claims = &auth.Claims{UserID: fixedUserID}
				// But GenerateToken will fail on the second call
				js.ValidateErr = errors.New("token generation failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to generate new authentication tokens",
			wantTokens:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks - only need JWT service for this endpoint
			mockJWTService := &MockJWTService{}
			mockUserStore := NewMockUserStore()
			mockPasswordVerifier := &MockPasswordVerifier{}

			// Configure mocks based on test case
			tc.setupMocks(mockJWTService)

			// Create auth config
			authConfig := &config.AuthConfig{
				JWTSecret:                   "test-secret",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			}

			// Create auth handler
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := api.NewAuthHandler(
				mockUserStore,
				mockJWTService,
				mockPasswordVerifier,
				authConfig,
				logger,
			)

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

			// Set up router for testing
			router := chi.NewRouter()
			router.Post("/api/auth/refresh", handler.RefreshToken)

			// Create request and response recorder
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/refresh",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call the handler through the router
			router.ServeHTTP(w, req)

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
				if tc.name == "successful_token_refresh" {
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
				}
			}
		})
	}
}
