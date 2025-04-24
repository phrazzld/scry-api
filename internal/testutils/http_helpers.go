package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreateTestServer creates a httptest server with the given handler.
// This is a simple helper to reduce boilerplate in tests.
// Automatically registers cleanup via t.Cleanup() so callers don't need to manually close the server.
func CreateTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

// CleanupResponseBody registers a cleanup function to close the response body
// to prevent resource leaks. Should be used in tests when receiving an HTTP response.
func CleanupResponseBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp != nil && resp.Body != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: failed to close response body: %v", err)
			}
		})
	}
}

// AssertErrorResponse checks that a response contains an error with the expected status code and message.
// Note: No longer registers cleanup for the response body as the request helpers handle this.
func AssertErrorResponse(
	t *testing.T,
	resp *http.Response,
	expectedStatus int,
	expectedErrorMsgPart string,
) {
	t.Helper()

	// Check status code
	assert.Equal(
		t,
		expectedStatus,
		resp.StatusCode,
		"Expected status code %d but got %d",
		expectedStatus,
		resp.StatusCode,
	)

	// For 204 No Content, body should be empty
	if expectedStatus == http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")
		assert.Empty(t, body, "Expected empty body for 204 No Content")
		return
	}

	// Read body for other status codes
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Parse error response
	var errResp shared.ErrorResponse
	err = json.Unmarshal(body, &errResp)
	require.NoError(t, err, "Failed to unmarshal error response: %s", string(body))

	// Verify error message
	assert.Contains(t, errResp.Error, expectedErrorMsgPart,
		"Error message should contain '%s' but got '%s'", expectedErrorMsgPart, errResp.Error)
}

// ExecuteInvalidJSONRequest sends a request with an invalid JSON body to test error handling.
// Automatically registers cleanup for the response body so callers don't need to manually close it.
func ExecuteInvalidJSONRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	method, path string,
) (*http.Response, error) {
	t.Helper()

	// Create request with invalid JSON body
	req, err := http.NewRequest(
		method,
		server.URL+path,
		bytes.NewBuffer(
			[]byte(`{"invalid_json": true,`),
		), // Malformed JSON (missing closing bracket)
	)
	require.NoError(t, err, "Failed to create request with invalid JSON")

	// Generate real auth token with the provided user ID
	authHeader, err := GenerateAuthHeader(userID)
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)

	// Register cleanup for the response body if the request succeeded
	if err == nil && resp != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		})
	}

	return resp, err
}

// ExecuteEmptyBodyRequest sends a request with an empty body to test validation.
// Automatically registers cleanup for the response body so callers don't need to manually close it.
func ExecuteEmptyBodyRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	method, path string,
) (*http.Response, error) {
	t.Helper()

	// Create request with empty body
	req, err := http.NewRequest(
		method,
		server.URL+path,
		bytes.NewBuffer([]byte(`{}`)), // Empty JSON object
	)
	require.NoError(t, err, "Failed to create request with empty body")

	// Generate real auth token with the provided user ID
	authHeader, err := GenerateAuthHeader(userID)
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)

	// Register cleanup for the response body if the request succeeded
	if err == nil && resp != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		})
	}

	return resp, err
}

// ExecuteCustomBodyRequest sends a request with a custom JSON body for testing.
// Automatically registers cleanup for the response body so callers don't need to manually close it.
func ExecuteCustomBodyRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	method, path string,
	body interface{},
) (*http.Response, error) {
	t.Helper()

	// Marshal the body to JSON
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err, "Failed to marshal request body")

	// Create request
	req, err := http.NewRequest(
		method,
		server.URL+path,
		bytes.NewBuffer(bodyBytes),
	)
	require.NoError(t, err, "Failed to create request")

	// Generate real auth token with the provided user ID
	authHeader, err := GenerateAuthHeader(userID)
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)

	// Register cleanup for the response body if the request succeeded
	if err == nil && resp != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		})
	}

	return resp, err
}
