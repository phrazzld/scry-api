//go:build integration

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetNextReviewCardIntegration tests the GET /cards/next endpoint with real dependencies
func TestGetNextReviewCardIntegration(t *testing.T) {
	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation and with a test user
	testutils.WithAuthenticatedUser(t, db, func(t *testing.T, tx *sql.Tx, auth *testutils.TestUserAuth) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create logger for testing (discards output)
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

		// Create stores with the transaction
		memoStore := postgres.NewPostgresMemoStore(tx, logger)
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create test memo
		testMemo, err := domain.NewMemo(auth.UserID, "Get Next Review Card Integration Test")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(t, memoStore.Create(ctx, testMemo), "Failed to save test memo")

		// Create test card with content
		cardContent := map[string]interface{}{
			"front": "What is the capital of France?",
			"back":  "Paris",
		}
		contentBytes, err := json.Marshal(cardContent)
		require.NoError(t, err, "Failed to marshal card content")

		card, err := domain.NewCard(auth.UserID, testMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(t, cardStore.CreateMultiple(ctx, []*domain.Card{card}), "Failed to save test card")

		// Create user card stats to make the card eligible for review
		pastTime := time.Now().UTC().Add(-24 * time.Hour) // Set review time in the past
		stats, err := domain.NewUserCardStats(auth.UserID, card.ID)
		require.NoError(t, err, "Failed to create test stats")
		stats.NextReviewAt = pastTime // Make card due for review
		require.NoError(t, statsStore.Create(ctx, stats), "Failed to save test stats")

		// Set up API server using our helper
		router := SetupAPITestServer(t, tx, APIServerOptions{
			Logger: logger,
		})

		// Create server
		server := httptest.NewServer(router)
		defer server.Close()

		// Test cases
		tests := []struct {
			name           string
			authToken      string
			expectedStatus int
			verifyResponse func(*testing.T, *http.Response)
		}{
			{
				name:           "Success",
				authToken:      auth.AuthToken,
				expectedStatus: http.StatusOK,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					// Verify response contains the expected card
					assert.Equal(t, http.StatusOK, resp.StatusCode)

					// Read and parse response body
					body, err := io.ReadAll(resp.Body)
					require.NoError(t, err, "Failed to read response body")

					var cardResp api.CardResponse
					err = json.Unmarshal(body, &cardResp)
					require.NoError(t, err, "Failed to unmarshal response")

					// Verify fields
					assert.Equal(t, card.ID.String(), cardResp.ID, "Card ID should match")
					assert.Equal(t, card.UserID.String(), cardResp.UserID, "User ID should match")
					assert.Equal(t, card.MemoID.String(), cardResp.MemoID, "Memo ID should match")

					// Verify content
					content, ok := cardResp.Content.(map[string]interface{})
					assert.True(t, ok, "Content should be a map")
					assert.Equal(t, "What is the capital of France?", content["front"], "Front content should match")
					assert.Equal(t, "Paris", content["back"], "Back content should match")
				},
			},
			{
				name:           "Unauthorized - No Token",
				authToken:      "",
				expectedStatus: http.StatusUnauthorized,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				},
			},
			{
				name:           "Unauthorized - Invalid Token",
				authToken:      "Bearer invalid-token",
				expectedStatus: http.StatusUnauthorized,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Make authenticated request using the helper
				resp, err := testutils.MakeAuthenticatedRequest(
					t,
					server,
					"GET",
					"/api/cards/next",
					nil,
					tc.authToken,
				)
				require.NoError(t, err, "Failed to execute request")

				// Verify response
				assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code should match expected")
				tc.verifyResponse(t, resp)
			})
		}

		// Test case: No cards due
		t.Run("No Cards Due", func(t *testing.T) {
			// Create another user with no due cards
			secondUser := testutils.CreateTestUserWithAuth(t, tx, "", "")

			// Make authenticated request using the new user
			resp, err := testutils.MakeAuthenticatedRequest(
				t,
				server,
				"GET",
				"/api/cards/next",
				nil,
				secondUser.AuthToken,
			)
			require.NoError(t, err, "Failed to execute request")

			// Verify response
			assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Should return 204 No Content when no cards are due")
		})
	})
}
