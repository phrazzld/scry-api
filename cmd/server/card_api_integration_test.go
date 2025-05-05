//go:build integration || test_without_external_deps

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCardEditIntegration tests the PUT /cards/{id} endpoint with real dependencies
func TestCardEditIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create stores with the transaction
		logger := slog.Default()
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		memoStore := postgres.NewPostgresMemoStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create test user
		testUser, err := domain.NewUser("edit-card-integration-test@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(t, userStore.Create(ctx, testUser), "Failed to save test user")

		// Create another user for ownership test
		otherUser, err := domain.NewUser("other-user-edit-card@example.com", "password123")
		require.NoError(t, err, "Failed to create other user")
		require.NoError(t, userStore.Create(ctx, otherUser), "Failed to save other user")

		// Create test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Edit Card Integration Test")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(t, memoStore.Create(ctx, testMemo), "Failed to save test memo")

		// Create test card with content
		initialContent := map[string]interface{}{
			"front": "Initial question",
			"back":  "Initial answer",
		}
		contentBytes, err := json.Marshal(initialContent)
		require.NoError(t, err, "Failed to marshal card content")

		card, err := domain.NewCard(testUser.ID, testMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(t, cardStore.CreateMultiple(ctx, []*domain.Card{card}), "Failed to save test card")

		// Create card owned by other user for ownership test
		otherUserMemo, err := domain.NewMemo(otherUser.ID, "Other User Memo")
		require.NoError(t, err, "Failed to create other user memo")
		require.NoError(t, memoStore.Create(ctx, otherUserMemo), "Failed to save other user memo")

		otherUserCard, err := domain.NewCard(otherUser.ID, otherUserMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create other user card")
		require.NoError(
			t,
			cardStore.CreateMultiple(ctx, []*domain.Card{otherUserCard}),
			"Failed to save other user card",
		)

		// Create SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err, "Failed to create SRS service")

		// Create repository adapters for service layer
		dbConn := cardStore.DB()
		cardRepo := service.NewCardRepositoryAdapter(cardStore, dbConn)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create services using the adapters
		cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
		require.NoError(t, err, "Failed to create card service")

		cardReviewService, err := card_review.NewCardReviewService(cardStore, statsStore, srsService, logger)
		require.NoError(t, err, "Failed to create card review service")

		// Create JWT service for auth
		jwtService, err := auth.NewTestJWTService()
		require.NoError(t, err, "Failed to create JWT service")

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
				// Create router with standard middleware
				router := chi.NewRouter()
				router.Use(chimiddleware.RequestID)
				router.Use(chimiddleware.RealIP)
				router.Use(chimiddleware.Recoverer)

				// Create auth middleware
				authMiddleware := middleware.NewAuthMiddleware(jwtService)

				// Create card handler with real services
				cardHandler := api.NewCardHandler(cardReviewService, cardService, logger)

				// Set up API routes
				router.Route("/api", func(r chi.Router) {
					r.Group(func(r chi.Router) {
						r.Use(authMiddleware.Authenticate)
						r.Put("/cards/{id}", cardHandler.EditCard)
					})
				})

				// Create server
				server := httptest.NewServer(router)
				defer server.Close()

				// Create request body
				requestBody := map[string]interface{}{
					"content": tc.requestContent,
				}
				bodyBytes, err := json.Marshal(requestBody)
				require.NoError(t, err, "Failed to marshal request body")

				// Create request
				req, err := http.NewRequest(
					http.MethodPut,
					server.URL+"/api/cards/"+tc.cardID.String(),
					bytes.NewBuffer(bodyBytes),
				)
				require.NoError(t, err, "Failed to create request")
				req.Header.Set("Content-Type", "application/json")

				// Add auth header
				authHeader, err := testutils.GenerateAuthHeader(tc.userID)
				require.NoError(t, err, "Failed to generate auth header")
				req.Header.Set("Authorization", authHeader)

				// Execute request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to execute request")
				defer resp.Body.Close()

				// Verify response
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create stores with the transaction
		logger := slog.Default()
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		memoStore := postgres.NewPostgresMemoStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create test user
		testUser, err := domain.NewUser("delete-card-integration-test@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(t, userStore.Create(ctx, testUser), "Failed to save test user")

		// Create another user for ownership test
		otherUser, err := domain.NewUser("other-user-delete-card@example.com", "password123")
		require.NoError(t, err, "Failed to create other user")
		require.NoError(t, userStore.Create(ctx, otherUser), "Failed to save other user")

		// Create test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Delete Card Integration Test")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(t, memoStore.Create(ctx, testMemo), "Failed to save test memo")

		// Create test card with content
		cardContent := map[string]interface{}{
			"front": "Test question",
			"back":  "Test answer",
		}
		contentBytes, err := json.Marshal(cardContent)
		require.NoError(t, err, "Failed to marshal card content")

		card, err := domain.NewCard(testUser.ID, testMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(t, cardStore.CreateMultiple(ctx, []*domain.Card{card}), "Failed to save test card")

		// Create card owned by other user for ownership test
		otherUserMemo, err := domain.NewMemo(otherUser.ID, "Other User Memo")
		require.NoError(t, err, "Failed to create other user memo")
		require.NoError(t, memoStore.Create(ctx, otherUserMemo), "Failed to save other user memo")

		otherUserCard, err := domain.NewCard(otherUser.ID, otherUserMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create other user card")
		require.NoError(
			t,
			cardStore.CreateMultiple(ctx, []*domain.Card{otherUserCard}),
			"Failed to save other user card",
		)

		// Create SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err, "Failed to create SRS service")

		// Create repository adapters for service layer
		dbConn := cardStore.DB()
		cardRepo := service.NewCardRepositoryAdapter(cardStore, dbConn)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create services using the adapters
		cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
		require.NoError(t, err, "Failed to create card service")

		cardReviewService, err := card_review.NewCardReviewService(cardStore, statsStore, srsService, logger)
		require.NoError(t, err, "Failed to create card review service")

		// Create JWT service for auth
		jwtService, err := auth.NewTestJWTService()
		require.NoError(t, err, "Failed to create JWT service")

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
				// Create router with standard middleware
				router := chi.NewRouter()
				router.Use(chimiddleware.RequestID)
				router.Use(chimiddleware.RealIP)
				router.Use(chimiddleware.Recoverer)

				// Create auth middleware
				authMiddleware := middleware.NewAuthMiddleware(jwtService)

				// Create card handler with real services
				cardHandler := api.NewCardHandler(cardReviewService, cardService, logger)

				// Set up API routes
				router.Route("/api", func(r chi.Router) {
					r.Group(func(r chi.Router) {
						r.Use(authMiddleware.Authenticate)
						r.Delete("/cards/{id}", cardHandler.DeleteCard)
					})
				})

				// Create server
				server := httptest.NewServer(router)
				defer server.Close()

				// Create request
				req, err := http.NewRequest(
					http.MethodDelete,
					server.URL+"/api/cards/"+tc.cardID.String(),
					nil,
				)
				require.NoError(t, err, "Failed to create request")

				// Add auth header
				authHeader, err := testutils.GenerateAuthHeader(tc.userID)
				require.NoError(t, err, "Failed to generate auth header")
				req.Header.Set("Authorization", authHeader)

				// Execute request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to execute request")
				defer resp.Body.Close()

				// Verify response
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Get a test database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create stores with the transaction
		logger := slog.Default()
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		memoStore := postgres.NewPostgresMemoStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create test user
		testUser, err := domain.NewUser("postpone-card-integration-test@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(t, userStore.Create(ctx, testUser), "Failed to save test user")

		// Create another user for ownership test
		otherUser, err := domain.NewUser("other-user-postpone-card@example.com", "password123")
		require.NoError(t, err, "Failed to create other user")
		require.NoError(t, userStore.Create(ctx, otherUser), "Failed to save other user")

		// Create test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Postpone Card Integration Test")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(t, memoStore.Create(ctx, testMemo), "Failed to save test memo")

		// Create test card with content
		cardContent := map[string]interface{}{
			"front": "Test question",
			"back":  "Test answer",
		}
		contentBytes, err := json.Marshal(cardContent)
		require.NoError(t, err, "Failed to marshal card content")

		card, err := domain.NewCard(testUser.ID, testMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(t, cardStore.CreateMultiple(ctx, []*domain.Card{card}), "Failed to save test card")

		// Create user card stats
		stats, err := domain.NewUserCardStats(testUser.ID, card.ID)
		require.NoError(t, err, "Failed to create test stats")
		stats.NextReviewAt = time.Now().UTC().Add(24 * time.Hour) // Set initial review time
		require.NoError(t, statsStore.Create(ctx, stats), "Failed to save test stats")

		// Create card owned by other user for ownership test
		otherUserMemo, err := domain.NewMemo(otherUser.ID, "Other User Memo")
		require.NoError(t, err, "Failed to create other user memo")
		require.NoError(t, memoStore.Create(ctx, otherUserMemo), "Failed to save other user memo")

		otherUserCard, err := domain.NewCard(otherUser.ID, otherUserMemo.ID, contentBytes)
		require.NoError(t, err, "Failed to create other user card")
		require.NoError(
			t,
			cardStore.CreateMultiple(ctx, []*domain.Card{otherUserCard}),
			"Failed to save other user card",
		)

		// Create SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err, "Failed to create SRS service")

		// Create repository adapters for service layer
		dbConn := cardStore.DB()
		cardRepo := service.NewCardRepositoryAdapter(cardStore, dbConn)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create services using the adapters
		cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
		require.NoError(t, err, "Failed to create card service")

		cardReviewService, err := card_review.NewCardReviewService(cardStore, statsStore, srsService, logger)
		require.NoError(t, err, "Failed to create card review service")

		// Create JWT service for auth
		jwtService, err := auth.NewTestJWTService()
		require.NoError(t, err, "Failed to create JWT service")

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
				// Create router with standard middleware
				router := chi.NewRouter()
				router.Use(chimiddleware.RequestID)
				router.Use(chimiddleware.RealIP)
				router.Use(chimiddleware.Recoverer)

				// Create auth middleware
				authMiddleware := middleware.NewAuthMiddleware(jwtService)

				// Create card handler with real services
				cardHandler := api.NewCardHandler(cardReviewService, cardService, logger)

				// Set up API routes
				router.Route("/api", func(r chi.Router) {
					r.Group(func(r chi.Router) {
						r.Use(authMiddleware.Authenticate)
						r.Post("/cards/{id}/postpone", cardHandler.PostponeCard)
					})
				})

				// Create server
				server := httptest.NewServer(router)
				defer server.Close()

				// Create request body
				requestBody := map[string]interface{}{
					"days": tc.days,
				}
				bodyBytes, err := json.Marshal(requestBody)
				require.NoError(t, err, "Failed to marshal request body")

				// Create request
				req, err := http.NewRequest(
					http.MethodPost,
					server.URL+"/api/cards/"+tc.cardID.String()+"/postpone",
					bytes.NewBuffer(bodyBytes),
				)
				require.NoError(t, err, "Failed to create request")
				req.Header.Set("Content-Type", "application/json")

				// Add auth header
				authHeader, err := testutils.GenerateAuthHeader(tc.userID)
				require.NoError(t, err, "Failed to generate auth header")
				req.Header.Set("Authorization", authHeader)

				// Execute request
				client := &http.Client{}
				resp, err := client.Do(req)
				require.NoError(t, err, "Failed to execute request")
				defer resp.Body.Close()

				// Verify response
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
