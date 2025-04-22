//go:build integration
// +build integration

// This file contains integration tests for the card review API endpoints.
// These tests require a real database connection to run.
// To run these tests:
//  1. Set up a local database with the appropriate schema
//  2. Set the DATABASE_URL environment variable to a valid connection string
//  3. Run the tests with the -tags=integration flag:
//     go test -v -tags=integration ./cmd/server
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// setupCardReviewTestServer creates a test server for card review API integration tests
func setupCardReviewTestServer(t *testing.T, tx store.DBTX) *httptest.Server {
	t.Helper()

	// Initialize logger
	log := slog.Default()

	// Create card store
	cardStore := postgres.NewPostgresCardStore(tx, log)

	// Create stats store
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, log)

	// Get database connection from the shared testDB variable declared in main_task_test.go
	var db *sql.DB
	if testDB != nil {
		db = testDB
	}

	// Create repository adapters
	cardRepoAdapter := card_review.NewCardRepositoryAdapter(cardStore, db)
	statsRepoAdapter := card_review.NewUserCardStatsRepositoryAdapter(statsStore)

	// Create SRS service with default parameters
	srsService := srs.NewDefaultService()

	// Create card review service with dependencies
	cardReviewService := card_review.NewCardReviewService(
		cardRepoAdapter,
		statsRepoAdapter,
		srsService,
		log,
	)

	// Create JWT service for auth
	jwtService, err := auth.NewJWTService(config.AuthConfig{
		JWTSecret:                   "thisisatestjwtsecretthatis32charslong",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 24 * 60, // 1 day
	})
	require.NoError(t, err)

	// Create auth middleware
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)

	// Create card handler
	cardHandler := api.NewCardHandler(cardReviewService, log)

	// Create router
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Register routes
	r.Route("/api", func(r chi.Router) {
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Card review endpoints
			r.Get("/cards/next", cardHandler.GetNextReviewCard)
			r.Post("/cards/{id}/answer", cardHandler.SubmitAnswer)
		})
	})

	// Create test server
	testServer := httptest.NewServer(r)

	return testServer
}

// TestCardReviewIntegration tests the card review API endpoints
func TestCardReviewIntegration(t *testing.T) {
	// Skip if database is not available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Run the test in a transaction for isolation
	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Create a test server with the transaction
		testServer := setupCardReviewTestServer(t, tx)
		defer testServer.Close()

		// Create user store for test data setup
		userStore := postgres.NewPostgresUserStore(tx, 10)

		// Create a test user
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("securepassword1234"), bcrypt.DefaultCost)
		require.NoError(t, err)

		user := &domain.User{
			ID:             uuid.New(),
			Email:          "card-review-test@example.com",
			HashedPassword: string(hashedPassword),
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		err = userStore.Create(context.Background(), user)
		require.NoError(t, err)

		userID := user.ID

		// Create a JWT token for the user
		jwtService, err := auth.NewJWTService(config.AuthConfig{
			JWTSecret:                   "thisisatestjwtsecretthatis32charslong",
			TokenLifetimeMinutes:        60,
			RefreshTokenLifetimeMinutes: 24 * 60, // 1 day
		})
		require.NoError(t, err)

		token, err := jwtService.GenerateToken(context.Background(), userID)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		// Create a test card
		memoID := uuid.New()
		cardContent := map[string]interface{}{
			"front": "What is the capital of France?",
			"back":  "Paris",
		}
		contentBytes, err := json.Marshal(cardContent)
		require.NoError(t, err)

		card := &domain.Card{
			ID:        uuid.New(),
			UserID:    userID,
			MemoID:    memoID,
			Content:   contentBytes,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		// Insert card directly using SQL
		_, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			card.ID, card.UserID, card.MemoID, card.Content, card.CreatedAt, card.UpdatedAt,
		)
		require.NoError(t, err)

		cardID := card.ID

		// Test GET /cards/next with no stats (no cards due)
		t.Run("GetNextCard_NoCardsDue", func(t *testing.T) {
			// Create request
			req, err := http.NewRequest("GET", testServer.URL+"/api/cards/next", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		// Create stats with a due date in the past
		now := time.Now().UTC()
		pastTime := now.Add(-24 * time.Hour)
		stats := &domain.UserCardStats{
			UserID:             userID,
			CardID:             cardID,
			Interval:           1,
			EaseFactor:         2.5,
			ConsecutiveCorrect: 0,
			LastReviewedAt:     pastTime,
			NextReviewAt:       pastTime, // Due in the past
			ReviewCount:        0,
			CreatedAt:          pastTime,
			UpdatedAt:          pastTime,
		}

		// Insert stats directly using SQL
		_, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO user_card_stats (user_id, card_id, interval, ease_factor, consecutive_correct,
				last_reviewed_at, next_review_at, review_count, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			userID, cardID, stats.Interval, stats.EaseFactor, stats.ConsecutiveCorrect,
			stats.LastReviewedAt, stats.NextReviewAt, stats.ReviewCount, stats.CreatedAt, stats.UpdatedAt,
		)
		require.NoError(t, err)

		// Test GET /cards/next with a card due for review
		t.Run("GetNextCard_Success", func(t *testing.T) {
			// Create request
			req, err := http.NewRequest("GET", testServer.URL+"/api/cards/next", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Verify response body
			var cardResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&cardResp)
			require.NoError(t, err)

			// Check card fields
			assert.Equal(t, cardID.String(), cardResp["id"])
			assert.Equal(t, userID.String(), cardResp["user_id"])
			assert.Equal(t, memoID.String(), cardResp["memo_id"])

			// Check card content
			content, ok := cardResp["content"].(map[string]interface{})
			assert.True(t, ok, "Content should be a map")
			assert.Equal(t, "What is the capital of France?", content["front"])
			assert.Equal(t, "Paris", content["back"])
		})

		// Test GET /cards/next with invalid auth token
		t.Run("GetNextCard_Unauthorized", func(t *testing.T) {
			// Create request
			req, err := http.NewRequest("GET", testServer.URL+"/api/cards/next", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer invalid-token")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})

		// Test POST /cards/{id}/answer with successful review
		t.Run("SubmitAnswer_Success", func(t *testing.T) {
			// Create review payload
			reviewPayload := map[string]string{
				"outcome": "good",
			}
			reviewBody, err := json.Marshal(reviewPayload)
			require.NoError(t, err)

			// Create request
			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("%s/api/cards/%s/answer", testServer.URL, cardID.String()),
				bytes.NewBuffer(reviewBody),
			)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Verify response body
			var statsResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&statsResp)
			require.NoError(t, err)

			// Check stats fields
			assert.Equal(t, userID.String(), statsResp["user_id"])
			assert.Equal(t, cardID.String(), statsResp["card_id"])
			assert.Greater(t, statsResp["interval"], float64(0))
			assert.Equal(t, float64(2.5), statsResp["ease_factor"]) // Unchanged for "good"
			assert.Equal(t, float64(1), statsResp["consecutive_correct"])
			assert.Equal(t, float64(1), statsResp["review_count"])

			// Check that stats were updated in database
			var interval int
			var easeFactor float64
			var consecutiveCorrect int
			var reviewCount int
			var nextReviewAt time.Time

			err = tx.QueryRowContext(
				context.Background(),
				`SELECT interval, ease_factor, consecutive_correct, review_count, next_review_at
				FROM user_card_stats
				WHERE user_id = $1 AND card_id = $2`,
				userID, cardID,
			).Scan(&interval, &easeFactor, &consecutiveCorrect, &reviewCount, &nextReviewAt)
			require.NoError(t, err)

			assert.Equal(t, 1, interval)
			assert.Equal(t, 2.5, easeFactor)
			assert.Equal(t, 1, consecutiveCorrect)
			assert.Equal(t, 1, reviewCount)
			assert.True(t, nextReviewAt.After(now))
		})

		// Test POST /cards/{id}/answer with non-existent card
		t.Run("SubmitAnswer_CardNotFound", func(t *testing.T) {
			// Create random card ID
			nonExistentCardID := uuid.New()

			// Create review payload
			reviewPayload := map[string]string{
				"outcome": "good",
			}
			reviewBody, err := json.Marshal(reviewPayload)
			require.NoError(t, err)

			// Create request
			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("%s/api/cards/%s/answer", testServer.URL, nonExistentCardID.String()),
				bytes.NewBuffer(reviewBody),
			)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		// Test POST /cards/{id}/answer with invalid outcome
		t.Run("SubmitAnswer_InvalidOutcome", func(t *testing.T) {
			// Create review payload with invalid outcome
			reviewPayload := map[string]string{
				"outcome": "invalid",
			}
			reviewBody, err := json.Marshal(reviewPayload)
			require.NoError(t, err)

			// Create request
			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("%s/api/cards/%s/answer", testServer.URL, cardID.String()),
				bytes.NewBuffer(reviewBody),
			)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		// Create a second user to test card ownership
		hashedPassword2, err := bcrypt.GenerateFromPassword([]byte("anotherpassword"), bcrypt.DefaultCost)
		require.NoError(t, err)

		user2 := &domain.User{
			ID:             uuid.New(),
			Email:          "another-user@example.com",
			HashedPassword: string(hashedPassword2),
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		err = userStore.Create(context.Background(), user2)
		require.NoError(t, err)

		user2ID := user2.ID

		// Create token for second user
		token2, err := jwtService.GenerateToken(context.Background(), user2ID)
		require.NoError(t, err)
		require.NotEmpty(t, token2)

		// Test POST /cards/{id}/answer with card not owned by user
		t.Run("SubmitAnswer_CardNotOwned", func(t *testing.T) {
			// Create review payload
			reviewPayload := map[string]string{
				"outcome": "good",
			}
			reviewBody, err := json.Marshal(reviewPayload)
			require.NoError(t, err)

			// Create request - user2's token but user1's card
			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("%s/api/cards/%s/answer", testServer.URL, cardID.String()),
				bytes.NewBuffer(reviewBody),
			)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token2) // User2's token
			req.Header.Set("Content-Type", "application/json")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})

		// Test POST /cards/{id}/answer with invalid auth token
		t.Run("SubmitAnswer_Unauthorized", func(t *testing.T) {
			// Create review payload
			reviewPayload := map[string]string{
				"outcome": "good",
			}
			reviewBody, err := json.Marshal(reviewPayload)
			require.NoError(t, err)

			// Create request
			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("%s/api/cards/%s/answer", testServer.URL, cardID.String()),
				bytes.NewBuffer(reviewBody),
			)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer invalid-token")
			req.Header.Set("Content-Type", "application/json")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			// Verify status code
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	})
}
