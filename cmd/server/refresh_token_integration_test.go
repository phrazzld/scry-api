//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenAPI_Integration(t *testing.T) {
	// Skip test if database is not available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Start test server with the shared database connection
	testServer := setupTestServer(t, testDB)
	defer testServer.Close()

	// Create a test user with authentication
	// Register a new user
	userEmail := "refresh-token-test@example.com"
	password := "securepassword1234"

	// Create the user using plain HTTP request
	createPayload := map[string]interface{}{
		"email":    userEmail,
		"password": password,
	}
	createBody, err := json.Marshal(createPayload)
	require.NoError(t, err)

	registerResp, err := http.Post(
		testServer.URL+"/api/auth/register",
		"application/json",
		bytes.NewBuffer(createBody),
	)
	require.NoError(t, err)
	defer func() {
		if err := registerResp.Body.Close(); err != nil {
			t.Errorf("Failed to close register response body: %v", err)
		}
	}()

	// Parse response to get refresh token
	var registerData map[string]interface{}
	err = json.NewDecoder(registerResp.Body).Decode(&registerData)
	require.NoError(t, err)

	// Get refresh token from login response
	userID := registerData["user_id"].(string)

	// Login to get a valid refresh token
	loginPayload := map[string]interface{}{
		"email":    userEmail,
		"password": password,
	}
	loginBody, err := json.Marshal(loginPayload)
	require.NoError(t, err)

	loginResp, err := http.Post(
		testServer.URL+"/api/auth/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	require.NoError(t, err)
	defer func() {
		if err := loginResp.Body.Close(); err != nil {
			t.Errorf("Failed to close login response body: %v", err)
		}
	}()

	// Parse login response to get refresh token
	var loginData map[string]interface{}
	err = json.NewDecoder(loginResp.Body).Decode(&loginData)
	require.NoError(t, err)

	validRefreshToken := loginData["refresh_token"].(string)

	// Test cases for refresh token endpoint
	testCases := []struct {
		name         string
		refreshToken string
		expectStatus int
	}{
		{
			name:         "Success - Valid refresh token",
			refreshToken: validRefreshToken,
			expectStatus: http.StatusOK,
		},
		{
			name:         "Error - Invalid refresh token format",
			refreshToken: "invalid-token-format",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error - Empty refresh token",
			refreshToken: "",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Error - Malformed refresh token",
			refreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0", // Missing signature
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request payload
			reqBody := map[string]interface{}{
				"refresh_token": tc.refreshToken,
			}
			payload, err := json.Marshal(reqBody)
			require.NoError(t, err)

			// Create and execute HTTP request
			resp, err := http.Post(
				testServer.URL+"/api/auth/refresh",
				"application/json",
				bytes.NewBuffer(payload),
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("Failed to close response body: %v", err)
				}
			}()

			// Assert the expected status code
			assert.Equal(t, tc.expectStatus, resp.StatusCode)

			if tc.expectStatus == http.StatusOK {
				// Verify successful response contains expected fields
				var response api.RefreshTokenResponse
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.NotEmpty(t, response.ExpiresAt)
			} else {
				// Verify error response format
				var errResp shared.ErrorResponse
				err = json.NewDecoder(resp.Body).Decode(&errResp)
				require.NoError(t, err)
				assert.NotEmpty(t, errResp.Error)
			}
		})
	}
}

// Test to verify validation errors in RefreshToken API
func TestRefreshTokenValidation_Integration(t *testing.T) {
	// Skip test if database is not available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Start test server with the shared database connection
	testServer := setupTestServer(t, testDB)
	defer testServer.Close()

	// Test case: Invalid JSON body
	t.Run("Invalid JSON body", func(t *testing.T) {
		// Create and execute HTTP request with invalid JSON
		resp, err := http.Post(
			testServer.URL+"/api/auth/refresh",
			"application/json",
			bytes.NewBuffer([]byte(`{"refresh_token": "test"`)), // Missing closing brace
		)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Verify response
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Verify error response mentions invalid JSON
		var errResp shared.ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(errResp.Error), "json")
	})

	// Test case: Missing required field
	t.Run("Missing refresh_token field", func(t *testing.T) {
		// Create and execute HTTP request with empty body
		resp, err := http.Post(
			testServer.URL+"/api/auth/refresh",
			"application/json",
			bytes.NewBuffer([]byte(`{}`)),
		)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Verify response
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Verify error response contains refresh_token validation error
		var errResp shared.ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "refresh_token")
	})
}

// Test to verify an expired refresh token is rejected
func TestExpiredRefreshToken_Integration(t *testing.T) {
	// Skip test if database is not available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Start test server with the shared database connection
	testServer := setupTestServer(t, testDB)
	defer testServer.Close()

	// Create a test user with authentication
	// Register a new user
	userEmail := "refresh-test@example.com"
	password := "securepassword1234"

	// Create the user using plain HTTP request
	createPayload := map[string]interface{}{
		"email":    userEmail,
		"password": password,
	}
	createBody, err := json.Marshal(createPayload)
	require.NoError(t, err)

	registerResp, err := http.Post(
		testServer.URL+"/api/auth/register",
		"application/json",
		bytes.NewBuffer(createBody),
	)
	require.NoError(t, err)
	defer func() {
		if err := registerResp.Body.Close(); err != nil {
			t.Errorf("Failed to close register response body: %v", err)
		}
	}()

	// Parse the response to get the user ID
	var registerData map[string]interface{}
	err = json.NewDecoder(registerResp.Body).Decode(&registerData)
	require.NoError(t, err)

	userID := registerData["user_id"].(string)

	// Convert user ID string to UUID
	userUUID, err := uuid.Parse(userID)
	require.NoError(t, err)

	// Create a JWT service with test configuration
	jwtService, err := auth.NewJWTService(config.AuthConfig{
		JWTSecret:                   "testsecrettestsecrettestsecrettestsecret",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	})
	require.NoError(t, err)

	// Generate an expired refresh token (expired 1 hour ago)
	expiredToken, err := jwtService.GenerateRefreshTokenWithExpiry(
		context.Background(),
		userUUID,
		time.Now().Add(-1*time.Hour),
	)
	require.NoError(t, err)

	// Create request payload with expired token
	reqBody := api.RefreshTokenRequest{
		RefreshToken: expiredToken,
	}
	payload, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// Create and execute HTTP request
	resp, err := http.Post(
		testServer.URL+"/api/auth/refresh",
		"application/json",
		bytes.NewBuffer(payload),
	)
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	// Verify response is an unauthorized error
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Verify response contains error message about invalid token
	var errResp shared.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "token")
}
