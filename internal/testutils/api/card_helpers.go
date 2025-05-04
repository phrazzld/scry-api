//go:build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CardOption is a function type that configures a Card for testing.
type CardOption func(*domain.Card)

// WithCardID sets a specific ID for the test card.
func WithCardID(id uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.ID = id
	}
}

// WithCardUserID sets a specific user ID for the test card.
func WithCardUserID(userID uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.UserID = userID
	}
}

// WithCardMemoID sets a specific memo ID for the test card.
func WithCardMemoID(memoID uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.MemoID = memoID
	}
}

// WithCardContent sets the content for the test card using a map.
// The map will be marshaled to JSON.
func WithCardContent(content map[string]interface{}) CardOption {
	return func(c *domain.Card) {
		contentBytes, _ := json.Marshal(content)
		c.Content = contentBytes
	}
}

// WithRawCardContent sets raw JSON content for the test card.
// This allows direct setting of pre-marshaled JSON.
func WithRawCardContent(content json.RawMessage) CardOption {
	return func(c *domain.Card) {
		c.Content = content
	}
}

// WithCardCreatedAt sets the creation timestamp for the test card.
func WithCardCreatedAt(createdAt time.Time) CardOption {
	return func(c *domain.Card) {
		c.CreatedAt = createdAt
	}
}

// WithCardUpdatedAt sets the update timestamp for the test card.
func WithCardUpdatedAt(updatedAt time.Time) CardOption {
	return func(c *domain.Card) {
		c.UpdatedAt = updatedAt
	}
}

// CreateCardForAPITest creates a Card instance for API testing with default values.
// Options can be passed to customize the card.
// If t is nil, error checking for JSON marshaling is skipped.
func CreateCardForAPITest(t *testing.T, opts ...CardOption) *domain.Card {
	if t != nil {
		t.Helper()
	}

	now := time.Now().UTC()
	userID := uuid.New()
	memoID := uuid.New()
	cardID := uuid.New()

	// Default test card content
	defaultContent := map[string]interface{}{
		"front": "What is the capital of France?",
		"back":  "Paris",
	}
	contentBytes, err := json.Marshal(defaultContent)
	if t != nil {
		require.NoError(t, err)
	}

	// Create card with default values
	card := &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   contentBytes,
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now.Add(-24 * time.Hour),
	}

	// Apply options
	for _, opt := range opts {
		opt(card)
	}

	return card
}

// StatsOption is a function that configures UserCardStats for testing.
type StatsOption func(*domain.UserCardStats)

// WithStatsUserID sets a specific user ID for the test stats.
func WithStatsUserID(userID uuid.UUID) StatsOption {
	return func(s *domain.UserCardStats) {
		s.UserID = userID
	}
}

// WithStatsCardID sets a specific card ID for the test stats.
func WithStatsCardID(cardID uuid.UUID) StatsOption {
	return func(s *domain.UserCardStats) {
		s.CardID = cardID
	}
}

// WithStatsInterval sets the interval for the test stats.
func WithStatsInterval(interval int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.Interval = interval
	}
}

// WithStatsEaseFactor sets the ease factor for the test stats.
func WithStatsEaseFactor(easeFactor float64) StatsOption {
	return func(s *domain.UserCardStats) {
		s.EaseFactor = easeFactor
	}
}

// WithStatsConsecutiveCorrect sets the consecutive correct count for the test stats.
func WithStatsConsecutiveCorrect(count int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.ConsecutiveCorrect = count
	}
}

// WithStatsLastReviewedAt sets the last reviewed timestamp for the test stats.
func WithStatsLastReviewedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.LastReviewedAt = timestamp
	}
}

// WithStatsNextReviewAt sets the next review timestamp for the test stats.
func WithStatsNextReviewAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.NextReviewAt = timestamp
	}
}

// WithStatsReviewCount sets the review count for the test stats.
func WithStatsReviewCount(count int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.ReviewCount = count
	}
}

// WithStatsCreatedAt sets the creation timestamp for the test stats.
func WithStatsCreatedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.CreatedAt = timestamp
	}
}

// WithStatsUpdatedAt sets the update timestamp for the test stats.
func WithStatsUpdatedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.UpdatedAt = timestamp
	}
}

// CreateStatsForAPITest creates a UserCardStats instance for API testing with default values.
// Options can be passed to customize the stats.
// If t is nil, no test helper functionality is used.
func CreateStatsForAPITest(t *testing.T, opts ...StatsOption) *domain.UserCardStats {
	if t != nil {
		t.Helper()
	}

	now := time.Now().UTC()
	userID := uuid.New()
	cardID := uuid.New()

	// Create stats with default values
	stats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 1,
		LastReviewedAt:     now,
		NextReviewAt:       now.Add(24 * time.Hour),
		ReviewCount:        1,
		CreatedAt:          now.Add(-24 * time.Hour),
		UpdatedAt:          now,
	}

	// Apply options
	for _, opt := range opts {
		opt(stats)
	}

	return stats
}

// ExecuteGetNextCardRequest executes a GET /cards/next request against the test server.
// Automatically registers cleanup for the response body.
func ExecuteGetNextCardRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
) (*http.Response, error) {
	t.Helper()

	// Create request
	req, err := http.NewRequest("GET", server.URL+"/api/cards/next", nil)
	if err != nil {
		return nil, err
	}

	// Generate auth token with the provided user ID
	authHeader, err := GenerateAuthHeader(userID)
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Add auth header
	req.Header.Set("Authorization", authHeader)

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

// ExecuteSubmitAnswerRequest executes a POST /cards/{id}/answer request against the test server.
// Automatically registers cleanup for the response body.
func ExecuteSubmitAnswerRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	cardID uuid.UUID,
	outcome domain.ReviewOutcome,
) (*http.Response, error) {
	t.Helper()

	// Create request body
	requestBody := map[string]string{"outcome": string(outcome)}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/cards/%s/answer", server.URL, cardID.String()),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, err
	}

	// Generate auth token with the provided user ID
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

// ExecuteSubmitAnswerRequestWithRawID executes a POST /cards/{id}/answer request
// with a raw ID string (can be invalid for testing error cases).
func ExecuteSubmitAnswerRequestWithRawID(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	rawCardID string,
	outcome domain.ReviewOutcome,
) (*http.Response, error) {
	t.Helper()

	// Create request body
	requestBody := map[string]string{"outcome": string(outcome)}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// Create request with raw ID string (may be invalid)
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/cards/%s/answer", server.URL, rawCardID),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, err
	}

	// Generate auth token with the provided user ID
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

// AssertValidationError checks that a response contains a validation error with the expected information.
func AssertValidationError(
	t *testing.T,
	resp *http.Response,
	field string,
	msgPart string,
) {
	t.Helper()

	// Check status code is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"Expected status code 400 for validation error but got %d", resp.StatusCode)

	// Read and parse the response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	var errResp shared.ErrorResponse
	err = json.Unmarshal(body, &errResp)
	require.NoError(t, err, "Failed to unmarshal error response: %s", string(body))

	// Check that the error message contains the expected field
	if field != "" {
		assert.Contains(t, errResp.Error, field,
			"Error should mention field '%s' but got: %s", field, errResp.Error)
	}

	// Check that the error message contains the expected message part
	if msgPart != "" {
		assert.Contains(t, errResp.Error, msgPart,
			"Error should contain '%s' but got: %s", msgPart, errResp.Error)
	}
}

// ExecuteInvalidJSONRequest sends a request with an invalid JSON body to test error handling.
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
		fmt.Sprintf("%s%s", server.URL, path),
		bytes.NewBuffer(
			[]byte(`{"invalid_json": true,`),
		), // Malformed JSON (missing closing bracket)
	)
	require.NoError(t, err, "Failed to create request with invalid JSON")

	// Generate auth token with the provided user ID
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
		fmt.Sprintf("%s%s", server.URL, path),
		bytes.NewBuffer([]byte(`{}`)), // Empty JSON object
	)
	require.NoError(t, err, "Failed to create request with empty body")

	// Generate auth token with the provided user ID
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
		fmt.Sprintf("%s%s", server.URL, path),
		bytes.NewBuffer(bodyBytes),
	)
	require.NoError(t, err, "Failed to create request")

	// Generate auth token with the provided user ID
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

// GenerateAuthHeader creates an Authorization header value with a valid JWT token for testing.
func GenerateAuthHeader(userID uuid.UUID) (string, error) {
	// Use auth package's default config
	jwtConfig := auth.DefaultJWTConfig()
	jwtService, err := auth.NewJWTService(jwtConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT service: %w", err)
	}

	// Generate token for the user
	token, err := jwtService.GenerateToken(context.Background(), userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return "Bearer " + token, nil
}
