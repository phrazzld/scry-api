//go:build integration || test_without_external_deps

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	api_test "github.com/phrazzld/scry-api/internal/testutils/api"
	"github.com/stretchr/testify/require"
)

// TestGetNextReviewCardAPI tests the GET /cards/next endpoint with various scenarios
func TestGetNextReviewCardAPI(t *testing.T) {

	// Test user
	userID := uuid.New()

	// Create sample card for testing
	memoID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Create a test card using the API helper
	card := api_test.CreateCardForAPITest(t,
		api_test.WithCardID(cardID),
		api_test.WithCardUserID(userID),
		api_test.WithCardMemoID(memoID),
		api_test.WithCardCreatedAt(now.Add(-24*time.Hour)),
		api_test.WithCardUpdatedAt(now.Add(-24*time.Hour)),
		api_test.WithCardContent(map[string]interface{}{
			"front": "What is the capital of France?",
			"back":  "Paris",
		}),
	)

	// Test cases
	tests := []struct {
		name           string
		setup          func(t *testing.T) *httptest.Server
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Success - Card Found",
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithNextCard(t, userID, card)
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name: "No Cards Due",
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithError(
					t,
					userID,
					card_review.ErrNoCardsDue,
				)
			},
			expectedStatus: http.StatusNoContent,
			expectedError:  "",
		},
		{
			name: "Unauthorized - No Valid JWT",
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithAuthError(
					t,
					userID,
					auth.ErrInvalidToken,
				)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name: "Server Error",
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithError(
					t,
					userID,
					errors.New("database error"),
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to get next review card",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the test server using the convenience constructor
			// Server is automatically closed via t.Cleanup()
			server := tc.setup(t)

			// Execute the request using the helper function
			// Response body is automatically closed via t.Cleanup()
			resp, err := api_test.ExecuteGetNextCardRequest(t, server, userID)
			require.NoError(t, err)

			// Verify the response
			if tc.expectedStatus == http.StatusOK {
				// Success case - verify card response
				api_test.AssertCardResponse(t, resp, card)
			} else {
				// Error case - verify error response
				api_test.AssertErrorResponse(t, resp, tc.expectedStatus, tc.expectedError)
			}
		})
	}
}

// TestSubmitAnswerAPI tests the POST /cards/{id}/answer endpoint with various scenarios
func TestSubmitAnswerAPI(t *testing.T) {

	// Test user and card
	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Create sample stats for testing
	sampleStats := api_test.CreateStatsForAPITest(t,
		api_test.WithStatsUserID(userID),
		api_test.WithStatsCardID(cardID),
		api_test.WithStatsInterval(1),
		api_test.WithStatsEaseFactor(2.5),
		api_test.WithStatsConsecutiveCorrect(1),
		api_test.WithStatsLastReviewedAt(now),
		api_test.WithStatsNextReviewAt(now.Add(24*time.Hour)),
		api_test.WithStatsReviewCount(1),
		api_test.WithStatsCreatedAt(now.Add(-24*time.Hour)),
		api_test.WithStatsUpdatedAt(now),
	)

	// Test cases
	tests := []struct {
		name           string
		cardID         uuid.UUID
		outcome        domain.ReviewOutcome
		setup          func(t *testing.T) *httptest.Server
		executeRequest bool // Special flag for the invalid card ID case
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "Success",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithUpdatedStats(t, userID, sampleStats)
			},
			executeRequest: true,
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:    "Card Not Found",
			cardID:  uuid.New(), // Different card ID
			outcome: domain.ReviewOutcomeGood,
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithError(
					t,
					userID,
					card_review.ErrCardNotFound,
				)
			},
			executeRequest: true,
			expectedStatus: http.StatusNotFound,
			expectedError:  "Card not found",
		},
		{
			name:    "Card Not Owned",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithError(
					t,
					userID,
					card_review.ErrCardNotOwned,
				)
			},
			executeRequest: true,
			expectedStatus: http.StatusForbidden,
			expectedError:  "You do not own this card",
		},
		{
			name:    "Invalid Answer",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithError(
					t,
					userID,
					card_review.ErrInvalidAnswer,
				)
			},
			executeRequest: true,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid answer",
		},
		{
			name:    "Invalid Card ID Format",
			cardID:  uuid.Nil, // Will be replaced with custom card ID string in the test
			outcome: domain.ReviewOutcomeGood,
			setup: func(t *testing.T) *httptest.Server {
				// Use empty server since error will happen when parsing the ID
				return api_test.SetupCardReviewTestServerWithNextCard(t, userID, nil)
			},
			executeRequest: false, // We'll handle this case differently
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid ID",
		},
		{
			name:    "Unauthorized - No Valid JWT",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithAuthError(
					t,
					userID,
					auth.ErrInvalidToken,
				)
			},
			executeRequest: true,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name:    "Server Error",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			setup: func(t *testing.T) *httptest.Server {
				return api_test.SetupCardReviewTestServerWithError(
					t,
					userID,
					errors.New("database error"),
				)
			},
			executeRequest: true,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to submit answer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the test server using the convenience constructor
			// Server is automatically closed via t.Cleanup()
			server := tc.setup(t)

			var resp *http.Response
			var err error

			if tc.executeRequest {
				// Execute normal request using the helper function
				resp, err = api_test.ExecuteSubmitAnswerRequest(
					t,
					server,
					userID,
					tc.cardID,
					tc.outcome,
				)
			} else if tc.name == "Invalid Card ID Format" {
				// Use the helper for invalid card ID format
				resp, err = api_test.ExecuteSubmitAnswerRequestWithRawID(t, server, userID, "not-a-uuid", tc.outcome)
			}

			require.NoError(t, err)
			// Response body is automatically closed via t.Cleanup() now

			// Verify the response
			if tc.expectedStatus == http.StatusOK {
				// Success case - verify stats response
				api_test.AssertStatsResponse(t, resp, sampleStats)
			} else {
				// Error case - verify error response
				api_test.AssertErrorResponse(t, resp, tc.expectedStatus, tc.expectedError)
			}
		})
	}
}

// TestInvalidRequestBody tests submitting an invalid JSON request body
// Note: Different validation error types produce different error messages:
// - Invalid JSON syntax - Returns the generic "Validation error" message
// - Empty body with missing required fields - Returns specific field validation errors
// - Invalid field values - Returns field-specific validation errors with details
func TestInvalidRequestBody(t *testing.T) {

	// Test user and card ID
	userID := uuid.New()
	cardID := uuid.New()

	// Create sample stats for tests that reach the service layer
	sampleStats := api_test.CreateStatsForAPITest(t,
		api_test.WithStatsUserID(userID),
		api_test.WithStatsCardID(cardID),
	)

	// Define test cases
	tests := []struct {
		name       string
		testType   string
		path       string
		payload    map[string]interface{} // For custom payload tests
		validation struct {
			field   string
			msgPart string
		}
		expectSuccess bool // For cases that should succeed rather than return validation errors
	}{
		// Basic required field validation
		{
			name:     "Submit Answer - Invalid JSON",
			testType: "invalid-json",
			path:     "/api/cards/" + cardID.String() + "/answer",
			validation: struct {
				field   string
				msgPart string
			}{
				field:   "", // No specific field for malformed JSON
				msgPart: "Validation error",
			},
		},
		{
			name:     "Submit Answer - Empty Body",
			testType: "empty-body",
			path:     "/api/cards/" + cardID.String() + "/answer",
			validation: struct {
				field   string
				msgPart string
			}{
				field:   "Outcome",
				msgPart: "required field",
			},
		},

		// Enum validation tests (oneof)
		{
			name:     "Submit Answer - Invalid Outcome Value",
			testType: "custom",
			path:     "/api/cards/" + cardID.String() + "/answer",
			payload: map[string]interface{}{
				"outcome": "medium", // Not in allowed enum: "again", "hard", "good", "easy"
			},
			validation: struct {
				field   string
				msgPart string
			}{
				field:   "Outcome",
				msgPart: "invalid value",
			},
		},
		{
			name:     "Submit Answer - Outcome With Incorrect Case",
			testType: "custom",
			path:     "/api/cards/" + cardID.String() + "/answer",
			payload: map[string]interface{}{
				"outcome": "GOOD", // Correct value but wrong case (case-sensitive)
			},
			validation: struct {
				field   string
				msgPart string
			}{
				field:   "Outcome",
				msgPart: "invalid value",
			},
		},

		// Edge cases with empty strings and non-string values
		{
			name:     "Submit Answer - Empty String Outcome",
			testType: "custom",
			path:     "/api/cards/" + cardID.String() + "/answer",
			payload: map[string]interface{}{
				"outcome": "", // Empty string
			},
			validation: struct {
				field   string
				msgPart string
			}{
				field:   "Outcome",
				msgPart: "required field", // Empty strings fail the "required" validation
			},
		},
		{
			name:     "Submit Answer - Numeric Outcome",
			testType: "custom",
			path:     "/api/cards/" + cardID.String() + "/answer",
			payload: map[string]interface{}{
				"outcome": 5, // Number instead of string
			},
			validation: struct {
				field   string
				msgPart string
			}{
				field:   "", // This fails JSON type validation, not field validation
				msgPart: "Validation error",
			},
		},

		// Test with nearly valid UUID
		{
			name:     "Submit Answer - Nearly Valid UUID",
			testType: "nearly-valid-uuid",
			path:     "/api/cards/almost-valid-uuid", // Not an API path with {id} parameter
			validation: struct {
				field   string
				msgPart string
			}{
				field:   "",
				msgPart: "Invalid ID",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// For most tests, we just need a server that accepts requests
			var server *httptest.Server
			if tc.expectSuccess {
				// For tests expecting success, set up a server that returns stats
				server = api_test.SetupCardReviewTestServerWithUpdatedStats(t, userID, sampleStats)
			} else {
				// For validation tests, the service behavior doesn't matter
				server = api_test.SetupCardReviewTestServerWithNextCard(t, userID, nil)
			}

			var resp *http.Response
			var err error

			// Use the appropriate helper based on the test type
			// Response body is automatically closed via t.Cleanup() now
			switch tc.testType {
			case "invalid-json":
				resp, err = api_test.ExecuteInvalidJSONRequest(t, server, userID, "POST", tc.path)
			case "empty-body":
				resp, err = api_test.ExecuteEmptyBodyRequest(t, server, userID, "POST", tc.path)
			case "custom":
				resp, err = api_test.ExecuteCustomBodyRequest(t, server, userID, "POST", tc.path, tc.payload)
			case "nearly-valid-uuid":
				resp, err = api_test.ExecuteSubmitAnswerRequestWithRawID(
					t, server, userID, "almost-valid-uuid",
					domain.ReviewOutcomeGood,
				)
			default:
				t.Fatalf("Unknown test type: %s", tc.testType)
			}

			require.NoError(t, err)

			// Handle special cases where we expect success
			if tc.expectSuccess {
				api_test.AssertStatsResponse(t, resp, sampleStats)
				return
			}

			// Verify response using the validation error helper
			api_test.AssertValidationError(t, resp, tc.validation.field, tc.validation.msgPart)
		})
	}

	// Create a separate test for Additional Fields - this is a valid request
	// that should pass validation and reach the service layer
	t.Run("Submit Answer - Valid With Extra Fields", func(t *testing.T) {
		// Create stats to be returned by the mock service
		stats := api_test.CreateStatsForAPITest(t,
			api_test.WithStatsUserID(userID),
			api_test.WithStatsCardID(cardID),
		)

		// Set up a server that returns the stats for a successful answer submission
		server := api_test.SetupCardReviewTestServerWithUpdatedStats(t, userID, stats)

		// Payload with valid outcome and extra fields that should be ignored
		payload := map[string]interface{}{
			"outcome":    "good",              // Valid outcome
			"extra_data": "should be ignored", // Extra field that should be ignored
		}

		// Execute the request
		resp, err := api_test.ExecuteCustomBodyRequest(t, server, userID, "POST",
			"/api/cards/"+cardID.String()+"/answer", payload)
		require.NoError(t, err)

		// Verify successful response
		api_test.AssertStatsResponse(t, resp, stats)
	})
}
