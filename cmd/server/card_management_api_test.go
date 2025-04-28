//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // postgres driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEditCardEndpoint tests the PUT /cards/{id} endpoint
func TestEditCardEndpoint(t *testing.T) {
	// Initialize test database connection
	db := testutils.GetTestDBWithT(t)

	// Run tests in transaction for isolation and automatic cleanup
	testutils.WithTx(t, db, func(tx store.DBTX) {
		sqlTx, ok := tx.(*sql.Tx)
		require.True(t, ok, "Expected tx to be *sql.Tx")

		// Create a test user
		userID := createTestUser(t, sqlTx)

		// Create a test user that doesn't own the card (for forbidden test)
		otherUserID := createTestUser(t, sqlTx)

		// Create a test card owned by userID
		card := createTestCard(t, sqlTx, userID)

		// Get token for authentication
		authToken := getAuthToken(t, userID)
		otherAuthToken := getAuthToken(t, otherUserID)

		// Test cases
		tests := []struct {
			name            string
			cardID          string
			authToken       string
			requestBody     map[string]interface{}
			expectedStatus  int
			expectedMessage string
			verify          func(t *testing.T, tx store.DBTX, cardID uuid.UUID)
		}{
			{
				name:      "Success",
				cardID:    card.ID.String(),
				authToken: authToken,
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Updated question",
						"back":  "Updated answer",
					},
				},
				expectedStatus:  http.StatusNoContent,
				expectedMessage: "",
				verify: func(t *testing.T, tx store.DBTX, cardID uuid.UUID) {
					// Verify the card content and updated_at timestamp were updated
					sqlTx, ok := tx.(*sql.Tx)
					require.True(t, ok, "Expected tx to be *sql.Tx")

					updatedCard, err := getCardByID(sqlTx, cardID)
					require.NoError(t, err)

					// Decode content to verify it was updated
					var content map[string]interface{}
					err = json.Unmarshal(updatedCard.Content, &content)
					require.NoError(t, err)

					assert.Equal(t, "Updated question", content["front"])
					assert.Equal(t, "Updated answer", content["back"])

					// Verify updated_at is later than the original value
					assert.True(t, updatedCard.UpdatedAt.After(card.UpdatedAt),
						"Expected updated_at to be later than original, but got %v vs %v",
						updatedCard.UpdatedAt, card.UpdatedAt)
				},
			},
			{
				name:      "Unauthorized - No Token",
				cardID:    card.ID.String(),
				authToken: "", // No token
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Unauthorized update attempt",
						"back":  "Should fail",
					},
				},
				expectedStatus:  http.StatusUnauthorized,
				expectedMessage: "Unauthorized",
			},
			{
				name:      "Forbidden - Not Owner",
				cardID:    card.ID.String(),
				authToken: otherAuthToken, // Token from different user
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Forbidden update attempt",
						"back":  "Should fail",
					},
				},
				expectedStatus:  http.StatusForbidden,
				expectedMessage: "not owned",
			},
			{
				name:      "Not Found - Card ID Doesn't Exist",
				cardID:    uuid.New().String(), // Random non-existent ID
				authToken: authToken,
				requestBody: map[string]interface{}{
					"content": map[string]interface{}{
						"front": "Non-existent card",
						"back":  "Should fail",
					},
				},
				expectedStatus:  http.StatusNotFound,
				expectedMessage: "not found",
			},
			{
				name:        "Bad Request - Invalid JSON",
				cardID:      card.ID.String(),
				authToken:   authToken,
				requestBody: map[string]interface{}{
					// Missing required "content" field
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Validation",
			},
			{
				name:            "Bad Request - Invalid UUID",
				cardID:          "not-a-uuid",
				authToken:       authToken,
				requestBody:     map[string]interface{}{},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Invalid ID",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Create test server with real application router but using transactions
				sqlTx, ok := tx.(*sql.Tx)
				require.True(t, ok, "Expected tx to be *sql.Tx")

				server := setupCardManagementTestServer(t, sqlTx)
				defer server.Close()

				// Create request body
				var requestBody []byte
				var err error

				// Create proper request body for valid tests
				if len(tc.requestBody) > 0 {
					requestBody, err = json.Marshal(tc.requestBody)
					require.NoError(t, err)
				} else {
					// For invalid JSON test
					requestBody = []byte("{invalid-json")
				}

				// Create HTTP request
				url := fmt.Sprintf("%s/api/cards/%s", server.URL, tc.cardID)
				req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(requestBody))
				require.NoError(t, err)

				// Set headers
				req.Header.Set("Content-Type", "application/json")
				if tc.authToken != "" {
					req.Header.Set("Authorization", "Bearer "+tc.authToken)
				}

				// Send request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer func() {
					err := resp.Body.Close()
					if err != nil {
						t.Logf("Warning: failed to close response body: %v", err)
					}
				}()

				// Check status code
				assert.Equal(t, tc.expectedStatus, resp.StatusCode)

				// For successful requests (204 No Content)
				if tc.expectedStatus == http.StatusNoContent {
					// No content to verify in response body
					assert.Equal(t, 0, resp.ContentLength)

					// Run additional verification if provided
					if tc.verify != nil {
						cardID, err := uuid.Parse(tc.cardID)
						require.NoError(t, err)
						tc.verify(t, tx, cardID)
					}
					return
				}

				// For error responses, verify error message
				var errorResp map[string]string
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(t, err)

				assert.Contains(t, errorResp["error"], tc.expectedMessage)
			})
		}
	})
}

// TestDeleteCardEndpoint tests the DELETE /cards/{id} endpoint
func TestDeleteCardEndpoint(t *testing.T) {
	// Initialize test database connection
	db := testutils.GetTestDBWithT(t)

	// Run tests in transaction for isolation and automatic cleanup
	testutils.WithTx(t, db, func(tx store.DBTX) {
		sqlTx, ok := tx.(*sql.Tx)
		require.True(t, ok, "Expected tx to be *sql.Tx")

		// Create a test user
		userID := createTestUser(t, sqlTx)

		// Create a test user that doesn't own the card (for forbidden test)
		otherUserID := createTestUser(t, sqlTx)

		// Create a test card owned by userID
		card := createTestCard(t, sqlTx, userID)

		// Get token for authentication
		authToken := getAuthToken(t, userID)
		otherAuthToken := getAuthToken(t, otherUserID)

		// Test cases
		tests := []struct {
			name            string
			cardID          string
			authToken       string
			expectedStatus  int
			expectedMessage string
			verify          func(t *testing.T, tx store.DBTX, cardID uuid.UUID)
		}{
			{
				name:            "Success",
				cardID:          card.ID.String(),
				authToken:       authToken,
				expectedStatus:  http.StatusNoContent,
				expectedMessage: "",
				verify: func(t *testing.T, tx store.DBTX, cardID uuid.UUID) {
					// Verify the card was deleted
					sqlTx, ok := tx.(*sql.Tx)
					require.True(t, ok, "Expected tx to be *sql.Tx")

					// Card should not exist
					_, err := getCardByID(sqlTx, cardID)
					assert.Equal(t, store.ErrCardNotFound, err, "Expected card to be deleted")

					// User card stats should also be deleted (cascade delete)
					stats := getUserCardStats(t, sqlTx, userID, cardID)
					assert.Nil(t, stats, "Expected user card stats to be cascade deleted")
				},
			},
			{
				name:            "Unauthorized - No Token",
				cardID:          card.ID.String(),
				authToken:       "", // No token
				expectedStatus:  http.StatusUnauthorized,
				expectedMessage: "Unauthorized",
			},
			{
				name:            "Forbidden - Not Owner",
				cardID:          card.ID.String(),
				authToken:       otherAuthToken, // Token from different user
				expectedStatus:  http.StatusForbidden,
				expectedMessage: "not owned",
			},
			{
				name:            "Not Found - Card ID Doesn't Exist",
				cardID:          uuid.New().String(), // Random non-existent ID
				authToken:       authToken,
				expectedStatus:  http.StatusNotFound,
				expectedMessage: "not found",
			},
			{
				name:            "Bad Request - Invalid UUID",
				cardID:          "not-a-uuid",
				authToken:       authToken,
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Invalid ID",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Create test server with real application router but using transactions
				sqlTx, ok := tx.(*sql.Tx)
				require.True(t, ok, "Expected tx to be *sql.Tx")

				server := setupCardManagementTestServer(t, sqlTx)
				defer server.Close()

				// Create HTTP request
				url := fmt.Sprintf("%s/api/cards/%s", server.URL, tc.cardID)
				req, err := http.NewRequest(http.MethodDelete, url, nil)
				require.NoError(t, err)

				// Set headers
				if tc.authToken != "" {
					req.Header.Set("Authorization", "Bearer "+tc.authToken)
				}

				// Send request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer func() {
					err := resp.Body.Close()
					if err != nil {
						t.Logf("Warning: failed to close response body: %v", err)
					}
				}()

				// Check status code
				assert.Equal(t, tc.expectedStatus, resp.StatusCode)

				// For successful requests (204 No Content)
				if tc.expectedStatus == http.StatusNoContent {
					// No content to verify in response body
					assert.Equal(t, 0, resp.ContentLength)

					// Run additional verification if provided
					if tc.verify != nil {
						cardID, err := uuid.Parse(tc.cardID)
						require.NoError(t, err)
						tc.verify(t, tx, cardID)
					}
					return
				}

				// For error responses, verify error message
				var errorResp map[string]string
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(t, err)

				assert.Contains(t, errorResp["error"], tc.expectedMessage)
			})
		}
	})
}

// Helper functions

// createTestUser creates a user in the test database and returns the user ID
func createTestUser(t *testing.T, tx *sql.Tx) uuid.UUID {
	t.Helper()

	// Generate a unique email
	uniqueSuffix := uuid.New().String()[:8]
	email := fmt.Sprintf("test_%s@example.com", uniqueSuffix)

	// Create user
	hashedPassword := "$2a$10$oVd5DrhQBQH8iVeLsiW0De.Gx1tX38cP9jq6SxqNOILHAlVmWpYqC" // Test password hash

	userID := uuid.New()
	now := time.Now().UTC()

	// Insert user into database
	query := `
		INSERT INTO users (id, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := tx.Exec(query, userID, email, hashedPassword, now, now)
	require.NoError(t, err)

	return userID
}

// createTestCard creates a card in the test database and returns it
func createTestCard(t *testing.T, tx *sql.Tx, userID uuid.UUID) *domain.Card {
	t.Helper()

	// Create a memo for the card
	memoID := uuid.New()
	now := time.Now().UTC()

	memoQuery := `
		INSERT INTO memos (id, user_id, text, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.Exec(memoQuery, memoID, userID, "Test memo content", "pending", now, now)
	require.NoError(t, err)

	// Create the card
	cardID := uuid.New()
	content := map[string]interface{}{
		"front": "Test question",
		"back":  "Test answer",
	}
	contentBytes, err := json.Marshal(content)
	require.NoError(t, err)

	cardQuery := `
		INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(cardQuery, cardID, userID, memoID, contentBytes, now, now)
	require.NoError(t, err)

	// Create initial user card stats
	statsQuery := `
		INSERT INTO user_card_stats (
			user_id, card_id, interval, ease_factor, consecutive_correct,
			last_reviewed_at, next_review_at, review_count, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.Exec(
		statsQuery,
		userID, cardID, 1, 2.5, 0,
		now, now.Add(24*time.Hour), 0, now, now,
	)
	require.NoError(t, err)

	return &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   contentBytes,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// getCardByID retrieves a card from the database by ID
func getCardByID(tx *sql.Tx, cardID uuid.UUID) (*domain.Card, error) {
	query := `
		SELECT id, user_id, memo_id, content, created_at, updated_at
		FROM cards
		WHERE id = $1
	`

	row := tx.QueryRow(query, cardID)

	var card domain.Card
	err := row.Scan(
		&card.ID,
		&card.UserID,
		&card.MemoID,
		&card.Content,
		&card.CreatedAt,
		&card.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, store.ErrCardNotFound
	} else if err != nil {
		return nil, err
	}

	return &card, nil
}

// getAuthToken generates a valid JWT token for the given user ID
func getAuthToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()

	// Create JWT service
	_, jwtService, _, err := testutils.CreateAuthComponents(t)
	require.NoError(t, err)

	// Generate token
	token, err := jwtService.GenerateToken(context.Background(), userID)
	require.NoError(t, err)

	return token
}

// setupCardManagementTestServer creates a test server with real application router for card management tests
func setupCardManagementTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	// Create a router that uses the transaction
	router, err := setupCardManagementTestRouter(t, tx)
	require.NoError(t, err)

	// Create and return test server
	return httptest.NewServer(router)
}

// TestPostponeCardEndpoint tests the POST /cards/{id}/postpone endpoint
func TestPostponeCardEndpoint(t *testing.T) {
	// Initialize test database connection
	db := testutils.GetTestDBWithT(t)

	// Run tests in transaction for isolation and automatic cleanup
	testutils.WithTx(t, db, func(tx store.DBTX) {
		sqlTx, ok := tx.(*sql.Tx)
		require.True(t, ok, "Expected tx to be *sql.Tx")

		// Create a test user
		userID := createTestUser(t, sqlTx)

		// Create a test user that doesn't own the card (for forbidden test)
		otherUserID := createTestUser(t, sqlTx)

		// Create a test card owned by userID
		card := createTestCard(t, sqlTx, userID)

		// Get original stats for the card
		originalStats := getUserCardStats(t, sqlTx, userID, card.ID)
		require.NotNil(t, originalStats, "Expected user card stats to exist")

		// Get token for authentication
		authToken := getAuthToken(t, userID)
		otherAuthToken := getAuthToken(t, otherUserID)

		// Test cases
		tests := []struct {
			name            string
			cardID          string
			authToken       string
			requestBody     map[string]interface{}
			expectedStatus  int
			expectedMessage string
			verify          func(t *testing.T, tx store.DBTX, response map[string]interface{}, originalStats *domain.UserCardStats)
		}{
			{
				name:      "Success",
				cardID:    card.ID.String(),
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusOK,
				expectedMessage: "",
				verify: func(t *testing.T, tx store.DBTX, response map[string]interface{}, originalStats *domain.UserCardStats) {
					// Verify response contains expected fields
					assert.Equal(t, userID.String(), response["user_id"])
					assert.Equal(t, card.ID.String(), response["card_id"])

					// Get the updated stats from the database
					updatedStats := getUserCardStats(t, sqlTx, userID, card.ID)
					require.NotNil(t, updatedStats, "Expected user card stats to exist")

					// Calculate expected next review date (original + 7 days)
					expectedNextReviewAt := originalStats.NextReviewAt.AddDate(0, 0, 7)

					// Parse response next_review_at as RFC3339 string
					nextReviewAtStr, ok := response["next_review_at"].(string)
					require.True(t, ok, "Expected next_review_at to be a string")

					nextReviewAt, err := time.Parse(time.RFC3339, nextReviewAtStr)
					require.NoError(t, err, "Failed to parse next_review_at")

					// Verify next_review_at in response matches expected
					assert.WithinDuration(t, expectedNextReviewAt, nextReviewAt, time.Second,
						"Expected next_review_at to be %v, but got %v", expectedNextReviewAt, nextReviewAt)

					// Verify next_review_at in database matches expected
					assert.WithinDuration(t, expectedNextReviewAt, updatedStats.NextReviewAt, time.Second,
						"Expected next_review_at to be %v, but got %v", expectedNextReviewAt, updatedStats.NextReviewAt)

					// Verify updated_at is later than the original value
					assert.True(t, updatedStats.UpdatedAt.After(originalStats.UpdatedAt),
						"Expected updated_at to be later than original, but got %v vs %v",
						updatedStats.UpdatedAt, originalStats.UpdatedAt)
				},
			},
			{
				name:      "Bad Request - Invalid Days Parameter",
				cardID:    card.ID.String(),
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 0, // Should be >= 1
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Validation",
			},
			{
				name:        "Bad Request - Missing Days Parameter",
				cardID:      card.ID.String(),
				authToken:   authToken,
				requestBody: map[string]interface{}{
					// Missing required "days" field
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Validation",
			},
			{
				name:      "Unauthorized - No Token",
				cardID:    card.ID.String(),
				authToken: "", // No token
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusUnauthorized,
				expectedMessage: "Unauthorized",
			},
			{
				name:      "Forbidden - Not Owner",
				cardID:    card.ID.String(),
				authToken: otherAuthToken, // Token from different user
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusForbidden,
				expectedMessage: "not owned",
			},
			{
				name:      "Not Found - Card ID Doesn't Exist",
				cardID:    uuid.New().String(), // Random non-existent ID
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusNotFound,
				expectedMessage: "not found",
			},
			{
				name:      "Bad Request - Invalid UUID",
				cardID:    "not-a-uuid",
				authToken: authToken,
				requestBody: map[string]interface{}{
					"days": 7,
				},
				expectedStatus:  http.StatusBadRequest,
				expectedMessage: "Invalid ID",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Create test server with real application router but using transactions
				sqlTx, ok := tx.(*sql.Tx)
				require.True(t, ok, "Expected tx to be *sql.Tx")

				server := setupCardManagementTestServer(t, sqlTx)
				defer server.Close()

				// Create request body
				requestBody, err := json.Marshal(tc.requestBody)
				require.NoError(t, err)

				// Create HTTP request
				url := fmt.Sprintf("%s/api/cards/%s/postpone", server.URL, tc.cardID)
				req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
				require.NoError(t, err)

				// Set headers
				req.Header.Set("Content-Type", "application/json")
				if tc.authToken != "" {
					req.Header.Set("Authorization", "Bearer "+tc.authToken)
				}

				// Send request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer func() {
					err := resp.Body.Close()
					if err != nil {
						t.Logf("Warning: failed to close response body: %v", err)
					}
				}()

				// Check status code
				assert.Equal(t, tc.expectedStatus, resp.StatusCode)

				// For successful requests (200 OK)
				if tc.expectedStatus == http.StatusOK {
					// Parse response body
					var response map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&response)
					require.NoError(t, err)

					// Run additional verification if provided
					if tc.verify != nil {
						tc.verify(t, tx, response, originalStats)
					}
					return
				}

				// For error responses, verify error message
				var errorResp map[string]string
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(t, err)

				assert.Contains(t, errorResp["error"], tc.expectedMessage)
			})
		}
	})
}

// getUserCardStats retrieves user card stats from the database
// Returns nil if the stats don't exist
func getUserCardStats(t *testing.T, tx *sql.Tx, userID, cardID uuid.UUID) *domain.UserCardStats {
	t.Helper()

	query := `
		SELECT user_id, card_id, interval, ease_factor, consecutive_correct,
		       last_reviewed_at, next_review_at, review_count, created_at, updated_at
		FROM user_card_stats
		WHERE user_id = $1 AND card_id = $2
	`

	row := tx.QueryRow(query, userID, cardID)

	var stats domain.UserCardStats
	err := row.Scan(
		&stats.UserID,
		&stats.CardID,
		&stats.Interval,
		&stats.EaseFactor,
		&stats.ConsecutiveCorrect,
		&stats.LastReviewedAt,
		&stats.NextReviewAt,
		&stats.ReviewCount,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		t.Fatalf("Failed to query user card stats: %v", err)
	}

	return &stats
}

// setupCardManagementTestRouter creates a chi router configured with the application's routes and middleware
// but uses the provided transaction for database operations
func setupCardManagementTestRouter(t *testing.T, tx *sql.Tx) (http.Handler, error) {
	t.Helper()

	// Create a test database wrapper that uses the transaction
	txDB := &testutils.TxDB{Tx: tx}

	// Create auth components for JWT token validation
	authConfig, jwtService, passwordVerifier, err := testutils.CreateAuthComponents(t)
	if err != nil {
		return nil, err
	}

	// Create stores that use the transaction
	userStore := testutils.CreatePostgresUserStore(txDB)
	cardStore := testutils.CreatePostgresCardStore(txDB)
	userCardStatsStore := testutils.CreatePostgresUserCardStatsStore(txDB)

	// Create SRS service
	srsService, err := testutils.CreateSRSService()
	if err != nil {
		return nil, err
	}

	// Create card repository and stats repository adapters
	cardRepoAdapter := testutils.CreateCardRepositoryAdapter(cardStore, txDB)
	statsRepoAdapter := testutils.CreateStatsRepositoryAdapter(userCardStatsStore)

	// Create card service
	cardService, err := testutils.CreateCardService(cardRepoAdapter, statsRepoAdapter, srsService)
	if err != nil {
		return nil, err
	}

	// Create card review service
	cardReviewService, err := testutils.CreateCardReviewService(cardStore, userCardStatsStore, srsService)
	if err != nil {
		return nil, err
	}

	// Create router with standard middleware
	router := testutils.CreateTestRouter(t)

	// Create handlers
	authHandler := testutils.CreateAuthHandler(userStore, jwtService, passwordVerifier, authConfig)
	cardHandler := testutils.CreateCardHandler(cardReviewService, cardService)

	// Configure authentication middleware
	authMiddleware := testutils.CreateAuthMiddleware(jwtService)

	// Set up routes
	router.Route("/api", func(r chi.Router) {
		// Auth endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Card review endpoints
			r.Get("/cards/next", cardHandler.GetNextReviewCard)
			r.Post("/cards/{id}/answer", cardHandler.SubmitAnswer)

			// Card management endpoints (our test targets)
			r.Put("/cards/{id}", cardHandler.EditCard)
			r.Delete("/cards/{id}", cardHandler.DeleteCard)
			r.Post("/cards/{id}/postpone", cardHandler.PostponeCard)
		})
	})

	return router, nil
}
