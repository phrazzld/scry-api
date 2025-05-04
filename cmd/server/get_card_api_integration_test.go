//go:build integration

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/phrazzld/scry-api/internal/testutils/api"
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

		// Set up API server using our standardized helper
		server := api.SetupTestServer(t, api.TestServerOptions{
			Tx:     tx,
			Logger: logger,
		})

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
					api.AssertCardResponse(t, resp, card)
				},
			},
			{
				name:           "Unauthorized - No Token",
				authToken:      "",
				expectedStatus: http.StatusUnauthorized,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					api.AssertErrorResponse(t, resp, http.StatusUnauthorized, "")
				},
			},
			{
				name:           "Unauthorized - Invalid Token",
				authToken:      "Bearer invalid-token",
				expectedStatus: http.StatusUnauthorized,
				verifyResponse: func(t *testing.T, resp *http.Response) {
					api.AssertErrorResponse(t, resp, http.StatusUnauthorized, "")
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Make authenticated request using the standardized helper
				resp, err := api.ExecuteAuthenticatedRequest(
					t,
					server,
					"GET",
					api.GetNextCardPath(),
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
			resp, err := api.ExecuteAuthenticatedRequest(
				t,
				server,
				"GET",
				api.GetNextCardPath(),
				nil,
				secondUser.AuthToken,
			)
			require.NoError(t, err, "Failed to execute request")

			// Verify response
			api.AssertResponse(t, resp, http.StatusNoContent)
		})
	})
}
