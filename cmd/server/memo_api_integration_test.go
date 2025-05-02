//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateMemoAPI_Integration tests the memo creation endpoint
func TestCreateMemoAPI_Integration(t *testing.T) {
	// Skip test if database is not available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Start test server with the shared database connection
	testServer := setupTestServer(t, testDB)
	defer testServer.Close()

	// Create a test user for authentication
	userEmail := "memo-test@example.com"
	password := "securepassword1234"

	// Register the user
	registerPayload := map[string]interface{}{
		"email":    userEmail,
		"password": password,
	}
	registerBody, err := json.Marshal(registerPayload)
	require.NoError(t, err)

	registerResp, err := http.Post(
		testServer.URL+"/api/auth/register",
		"application/json",
		bytes.NewBuffer(registerBody),
	)
	require.NoError(t, err)
	defer func() {
		if err := registerResp.Body.Close(); err != nil {
			t.Errorf("Failed to close register response body: %v", err)
		}
	}()

	// Parse registration response to get the token
	var registerData map[string]interface{}
	err = json.NewDecoder(registerResp.Body).Decode(&registerData)
	require.NoError(t, err)
	userID := registerData["user_id"].(string)
	token := registerData["token"].(string)

	t.Run("Success - Create memo", func(t *testing.T) {
		// Create request payload with valid memo text
		reqBody := map[string]interface{}{
			"text": "This is a test memo for integration testing",
		}
		payload, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create the request
		req, err := http.NewRequest(
			http.MethodPost,
			testServer.URL+"/api/memos",
			bytes.NewBuffer(payload),
		)
		require.NoError(t, err)

		// Add authorization header
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		// Execute request
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Verify successful response
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		// Parse response
		var memoResp api.MemoResponse
		err = json.NewDecoder(resp.Body).Decode(&memoResp)
		require.NoError(t, err)

		// Verify response fields
		assert.NotEmpty(t, memoResp.ID)
		assert.Equal(t, userID, memoResp.UserID)
		assert.Equal(t, reqBody["text"], memoResp.Text)
		assert.NotEmpty(t, memoResp.CreatedAt)
		assert.NotEmpty(t, memoResp.UpdatedAt)
	})

	t.Run("Error - Unauthorized", func(t *testing.T) {
		// Create request payload
		reqBody := map[string]interface{}{
			"text": "This memo should fail due to no auth",
		}
		payload, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Execute request without auth token
		resp, err := http.Post(
			testServer.URL+"/api/memos",
			"application/json",
			bytes.NewBuffer(payload),
		)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Verify unauthorized response
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Error - Empty Text", func(t *testing.T) {
		// Create request with empty text
		reqBody := map[string]interface{}{
			"text": "",
		}
		payload, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create the request
		req, err := http.NewRequest(
			http.MethodPost,
			testServer.URL+"/api/memos",
			bytes.NewBuffer(payload),
		)
		require.NoError(t, err)

		// Add authorization header
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		// Execute request
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Verify validation error
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Verify error response contains text validation error
		var errResp shared.ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "text")
	})

	t.Run("Error - Invalid JSON", func(t *testing.T) {
		// Create the request with invalid JSON
		req, err := http.NewRequest(
			http.MethodPost,
			testServer.URL+"/api/memos",
			bytes.NewBuffer([]byte(`{"text": "missing closing brace"`)),
		)
		require.NoError(t, err)

		// Add authorization header
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		// Execute request
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Verify bad request error
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Verify error response mentions invalid JSON
		var errResp shared.ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(errResp.Error), "json")
	})
}
