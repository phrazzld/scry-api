package middleware_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Create a simplified mock for testing purposes - we only need to stub ValidateToken
type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) ValidateToken(ctx context.Context, token string) (*auth.Claims, error) {
	args := m.Called(ctx, token)
	var claims *auth.Claims
	if arg := args.Get(0); arg != nil {
		claims = arg.(*auth.Claims)
	}
	return claims, args.Error(1)
}

func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, token string) (*auth.Claims, error) {
	args := m.Called(ctx, token)
	var claims *auth.Claims
	if arg := args.Get(0); arg != nil {
		claims = arg.(*auth.Claims)
	}
	return claims, args.Error(1)
}

// setupLogCapture sets up a string builder to capture logs and returns:
// 1. A function to get the captured logs
// 2. A cleanup function to restore the original logger
func setupLogCapture() (func() string, func()) {
	var logBuf strings.Builder
	handlerOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable all log levels
	}
	logger := slog.New(slog.NewTextHandler(&logBuf, handlerOpts))
	oldLogger := slog.Default()
	slog.SetDefault(logger)

	// Return a function to get the log content and a cleanup function
	return func() string {
			return logBuf.String()
		}, func() {
			slog.SetDefault(oldLogger)
		}
}

// TestAuthMiddlewareErrorRedaction verifies that the auth middleware properly redacts errors
func TestAuthMiddlewareErrorRedaction(t *testing.T) {
	// Define test cases with pairs of sensitiveErrorText and the appropriate auth error
	testCases := []struct {
		sensitiveErrorText string
		actualError        error
	}{
		{
			"token validation failed with key: AKIAIOSFODNN7EXAMPLE",
			auth.ErrInvalidToken,
		},
		{
			"invalid token format: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			auth.ErrInvalidToken,
		},
		{
			"token signature verification failed with secret: my-super-secret-key-123!",
			auth.ErrInvalidToken,
		},
		{
			"error connecting to auth database: postgres://auth_user:p4ssw0rd!@auth-db.example.com:5432/auth",
			errors.New("database connection error"),
		},
	}

	for _, tc := range testCases {
		t.Run("redacts: "+tc.sensitiveErrorText[:20]+"...", func(t *testing.T) {
			// Setup log capture
			getLogs, cleanup := setupLogCapture()
			defer cleanup()

			// Create a mock JWT service that returns a sensitive error
			mockJWTService := new(MockJWTService)

			// Wrap the actual error with our sensitive text to simulate a real-world error
			// but use the appropriate error type for handling
			wrappedErr := fmt.Errorf("%s: %w", tc.sensitiveErrorText, tc.actualError)

			// Mock the ValidateToken method with the appropriate argument types
			mockJWTService.On("ValidateToken", mock.Anything, mock.Anything).Return(nil, wrappedErr)

			// Create the middleware
			authMiddleware := middleware.NewAuthMiddleware(mockJWTService)

			// Create a test handler that just returns 200 OK
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap the test handler with our middleware
			handler := authMiddleware.Authenticate(nextHandler)

			// Create a test request with an Authorization header
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer invalid-token")

			// Create a recorder to capture the response
			recorder := httptest.NewRecorder()

			// Process the request
			handler.ServeHTTP(recorder, req)

			// Get logs
			logs := getLogs()

			// Get the appropriate expected status code for the error
			// Auth token errors should return a 401 Unauthorized
			var expectedStatus int
			if errors.Is(tc.actualError, auth.ErrInvalidToken) ||
				errors.Is(tc.actualError, auth.ErrExpiredToken) ||
				errors.Is(tc.actualError, auth.ErrExpiredRefreshToken) ||
				errors.Is(tc.actualError, auth.ErrInvalidRefreshToken) ||
				errors.Is(tc.actualError, auth.ErrWrongTokenType) {
				expectedStatus = http.StatusUnauthorized
			} else {
				expectedStatus = http.StatusInternalServerError
			}

			// Verify response is the expected status for this type of error (now handled by HandleAPIError)
			assert.Equal(t, expectedStatus, recorder.Code)

			// Verify sensitive information is not in the logs
			assert.NotContains(t, logs, "AKIAIOSFODNN7EXAMPLE", "Logs should not contain AWS keys")
			assert.NotContains(t, logs, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "Logs should not contain JWT tokens")
			assert.NotContains(t, logs, "my-super-secret-key-123", "Logs should not contain secret keys")
			assert.NotContains(t, logs, "postgres://", "Logs should not contain connection strings")
			assert.NotContains(t, logs, "p4ssw0rd", "Logs should not contain passwords")

			// Verify redaction has occurred
			if strings.Contains(tc.sensitiveErrorText, "postgres://") ||
				strings.Contains(tc.sensitiveErrorText, "p4ssw0rd") {
				assert.Contains(t, logs, "[REDACTED_CREDENTIAL]", "Logs should redact credentials")
			}

			if strings.Contains(tc.sensitiveErrorText, "AKIA") {
				assert.Contains(t, logs, "[REDACTED_KEY]", "Logs should redact keys")
			}
		})
	}
}

// TestSpecificErrorHandling tests that specific error types are handled consistently
func TestSpecificErrorHandling(t *testing.T) {
	testCases := []struct {
		name            string
		error           error
		expectedCode    int
		expectedMessage string
	}{
		{
			name:            "expired token",
			error:           auth.ErrExpiredToken,
			expectedCode:    http.StatusUnauthorized, // Updated from StatusInternalServerError to StatusUnauthorized
			expectedMessage: "Invalid token",
		},
		{
			name:            "invalid token",
			error:           auth.ErrInvalidToken,
			expectedCode:    http.StatusUnauthorized, // Updated from StatusInternalServerError to StatusUnauthorized
			expectedMessage: "Invalid token",
		},
		{
			name:            "other validation error",
			error:           errors.New("some other validation error with sensitive data: api_key=1234567890"),
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "Authentication error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup log capture
			getLogs, cleanup := setupLogCapture()
			defer cleanup()

			// Create a mock JWT service that returns the specific error
			mockJWTService := new(MockJWTService)
			mockJWTService.On("ValidateToken", mock.Anything, mock.Anything).Return(nil, tc.error)

			// Create the middleware
			authMiddleware := middleware.NewAuthMiddleware(mockJWTService)

			// Create a test handler
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap the test handler with our middleware
			handler := authMiddleware.Authenticate(nextHandler)

			// Create a test request with an Authorization header
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer test-token")

			// Create a recorder to capture the response
			recorder := httptest.NewRecorder()

			// Process the request
			handler.ServeHTTP(recorder, req)

			// Get logs
			logs := getLogs()

			// Verify response has the expected status code
			assert.Equal(t, tc.expectedCode, recorder.Code)

			// Verify no sensitive information in logs
			assert.NotContains(t, logs, "api_key=1234567890", "Logs should not contain API keys")

			// For the third case, make sure redaction happened
			if tc.name == "other validation error" {
				assert.Contains(t, logs, "[REDACTED_KEY]", "Logs should redact API keys")
			}
		})
	}
}
