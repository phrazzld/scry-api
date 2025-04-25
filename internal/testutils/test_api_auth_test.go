package testutils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealJWTWithMiddleware verifies that our test JWT service works correctly
// with the API middleware
func TestRealJWTWithMiddleware(t *testing.T) {
	// Create test JWT service
	jwtService := NewTestJWTService()

	// Create auth middleware using the JWT service
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create a test router
	r := chi.NewRouter()

	// Define a simple protected handler
	var capturedUserID uuid.UUID
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
		if !ok {
			http.Error(w, "User ID not found in context", http.StatusInternalServerError)
			return
		}
		capturedUserID = userID
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
			t.Logf("Error encoding response: %v", err)
		}
	})

	// Set up routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Get("/protected", protectedHandler)
	})

	// Create test server
	server := httptest.NewServer(r)
	defer server.Close()

	tests := []struct {
		name           string
		setupAuth      func() string
		expectedStatus int
	}{
		{
			name: "Valid token",
			setupAuth: func() string {
				userID := uuid.New()
				token, err := jwtService.GenerateToken(context.Background(), userID)
				require.NoError(t, err)
				capturedUserID = uuid.Nil // Reset
				return "Bearer " + token
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Missing token",
			setupAuth: func() string {
				return ""
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid token format",
			setupAuth: func() string {
				return "Bearer invalid-token"
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Wrong auth header format",
			setupAuth: func() string {
				userID := uuid.New()
				token, err := jwtService.GenerateToken(context.Background(), userID)
				require.NoError(t, err)
				return "Token " + token // Wrong prefix
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	// Execute tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the request
			req, err := http.NewRequest("GET", server.URL+"/protected", nil)
			require.NoError(t, err)

			// Set auth header if provided
			authHeader := tt.setupAuth()
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// If success, verify user ID was extracted
			if tt.expectedStatus == http.StatusOK {
				assert.NotEqual(
					t,
					uuid.Nil,
					capturedUserID,
					"User ID should be extracted from token",
				)
			}
		})
	}
}

// TestGenerateAuthHeader_with_middleware ensures the GenerateAuthHeader helper
// produces tokens that work with the middleware
func TestGenerateAuthHeader_with_middleware(t *testing.T) {
	// Create auth middleware with our JWT service
	jwtService := NewTestJWTService()
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create router with protected endpoint
	r := chi.NewRouter()
	var extractedUserID uuid.UUID

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
			if !ok {
				http.Error(w, "User ID not found in context", http.StatusInternalServerError)
				return
			}
			extractedUserID = userID
			w.WriteHeader(http.StatusOK)
		})
	})

	// Create test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Generate a user ID and auth header
	userID := uuid.New()
	authHeader, err := GenerateAuthHeader(userID)
	require.NoError(t, err)

	// Make request with auth header
	req, err := http.NewRequest("GET", server.URL+"/protected", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", authHeader)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	// Verify success
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, userID, extractedUserID, "User ID from token should match original")
}
