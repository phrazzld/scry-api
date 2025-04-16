package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/store"
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

// MockUserStore implements store.UserStore for testing
type MockUserStore struct {
	users           map[string]*domain.User
	lastUserID      uuid.UUID
	createError     error
	getByEmailError error
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users: make(map[string]*domain.User),
	}
}

func (m *MockUserStore) Create(ctx context.Context, user *domain.User) error {
	if m.createError != nil {
		return m.createError
	}
	if _, exists := m.users[user.Email]; exists {
		return store.ErrEmailExists
	}
	m.users[user.Email] = user
	m.lastUserID = user.ID
	return nil
}

func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailError != nil {
		return nil, m.getByEmailError
	}
	user, exists := m.users[email]
	if !exists {
		return nil, store.ErrUserNotFound
	}
	return user, nil
}

func (m *MockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}

func (m *MockUserStore) Update(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *MockUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// LoginMockUserStore is a specialized mock for login tests
type LoginMockUserStore struct {
	getByEmailError error
	userID          uuid.UUID
	userEmail       string
	hashedPassword  string
}

func NewLoginMockUserStore(userID uuid.UUID, email, hashedPassword string) *LoginMockUserStore {
	return &LoginMockUserStore{
		userID:         userID,
		userEmail:      email,
		hashedPassword: hashedPassword,
	}
}

func (m *LoginMockUserStore) Create(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *LoginMockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailError != nil {
		return nil, m.getByEmailError
	}

	if email != m.userEmail {
		return nil, store.ErrUserNotFound
	}

	return &domain.User{
		ID:             m.userID,
		Email:          m.userEmail,
		HashedPassword: m.hashedPassword,
	}, nil
}

func (m *LoginMockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}

func (m *LoginMockUserStore) Update(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *LoginMockUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestRegister(t *testing.T) {
	t.Parallel()

	// Create dependencies
	userStore := NewMockUserStore()
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
	userStore := NewLoginMockUserStore(userID, testEmail, dummyHash)

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
