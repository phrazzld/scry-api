package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPasswordVerifier implements auth.PasswordVerifier for testing
type MockPasswordVerifier struct {
	ShouldSucceed bool
}

func (m *MockPasswordVerifier) Compare(hashedPassword, password string) error {
	if m.ShouldSucceed {
		return nil // Successful comparison
	}
	return errors.New("password mismatch") // Failed comparison
}

func TestRegister(t *testing.T) {
	t.Parallel()

	// Create dependencies
	userStore := mocks.NewMockUserStore()
	jwtService := &mocks.MockJWTService{Token: "test-token", Err: nil}
	passwordVerifier := &MockPasswordVerifier{ShouldSucceed: true}

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
				assert.Equal(t, "test-token", authResp.Token)
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
		passwordVerifier *MockPasswordVerifier
		wantStatus       int
		wantToken        bool
	}{
		{
			name: "valid login",
			payload: map[string]interface{}{
				"email":    testEmail,
				"password": testPassword,
			},
			passwordVerifier: &MockPasswordVerifier{ShouldSucceed: true},
			wantStatus:       http.StatusOK,
			wantToken:        true,
		},
		{
			name: "invalid email",
			payload: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": testPassword,
			},
			passwordVerifier: &MockPasswordVerifier{ShouldSucceed: false},
			wantStatus:       http.StatusUnauthorized,
			wantToken:        false,
		},
		{
			name: "invalid password",
			payload: map[string]interface{}{
				"email":    testEmail,
				"password": "wrongpassword",
			},
			passwordVerifier: &MockPasswordVerifier{ShouldSucceed: false},
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
				assert.Equal(t, "test-token", authResp.Token)
				// We haven't implemented ExpiresAt in Login yet, so we don't check it here
			}
		})
	}
}
