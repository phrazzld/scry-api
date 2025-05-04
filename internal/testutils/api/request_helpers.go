//go:build integration

package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RequestOption is a function that configures an HTTP request.
// This follows the functional options pattern for flexible request configuration.
type RequestOption func(*http.Request)

// WithHeader adds a header to the request.
func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithContentType sets the Content-Type header.
func WithContentType(contentType string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Content-Type", contentType)
	}
}

// WithAccept sets the Accept header.
func WithAccept(accept string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Accept", accept)
	}
}

// WithAuth adds an Authorization header with the provided token.
func WithAuth(token string) RequestOption {
	return func(req *http.Request) {
		if token != "" {
			// Add "Bearer " prefix if not present
			if len(token) > 7 && token[:7] != "Bearer " {
				token = "Bearer " + token
			}
			req.Header.Set("Authorization", token)
		}
	}
}

// WithAuthForUser generates and adds an Authorization header for the given user ID.
func WithAuthForUser(t *testing.T, userID uuid.UUID) RequestOption {
	return func(req *http.Request) {
		token := GenerateAuthHeaderForTestingT(t, userID)
		req.Header.Set("Authorization", token)
	}
}

// GenerateAuthHeaderForTestingT generates an Authorization header with Bearer prefix for testing.
// This is a convenience wrapper around auth.GenerateAuthHeaderForTestingT.
func GenerateAuthHeaderForTestingT(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	return auth.GenerateAuthHeaderForTestingT(t, userID)
}

// ExecuteRequest sends an HTTP request to the given server.
// It automatically registers cleanup for the response body when the test completes.
// The options parameter allows for customizing the request with functional options.
func ExecuteRequest(
	t *testing.T,
	server *httptest.Server,
	method string,
	path string,
	body io.Reader,
	options ...RequestOption,
) (*http.Response, error) {
	t.Helper()

	// Create the request
	req, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply all options to the request
	for _, option := range options {
		option(req)
	}

	// Set Content-Type to JSON if there's a body and Content-Type isn't set already
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)

	// Register cleanup for the response body
	if err == nil && resp != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		})
	}

	return resp, err
}

// ExecuteJSONRequest sends an HTTP request with a JSON body.
// This is a convenience wrapper around ExecuteRequest for JSON payloads.
func ExecuteJSONRequest(
	t *testing.T,
	server *httptest.Server,
	method string,
	path string,
	payload interface{},
	options ...RequestOption,
) (*http.Response, error) {
	t.Helper()

	// Marshal the payload to JSON
	var bodyReader io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON payload: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	// Add content-type option for JSON
	options = append(options, WithContentType("application/json"))

	// Execute the request
	return ExecuteRequest(t, server, method, path, bodyReader, options...)
}

// ExecuteAuthenticatedRequest sends an HTTP request with authentication.
// This is a convenience wrapper around ExecuteRequest for authenticated requests.
func ExecuteAuthenticatedRequest(
	t *testing.T,
	server *httptest.Server,
	method string,
	path string,
	body io.Reader,
	authToken string,
	options ...RequestOption,
) (*http.Response, error) {
	t.Helper()

	// Add auth option
	options = append(options, WithAuth(authToken))

	// Execute the request
	return ExecuteRequest(t, server, method, path, body, options...)
}

// ExecuteAuthenticatedJSONRequest sends an HTTP request with authentication and a JSON body.
// This is a convenience wrapper combining ExecuteAuthenticatedRequest and ExecuteJSONRequest.
func ExecuteAuthenticatedJSONRequest(
	t *testing.T,
	server *httptest.Server,
	method string,
	path string,
	payload interface{},
	authToken string,
	options ...RequestOption,
) (*http.Response, error) {
	t.Helper()

	// Marshal the payload to JSON
	var bodyReader io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON payload: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	// Add auth and content-type options
	options = append(options, WithAuth(authToken), WithContentType("application/json"))

	// Execute the request
	return ExecuteRequest(t, server, method, path, bodyReader, options...)
}

// AssertResponse checks that a response has the expected status code.
func AssertResponse(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()
	assert.Equal(t, expectedStatus, resp.StatusCode, "Status code should match expected")
}

// AssertJSONResponse checks that a response has the expected status code and parses the JSON body.
// It returns the parsed body for further assertions.
func AssertJSONResponse(t *testing.T, resp *http.Response, expectedStatus int, result interface{}) {
	t.Helper()

	// Assert status code
	assert.Equal(t, expectedStatus, resp.StatusCode, "Status code should match expected")

	// For empty/no content responses, don't try to parse the body
	if expectedStatus == http.StatusNoContent || expectedStatus == http.StatusNotModified {
		return
	}

	// Read and parse the response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Skip parsing if result is nil
	if result == nil {
		return
	}

	// Parse the JSON body
	err = json.Unmarshal(body, result)
	require.NoError(t, err, "Failed to parse JSON response: %s", string(body))
}

// AssertErrorResponse checks that a response contains an error with the expected status code and message.
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
	if expectedErrorMsgPart != "" {
		assert.Contains(t, errResp.Error, expectedErrorMsgPart,
			"Error message should contain '%s' but got '%s'", expectedErrorMsgPart, errResp.Error)
	}
}

// AssertCardResponse checks that a response contains a valid card with the expected values.
func AssertCardResponse(t *testing.T, resp *http.Response, expectedCard *domain.Card) {
	t.Helper()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Parse response
	var cardResp api.CardResponse
	err = json.Unmarshal(body, &cardResp)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, expectedCard.ID.String(), cardResp.ID)
	assert.Equal(t, expectedCard.UserID.String(), cardResp.UserID)
	assert.Equal(t, expectedCard.MemoID.String(), cardResp.MemoID)

	// Check content if expected card has valid JSON content
	var expectedContent map[string]interface{}
	if err := json.Unmarshal(expectedCard.Content, &expectedContent); err == nil {
		content, ok := cardResp.Content.(map[string]interface{})
		assert.True(t, ok, "Content should be a map")

		// Check content fields
		for key, expectedValue := range expectedContent {
			assert.Equal(t, expectedValue, content[key])
		}
	}
}

// AssertStatsResponse checks that a response contains valid stats with the expected values.
func AssertStatsResponse(t *testing.T, resp *http.Response, expectedStats *domain.UserCardStats) {
	t.Helper()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Parse response
	var statsResp api.UserCardStatsResponse
	err = json.Unmarshal(body, &statsResp)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, expectedStats.UserID.String(), statsResp.UserID)
	assert.Equal(t, expectedStats.CardID.String(), statsResp.CardID)
	assert.Equal(t, expectedStats.Interval, statsResp.Interval)
	assert.Equal(t, expectedStats.EaseFactor, statsResp.EaseFactor)
	assert.Equal(t, expectedStats.ConsecutiveCorrect, statsResp.ConsecutiveCorrect)
	assert.Equal(t, expectedStats.ReviewCount, statsResp.ReviewCount)
	assert.True(t, expectedStats.LastReviewedAt.Equal(statsResp.LastReviewedAt))
	assert.True(t, expectedStats.NextReviewAt.Equal(statsResp.NextReviewAt))
}

// GetCardByID retrieves a card by ID from the database.
// This is a convenience helper for tests.
func GetCardByID(tx *sql.Tx, cardID uuid.UUID) (*domain.Card, error) {
	query := `SELECT id, user_id, memo_id, content, created_at, updated_at FROM cards WHERE id = $1`

	var card domain.Card
	err := tx.QueryRow(query, cardID).Scan(
		&card.ID,
		&card.UserID,
		&card.MemoID,
		&card.Content,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}

	return &card, nil
}

// GetAuthToken generates an auth token for the given user ID.
// This is a backward compatibility wrapper around auth.GenerateAuthHeaderForTestingT.
func GetAuthToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	return auth.GenerateAuthHeaderForTestingT(t, userID)
}

// Common request paths

// GetNextCardPath returns the path for the get next card endpoint.
func GetNextCardPath() string {
	return "/api/cards/next"
}

// GetCardPath returns the path for the get card endpoint.
func GetCardPath(cardID uuid.UUID) string {
	return fmt.Sprintf("/api/cards/%s", cardID.String())
}

// AnswerCardPath returns the path for the answer card endpoint.
func AnswerCardPath(cardID uuid.UUID) string {
	return fmt.Sprintf("/api/cards/%s/answer", cardID.String())
}

// PostponeCardPath returns the path for the postpone card endpoint.
func PostponeCardPath(cardID uuid.UUID) string {
	return fmt.Sprintf("/api/cards/%s/postpone", cardID.String())
}

// AuthRegisterPath returns the path for the auth register endpoint.
func AuthRegisterPath() string {
	return "/api/auth/register"
}

// AuthLoginPath returns the path for the auth login endpoint.
func AuthLoginPath() string {
	return "/api/auth/login"
}

// AuthRefreshPath returns the path for the auth refresh endpoint.
func AuthRefreshPath() string {
	return "/api/auth/refresh"
}
