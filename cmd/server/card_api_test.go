//go:build integration || test_without_external_deps

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data creation helpers

// setupTestUser creates a test user in the database
func setupTestUser(t *testing.T, ctx context.Context, tx *sql.Tx, email string) *domain.User {
	userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
	user, err := domain.NewUser(email, "password123456")
	require.NoError(t, err, "Failed to create test user")
	require.NoError(t, userStore.Create(ctx, user), "Failed to save test user")
	return user
}

// setupTestMemo creates a test memo in the database
func setupTestMemo(t *testing.T, ctx context.Context, tx *sql.Tx, userID uuid.UUID, title string) *domain.Memo {
	memoStore := postgres.NewPostgresMemoStore(tx, nil)
	memo, err := domain.NewMemo(userID, title)
	require.NoError(t, err, "Failed to create test memo")
	require.NoError(t, memoStore.Create(ctx, memo), "Failed to save test memo")
	return memo
}

// setupTestCard creates a test card in the database
func setupTestCard(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	userID, memoID uuid.UUID,
	content map[string]interface{},
) *domain.Card {
	cardStore := postgres.NewPostgresCardStore(tx, nil)
	contentBytes, err := json.Marshal(content)
	require.NoError(t, err, "Failed to marshal card content")

	card, err := domain.NewCard(userID, memoID, contentBytes)
	require.NoError(t, err, "Failed to create test card")
	require.NoError(t, cardStore.CreateMultiple(ctx, []*domain.Card{card}), "Failed to save test card")
	return card
}

// setupTestUserCardStats creates test stats in the database
func setupTestUserCardStats(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	userID, cardID uuid.UUID,
	nextReviewAt time.Time,
) *domain.UserCardStats {
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
	stats, err := domain.NewUserCardStats(userID, cardID)
	require.NoError(t, err, "Failed to create test stats")
	stats.NextReviewAt = nextReviewAt
	require.NoError(t, statsStore.Create(ctx, stats), "Failed to save test stats")
	return stats
}

// TestCardEditIntegration tests the PUT /cards/{id} endpoint with real dependencies
func TestCardEditIntegration(t *testing.T) {
	// Skip test if database connection not available
	if testdb.ShouldSkipDatabaseTest() {
		t.Skip("DATABASE_URL or SCRY_TEST_DB_URL not set - skipping integration test")
	}

	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create test data
		testUser := setupTestUser(t, ctx, tx, "edit-card-integration-test@example.com")
		otherUser := setupTestUser(t, ctx, tx, "other-user-edit-card@example.com")

		testMemo := setupTestMemo(t, ctx, tx, testUser.ID, "Edit Card Integration Test")
		otherUserMemo := setupTestMemo(t, ctx, tx, otherUser.ID, "Other User Memo")

		cardContent := map[string]interface{}{
			"front": "Initial question",
			"back":  "Initial answer",
		}

		card := setupTestCard(t, ctx, tx, testUser.ID, testMemo.ID, cardContent)
		otherUserCard := setupTestCard(t, ctx, tx, otherUser.ID, otherUserMemo.ID, cardContent)

		// Store references for verification
		cardStore := postgres.NewPostgresCardStore(tx, nil)

		// Set up test server
		server := api.SetupCardManagementTestServer(t, tx)

		// Set up tests with different scenarios
		tests := []struct {
			name           string
			userID         uuid.UUID
			cardID         uuid.UUID
			requestContent map[string]interface{}
			expectedStatus int
		}{
			{
				name:   "Success",
				userID: testUser.ID,
				cardID: card.ID,
				requestContent: map[string]interface{}{
					"front": "Updated question",
					"back":  "Updated answer",
				},
				expectedStatus: http.StatusNoContent,
			},
			{
				name:   "Card Not Found",
				userID: testUser.ID,
				cardID: uuid.New(), // Non-existent card
				requestContent: map[string]interface{}{
					"front": "Updated question",
					"back":  "Updated answer",
				},
				expectedStatus: http.StatusNotFound,
			},
			{
				name:   "Not Card Owner",
				userID: testUser.ID,
				cardID: otherUserCard.ID, // Card owned by other user
				requestContent: map[string]interface{}{
					"front": "Updated question",
					"back":  "Updated answer",
				},
				expectedStatus: http.StatusForbidden,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Create request body
				requestBody := map[string]interface{}{
					"content": tc.requestContent,
				}

				// Build the path
				path := api.GetCardPath(tc.cardID)

				// Execute request
				resp, err := api.ExecuteAuthenticatedJSONRequest(
					t,
					server,
					http.MethodPut,
					path,
					requestBody,
					api.GetAuthToken(t, tc.userID),
				)
				require.NoError(t, err, "Failed to execute request")

				// Verify response status code
				assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code should match expected")

				// For success case, verify the card was actually updated in the database
				if tc.name == "Success" {
					updatedCard, err := cardStore.GetByID(ctx, tc.cardID)
					require.NoError(t, err, "Failed to get updated card")

					var updatedContent map[string]interface{}
					err = json.Unmarshal(updatedCard.Content, &updatedContent)
					require.NoError(t, err, "Failed to unmarshal updated content")

					assert.Equal(t, "Updated question", updatedContent["front"], "Card front should be updated")
					assert.Equal(t, "Updated answer", updatedContent["back"], "Card back should be updated")
				}
			})
		}
	})
}

// TestCardDeleteIntegration tests the DELETE /cards/{id} endpoint with real dependencies
func TestCardDeleteIntegration(t *testing.T) {
	// Skip test if database connection not available
	if testdb.ShouldSkipDatabaseTest() {
		t.Skip("DATABASE_URL or SCRY_TEST_DB_URL not set - skipping integration test")
	}

	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create test data
		testUser := setupTestUser(t, ctx, tx, "delete-card-integration-test@example.com")
		otherUser := setupTestUser(t, ctx, tx, "other-user-delete-card@example.com")

		testMemo := setupTestMemo(t, ctx, tx, testUser.ID, "Delete Card Integration Test")
		otherUserMemo := setupTestMemo(t, ctx, tx, otherUser.ID, "Other User Memo")

		cardContent := map[string]interface{}{
			"front": "Test question",
			"back":  "Test answer",
		}

		card := setupTestCard(t, ctx, tx, testUser.ID, testMemo.ID, cardContent)
		otherUserCard := setupTestCard(t, ctx, tx, otherUser.ID, otherUserMemo.ID, cardContent)

		// Store references for verification
		cardStore := postgres.NewPostgresCardStore(tx, nil)

		// Set up test server
		server := api.SetupCardManagementTestServer(t, tx)

		// Set up tests with different scenarios
		tests := []struct {
			name           string
			userID         uuid.UUID
			cardID         uuid.UUID
			expectedStatus int
			checkDeleted   bool
		}{
			{
				name:           "Success",
				userID:         testUser.ID,
				cardID:         card.ID,
				expectedStatus: http.StatusNoContent,
				checkDeleted:   true,
			},
			{
				name:           "Card Not Found",
				userID:         testUser.ID,
				cardID:         uuid.New(), // Non-existent card
				expectedStatus: http.StatusNotFound,
				checkDeleted:   false,
			},
			{
				name:           "Not Card Owner",
				userID:         testUser.ID,
				cardID:         otherUserCard.ID, // Card owned by other user
				expectedStatus: http.StatusForbidden,
				checkDeleted:   false,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Build the path
				path := api.GetCardPath(tc.cardID)

				// Execute request
				resp, err := api.ExecuteAuthenticatedJSONRequest(
					t,
					server,
					http.MethodDelete,
					path,
					nil, // No body for DELETE
					api.GetAuthToken(t, tc.userID),
				)
				require.NoError(t, err, "Failed to execute request")

				// Verify response status code
				assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code should match expected")

				// For success case, verify the card was actually deleted from the database
				if tc.checkDeleted {
					_, err := cardStore.GetByID(ctx, tc.cardID)
					assert.Error(t, err, "Card should be deleted")
				}
			})
		}
	})
}

// TestCardPostponeIntegration tests the POST /cards/{id}/postpone endpoint with real dependencies
func TestCardPostponeIntegration(t *testing.T) {
	// Skip test if database connection not available
	if testdb.ShouldSkipDatabaseTest() {
		t.Skip("DATABASE_URL or SCRY_TEST_DB_URL not set - skipping integration test")
	}

	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create test data
		testUser := setupTestUser(t, ctx, tx, "postpone-card-integration-test@example.com")
		otherUser := setupTestUser(t, ctx, tx, "other-user-postpone-card@example.com")

		testMemo := setupTestMemo(t, ctx, tx, testUser.ID, "Postpone Card Integration Test")
		otherUserMemo := setupTestMemo(t, ctx, tx, otherUser.ID, "Other User Memo")

		cardContent := map[string]interface{}{
			"front": "Test question",
			"back":  "Test answer",
		}

		card := setupTestCard(t, ctx, tx, testUser.ID, testMemo.ID, cardContent)
		otherUserCard := setupTestCard(t, ctx, tx, otherUser.ID, otherUserMemo.ID, cardContent)

		// Create user card stats with a next review date
		stats := setupTestUserCardStats(t, ctx, tx, testUser.ID, card.ID, time.Now().UTC().Add(24*time.Hour))

		// Store references for verification
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)

		// Set up test server
		server := api.SetupCardManagementTestServer(t, tx)

		// Set up tests with different scenarios
		tests := []struct {
			name           string
			userID         uuid.UUID
			cardID         uuid.UUID
			days           int
			expectedStatus int
		}{
			{
				name:           "Success",
				userID:         testUser.ID,
				cardID:         card.ID,
				days:           7,
				expectedStatus: http.StatusOK,
			},
			{
				name:           "Card Not Found",
				userID:         testUser.ID,
				cardID:         uuid.New(), // Non-existent card
				days:           7,
				expectedStatus: http.StatusNotFound,
			},
			{
				name:           "Not Card Owner",
				userID:         testUser.ID,
				cardID:         otherUserCard.ID, // Card owned by other user
				days:           7,
				expectedStatus: http.StatusForbidden,
			},
			{
				name:           "Invalid Days Value",
				userID:         testUser.ID,
				cardID:         card.ID,
				days:           -1, // Invalid days value
				expectedStatus: http.StatusBadRequest,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Create request body
				requestBody := map[string]interface{}{
					"days": tc.days,
				}

				// Build the path
				path := api.PostponeCardPath(tc.cardID)

				// Execute request
				resp, err := api.ExecuteAuthenticatedJSONRequest(
					t,
					server,
					http.MethodPost,
					path,
					requestBody,
					api.GetAuthToken(t, tc.userID),
				)
				require.NoError(t, err, "Failed to execute request")

				// Verify response status code
				assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code should match expected")

				// For success case, verify the card's next review date was actually postponed
				if tc.name == "Success" {
					updatedStats, err := statsStore.Get(ctx, tc.userID, tc.cardID)
					require.NoError(t, err, "Failed to get updated stats")

					// Calculate expected next review date (approximately)
					expectedNextReview := stats.NextReviewAt.AddDate(0, 0, tc.days)

					// Allow for small time differences (within 1 minute)
					timeDiff := updatedStats.NextReviewAt.Sub(expectedNextReview)
					assert.Less(
						t,
						timeDiff.Abs(),
						time.Minute,
						"Next review date should be postponed by the specified days",
					)
				}
			})
		}
	})
}
