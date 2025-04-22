//go:build test_without_external_deps
// +build test_without_external_deps

package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/testutils"
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

	// Create a test card using the testutils helper
	card := testutils.CreateCardForAPITest(t,
		testutils.WithCardID(cardID),
		testutils.WithCardUserID(userID),
		testutils.WithCardMemoID(memoID),
		testutils.WithCardCreatedAt(now.Add(-24*time.Hour)),
		testutils.WithCardUpdatedAt(now.Add(-24*time.Hour)),
		testutils.WithCardContent(map[string]interface{}{
			"front": "What is the capital of France?",
			"back":  "Paris",
		}),
	)

	// Test cases
	tests := []struct {
		name           string
		serverOptions  testutils.CardReviewServerOptions
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Success - Card Found",
			serverOptions: testutils.CardReviewServerOptions{
				UserID:   userID,
				NextCard: card,
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name: "No Cards Due",
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				Error:  card_review.ErrNoCardsDue,
			},
			expectedStatus: http.StatusNoContent,
			expectedError:  "",
		},
		{
			name: "Unauthorized - No Valid JWT",
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
					return nil, auth.ErrInvalidToken
				},
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name: "Server Error",
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				Error:  errors.New("database error"),
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to get next review card",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the test server using the helper function
			server := testutils.SetupCardReviewTestServer(t, tc.serverOptions)
			defer server.Close()

			// Execute the request using the helper function
			resp, err := testutils.ExecuteGetNextCardRequest(t, server)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("Failed to close response body: %v", err)
				}
			}()

			// Verify the response
			if tc.expectedStatus == http.StatusOK {
				// Success case - verify card response
				testutils.AssertCardResponse(t, resp, card)
			} else {
				// Error case - verify error response
				testutils.AssertErrorResponse(t, resp, tc.expectedStatus, tc.expectedError)
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
	sampleStats := testutils.CreateStatsForAPITest(t,
		testutils.WithStatsUserID(userID),
		testutils.WithStatsCardID(cardID),
		testutils.WithStatsInterval(1),
		testutils.WithStatsEaseFactor(2.5),
		testutils.WithStatsConsecutiveCorrect(1),
		testutils.WithStatsLastReviewedAt(now),
		testutils.WithStatsNextReviewAt(now.Add(24*time.Hour)),
		testutils.WithStatsReviewCount(1),
		testutils.WithStatsCreatedAt(now.Add(-24*time.Hour)),
		testutils.WithStatsUpdatedAt(now),
	)

	// Test cases
	tests := []struct {
		name           string
		cardID         uuid.UUID
		outcome        domain.ReviewOutcome
		serverOptions  testutils.CardReviewServerOptions
		executeRequest bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "Success",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			serverOptions: testutils.CardReviewServerOptions{
				UserID:       userID,
				UpdatedStats: sampleStats,
			},
			executeRequest: true,
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:    "Card Not Found",
			cardID:  uuid.New(), // Different card ID
			outcome: domain.ReviewOutcomeGood,
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				Error:  card_review.ErrCardNotFound,
			},
			executeRequest: true,
			expectedStatus: http.StatusNotFound,
			expectedError:  "Card not found",
		},
		{
			name:    "Card Not Owned",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				Error:  card_review.ErrCardNotOwned,
			},
			executeRequest: true,
			expectedStatus: http.StatusForbidden,
			expectedError:  "You do not own this card",
		},
		{
			name:    "Invalid Answer",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				Error:  card_review.ErrInvalidAnswer,
			},
			executeRequest: true,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid answer",
		},
		{
			name:    "Invalid Card ID Format",
			cardID:  uuid.Nil, // Will be replaced with custom card ID string in the test
			outcome: domain.ReviewOutcomeGood,
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
			},
			executeRequest: false, // We'll handle this case differently
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid card ID format",
		},
		{
			name:    "Unauthorized - No Valid JWT",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
					return nil, auth.ErrInvalidToken
				},
			},
			executeRequest: true,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name:    "Server Error",
			cardID:  cardID,
			outcome: domain.ReviewOutcomeGood,
			serverOptions: testutils.CardReviewServerOptions{
				UserID: userID,
				Error:  errors.New("database error"),
			},
			executeRequest: true,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to submit answer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the test server using the helper function
			server := testutils.SetupCardReviewTestServer(t, tc.serverOptions)
			defer server.Close()

			var resp *http.Response
			var err error

			if tc.executeRequest {
				// Execute normal request using the helper function
				resp, err = testutils.ExecuteSubmitAnswerRequest(t, server, tc.cardID, tc.outcome)
			} else if tc.name == "Invalid Card ID Format" {
				// Use the helper for invalid card ID format
				resp, err = testutils.ExecuteSubmitAnswerRequestWithRawID(t, server, "not-a-uuid", tc.outcome)
			}

			require.NoError(t, err)
			defer func() {
				if resp != nil && resp.Body != nil {
					if err := resp.Body.Close(); err != nil {
						t.Errorf("Failed to close response body: %v", err)
					}
				}
			}()

			// Verify the response
			if tc.expectedStatus == http.StatusOK {
				// Success case - verify stats response
				testutils.AssertStatsResponse(t, resp, sampleStats)
			} else {
				// Error case - verify error response
				testutils.AssertErrorResponse(t, resp, tc.expectedStatus, tc.expectedError)
			}
		})
	}
}

// TestInvalidRequestBody tests submitting an invalid JSON request body
func TestInvalidRequestBody(t *testing.T) {
	// Test user and card ID
	userID := uuid.New()
	cardID := uuid.New()

	// Define test cases
	tests := []struct {
		name             string
		testType         string
		path             string
		expectedStatus   int
		expectedErrorMsg string
	}{
		{
			name:             "Submit Answer - Invalid JSON",
			testType:         "invalid-json",
			path:             "/api/cards/" + cardID.String() + "/answer",
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request format",
		},
		{
			name:             "Submit Answer - Empty Body",
			testType:         "empty-body",
			path:             "/api/cards/" + cardID.String() + "/answer",
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Validation error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test server
			server := testutils.SetupCardReviewTestServer(t, testutils.CardReviewServerOptions{
				UserID: userID,
			})
			defer server.Close()

			var resp *http.Response
			var err error

			// Use the appropriate helper based on the test type
			switch tc.testType {
			case "invalid-json":
				resp, err = testutils.ExecuteInvalidJSONRequest(t, server, "POST", tc.path)
			case "empty-body":
				resp, err = testutils.ExecuteEmptyBodyRequest(t, server, "POST", tc.path)
			default:
				t.Fatalf("Unknown test type: %s", tc.testType)
			}

			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("Failed to close response body: %v", err)
				}
			}()

			// Verify response using the helper
			testutils.AssertErrorResponse(t, resp, tc.expectedStatus, tc.expectedErrorMsg)
		})
	}
}

// errorReader is a simple io.Reader that always returns an error
type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}
