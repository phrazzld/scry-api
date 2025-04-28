//go:build integration

package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// This file contains integration tests for the CardService implementation.
// It uses real implementations with transaction-based test isolation.

// TestCardService_UpdateCardContent tests the UpdateCardContent method
func TestCardService_UpdateCardContent(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	// Run tests in transaction for isolation
	testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Setup shared test values
		ctx := context.Background()
		logger := slog.Default()

		// Create repository adapters with transaction
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create repositories
		cardRepo := service.NewCardRepositoryAdapter(cardStore, db)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create real SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err)

		// Create the card service
		cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
		require.NoError(t, err)

		// Create a test user
		userID := testutils.MustInsertUser(ctx, t, tx, "cardservice_update@example.com", bcrypt.MinCost)

		// Create a test memo
		memoID := uuid.New()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO memos (id, user_id, text, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, memoID, userID, "Test memo content", "pending", time.Now().UTC(), time.Now().UTC())
		require.NoError(t, err)

		// Create a test card
		originalContent := json.RawMessage(`{"front":"Original front","back":"Original back"}`)
		cardID := uuid.New()
		now := time.Now().UTC()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, cardID, userID, memoID, originalContent, now, now)
		require.NoError(t, err)

		// Create a card owned by another user
		otherUserID := testutils.MustInsertUser(ctx, t, tx, "cardservice_other@example.com", bcrypt.MinCost)
		otherCardID := uuid.New()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, otherCardID, otherUserID, memoID, originalContent, now, now)
		require.NoError(t, err)

		t.Run("success", func(t *testing.T) {
			// New content to update
			newContent := json.RawMessage(`{"front":"Updated front","back":"Updated back"}`)

			// Execute the method
			err := cardService.UpdateCardContent(ctx, userID, cardID, newContent)

			// Assertions
			assert.NoError(t, err)

			// Verify the card content was updated in database
			var updatedContent json.RawMessage
			err = tx.QueryRowContext(ctx, `
				SELECT content FROM cards WHERE id = $1
			`, cardID).Scan(&updatedContent)
			require.NoError(t, err)

			assert.JSONEq(t, string(newContent), string(updatedContent))
		})

		t.Run("card not found", func(t *testing.T) {
			// Generate random non-existent card ID
			nonExistentID := uuid.New()

			// New content to update
			newContent := json.RawMessage(`{"front":"Updated front","back":"Updated back"}`)

			// Execute the method
			err := cardService.UpdateCardContent(ctx, userID, nonExistentID, newContent)

			// Assertions
			assert.Error(t, err)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			assert.ErrorIs(t, errors.Unwrap(err), store.ErrCardNotFound)
		})

		t.Run("not card owner", func(t *testing.T) {
			// New content to update
			newContent := json.RawMessage(`{"front":"Updated front","back":"Updated back"}`)

			// Execute the method with wrong user trying to update other user's card
			err := cardService.UpdateCardContent(ctx, userID, otherCardID, newContent)

			// Assertions
			assert.Error(t, err)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			assert.ErrorIs(t, errors.Unwrap(err), service.ErrNotOwned)
		})

		t.Run("invalid content", func(t *testing.T) {
			// Invalid JSON content
			invalidContent := json.RawMessage(`{"front":"Invalid JSON`)

			// Execute the method
			err := cardService.UpdateCardContent(ctx, userID, cardID, invalidContent)

			// Assertions
			assert.Error(t, err)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			// The error should indicate invalid entity
			assert.Contains(t, err.Error(), "invalid")
		})
	})
}

// TestCardService_DeleteCard tests the DeleteCard method
func TestCardService_DeleteCard(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	// Run tests in transaction for isolation
	testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Setup shared test values
		ctx := context.Background()
		logger := slog.Default()

		// Create repository adapters with transaction
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create repositories
		cardRepo := service.NewCardRepositoryAdapter(cardStore, db)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create real SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err)

		// Create the card service
		cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
		require.NoError(t, err)

		// Create a test user
		userID := testutils.MustInsertUser(ctx, t, tx, "cardservice_delete@example.com", bcrypt.MinCost)

		// Create a test memo
		memoID := uuid.New()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO memos (id, user_id, text, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, memoID, userID, "Test memo content", "pending", time.Now().UTC(), time.Now().UTC())
		require.NoError(t, err)

		// Create a test card
		content := json.RawMessage(`{"front":"Card to delete","back":"Delete me"}`)
		cardID := uuid.New()
		now := time.Now().UTC()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, cardID, userID, memoID, content, now, now)
		require.NoError(t, err)

		// Create initial user card stats
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_card_stats (
				user_id, card_id, interval, ease_factor, consecutive_correct,
				last_reviewed_at, next_review_at, review_count, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, userID, cardID, 1, 2.5, 0, now, now.Add(24*time.Hour), 0, now, now)
		require.NoError(t, err)

		// Create a card owned by another user
		otherUserID := testutils.MustInsertUser(ctx, t, tx, "cardservice_delete_other@example.com", bcrypt.MinCost)
		otherCardID := uuid.New()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, otherCardID, otherUserID, memoID, content, now, now)
		require.NoError(t, err)

		t.Run("success", func(t *testing.T) {
			// Execute the method
			err := cardService.DeleteCard(ctx, userID, cardID)

			// Assertions
			assert.NoError(t, err)

			// Verify the card is deleted from database
			var count int
			err = tx.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM cards WHERE id = $1
			`, cardID).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 0, count, "Card should be deleted")

			// Verify the stats are also deleted
			err = tx.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM user_card_stats WHERE card_id = $1
			`, cardID).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 0, count, "Stats should be deleted")
		})

		t.Run("card not found", func(t *testing.T) {
			// Generate random non-existent card ID
			nonExistentID := uuid.New()

			// Execute the method
			err := cardService.DeleteCard(ctx, userID, nonExistentID)

			// Assertions
			assert.Error(t, err)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			assert.ErrorIs(t, errors.Unwrap(err), store.ErrCardNotFound)
		})

		t.Run("not card owner", func(t *testing.T) {
			// Execute the method with wrong user trying to delete other user's card
			err := cardService.DeleteCard(ctx, userID, otherCardID)

			// Assertions
			assert.Error(t, err)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			assert.ErrorIs(t, errors.Unwrap(err), service.ErrNotOwned)
		})
	})
}

// TestCardService_PostponeCard tests the PostponeCard method
func TestCardService_PostponeCard(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	// Run tests in transaction for isolation
	testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Setup shared test values
		ctx := context.Background()
		logger := slog.Default()
		days := 7

		// Create repository adapters with transaction
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create repositories
		cardRepo := service.NewCardRepositoryAdapter(cardStore, db)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create real SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err)

		// Create the card service
		cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
		require.NoError(t, err)

		// Create a test user
		userID := testutils.MustInsertUser(ctx, t, tx, "cardservice_postpone@example.com", bcrypt.MinCost)

		// Create a test memo
		memoID := uuid.New()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO memos (id, user_id, text, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, memoID, userID, "Test memo content", "pending", time.Now().UTC(), time.Now().UTC())
		require.NoError(t, err)

		// Create a test card
		content := json.RawMessage(`{"front":"Card to postpone","back":"Postpone me"}`)
		cardID := uuid.New()
		now := time.Now().UTC()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, cardID, userID, memoID, content, now, now)
		require.NoError(t, err)

		// Create initial user card stats
		nextReviewAt := now.Add(24 * time.Hour)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_card_stats (
				user_id, card_id, interval, ease_factor, consecutive_correct,
				last_reviewed_at, next_review_at, review_count, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, userID, cardID, 1, 2.5, 0, now, nextReviewAt, 0, now, now)
		require.NoError(t, err)

		// Create a card owned by another user
		otherUserID := testutils.MustInsertUser(ctx, t, tx, "cardservice_postpone_other@example.com", bcrypt.MinCost)
		otherCardID := uuid.New()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, otherCardID, otherUserID, memoID, content, now, now)
		require.NoError(t, err)

		t.Run("invalid days", func(t *testing.T) {
			// Execute with invalid days
			result, err := cardService.PostponeCard(ctx, userID, cardID, 0)

			// Assertions
			assert.Error(t, err)
			assert.Nil(t, result)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			assert.ErrorIs(t, errors.Unwrap(err), srs.ErrInvalidDays)
		})

		t.Run("success", func(t *testing.T) {
			// Execute the method
			result, err := cardService.PostponeCard(ctx, userID, cardID, days)

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Calculate expected next review date (original + days days)
			expectedNextReviewAt := nextReviewAt.AddDate(0, 0, days)

			// Verify with less than 1 second tolerance due to potential rounding
			assert.WithinDuration(t, expectedNextReviewAt, result.NextReviewAt, time.Second)

			// Verify the stats are updated in the database
			var dbNextReviewAt time.Time
			err = tx.QueryRowContext(ctx, `
				SELECT next_review_at FROM user_card_stats WHERE card_id = $1
			`, cardID).Scan(&dbNextReviewAt)
			require.NoError(t, err)
			assert.WithinDuration(t, expectedNextReviewAt, dbNextReviewAt, time.Second)
		})

		t.Run("card not found", func(t *testing.T) {
			// Generate random non-existent card ID
			nonExistentID := uuid.New()

			// Execute the method
			result, err := cardService.PostponeCard(ctx, userID, nonExistentID, days)

			// Assertions
			assert.Error(t, err)
			assert.Nil(t, result)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			assert.ErrorIs(t, errors.Unwrap(err), store.ErrCardNotFound)
		})

		t.Run("not card owner", func(t *testing.T) {
			// Execute the method with wrong user trying to postpone other user's card
			result, err := cardService.PostponeCard(ctx, userID, otherCardID, days)

			// Assertions
			assert.Error(t, err)
			assert.Nil(t, result)
			var cardSvcErr *service.CardServiceError
			assert.True(t, errors.As(err, &cardSvcErr))
			assert.ErrorIs(t, errors.Unwrap(err), service.ErrNotOwned)
		})
	})
}
