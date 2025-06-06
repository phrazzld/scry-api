package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Test timeout to prevent long-running tests
const testTimeout = 5 * time.Second

// checkIntegrationTestEnvironment checks if we're running in an environment
// where integration tests can be executed, by checking DATABASE_URL
//
// NOTE: This duplicates functionality in testutils.IsIntegrationTestEnvironment,
// but is kept here to avoid import cycles.
func checkIntegrationTestEnvironment() bool {
	return os.Getenv("DATABASE_URL") != ""
}

// getTestDBForCardStore gets a connection to the test database
//
// NOTE: This duplicates functionality in testutils.GetTestDB,
// but is kept here to avoid import cycles.
func getTestDBForCardStore() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection works
	if err := db.Ping(); err != nil {
		_ = db.Close() // Explicitly ignore error from Close
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// withTxForCardTest executes a function within a transaction and rolls it back afterward.
// This ensures that tests are isolated and don't affect each other.
//
// NOTE: This duplicates functionality in testutils.WithTx,
// but is kept here to avoid import cycles.
func withTxForCardTest(t *testing.T, db *sql.DB, fn func(tx *sql.Tx)) {
	t.Helper()

	// Start a transaction
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	require.NoError(t, err, "Failed to begin transaction")

	// Ensure the transaction is rolled back when the test completes
	defer func() {
		err := tx.Rollback()
		// Ignore error if transaction was already committed
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Logf("Error rolling back transaction: %v", err)
		}
	}()

	// Execute the test function
	fn(tx)
}

// TestCardStoreIntegration runs a complete set of integration tests for the CardStore implementation.
// This ensures all methods work as expected with a real database connection.
func TestCardStoreIntegration(t *testing.T) {
	// Skip the integration test wrapper if not in integration test environment
	if !checkIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Run integration tests for each CardStore method
	t.Run("TestPostgresCardStore_CreateMultiple", TestPostgresCardStore_CreateMultiple)
	t.Run("TestPostgresCardStore_GetNextReviewCard", TestPostgresCardStore_GetNextReviewCard)
}

// TestPostgresCardStore_GetNextReviewCard tests the GetNextReviewCard method
func TestPostgresCardStore_GetNextReviewCard(t *testing.T) {
	// Skip if not in integration test environment
	if !checkIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	t.Parallel() // Enable parallel testing

	// Get a database connection
	db, err := getTestDBForCardStore()
	require.NoError(t, err, "Failed to connect to test database")
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	withTxForCardTest(t, db, func(tx *sql.Tx) {
		// Create stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		cardStore := NewPostgresCardStore(tx, nil)
		memoStore := NewPostgresMemoStore(tx, nil)
		statsStore := NewPostgresUserCardStatsStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("getnextreview@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a second test user for multi-user isolation tests
		otherUser, err := domain.NewUser("othergetnextreview@example.com", "password123")
		require.NoError(t, err, "Failed to create other test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), otherUser),
			"Failed to create other test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for cards")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		// Create a memo for the other user
		otherMemo, err := domain.NewMemo(otherUser.ID, "Other user's memo for cards")
		require.NoError(t, err, "Failed to create other user's memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), otherMemo),
			"Failed to create other user's memo in DB",
		)

		// Helper function to create a card with stats
		createCardWithStats := func(userID, memoID uuid.UUID, nextReviewAt time.Time) (*domain.Card, *domain.UserCardStats, error) {
			content := json.RawMessage(`{"front":"Test front","back":"Test back"}`)
			card, err := domain.NewCard(userID, memoID, content)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create card: %w", err)
			}

			err = cardStore.CreateMultiple(context.Background(), []*domain.Card{card})
			if err != nil {
				return nil, nil, fmt.Errorf("failed to insert card: %w", err)
			}

			// Create stats for the card with specified next review time
			stats, err := domain.NewUserCardStats(userID, card.ID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create stats: %w", err)
			}
			stats.NextReviewAt = nextReviewAt

			err = statsStore.Create(context.Background(), stats)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to insert stats: %w", err)
			}

			return card, stats, nil
		}

		t.Run("no_cards_due", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a card with future review date (not due yet)
			futureTime := time.Now().UTC().Add(24 * time.Hour) // Tomorrow
			_, _, err := createCardWithStats(testUser.ID, testMemo.ID, futureTime)
			require.NoError(t, err, "Failed to create card with future review date")

			// Call GetNextReviewCard which should return ErrCardNotFound
			_, err = cardStore.GetNextReviewCard(ctx, testUser.ID)
			assert.Error(t, err, "GetNextReviewCard should return an error for no due cards")
			assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")
		})

		t.Run("multiple_cards_returns_oldest_due", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create three cards with different review times
			now := time.Now().UTC()
			pastTime1 := now.Add(-1 * time.Hour)    // 1 hour ago
			pastTime2 := now.Add(-3 * time.Hour)    // 3 hours ago (oldest, should be returned)
			pastTime3 := now.Add(-30 * time.Minute) // 30 minutes ago

			_, _, err := createCardWithStats(testUser.ID, testMemo.ID, pastTime1)
			require.NoError(t, err, "Failed to create card with past review date 1")

			oldestCard, _, err := createCardWithStats(testUser.ID, testMemo.ID, pastTime2)
			require.NoError(t, err, "Failed to create card with past review date 2")

			_, _, err = createCardWithStats(testUser.ID, testMemo.ID, pastTime3)
			require.NoError(t, err, "Failed to create card with past review date 3")

			// Call GetNextReviewCard which should return the oldest due card
			card, err := cardStore.GetNextReviewCard(ctx, testUser.ID)
			assert.NoError(t, err, "GetNextReviewCard should succeed with due cards")
			assert.NotNil(t, card, "Returned card should not be nil")
			assert.Equal(t, oldestCard.ID, card.ID, "Should return the oldest due card")
		})

		t.Run("user_isolation", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create card for other user that's due earlier
			otherUserPastTime := time.Now().UTC().Add(-5 * time.Hour) // 5 hours ago (earliest)
			_, _, err := createCardWithStats(otherUser.ID, otherMemo.ID, otherUserPastTime)
			require.NoError(t, err, "Failed to create card for other user")

			// Create card for test user
			userPastTime := time.Now().UTC().Add(-2 * time.Hour) // 2 hours ago
			userCard, _, err := createCardWithStats(testUser.ID, testMemo.ID, userPastTime)
			require.NoError(t, err, "Failed to create card for test user")

			// Call GetNextReviewCard for the test user
			// Should only return the test user's card, even though other user has earlier card
			card, err := cardStore.GetNextReviewCard(ctx, testUser.ID)
			assert.NoError(t, err, "GetNextReviewCard should succeed with due cards")
			assert.NotNil(t, card, "Returned card should not be nil")
			assert.Equal(t, userCard.ID, card.ID, "Should return only the test user's due card")
		})
	})
}

// TestPostgresCardStore_CreateMultiple tests the CreateMultiple method
func TestPostgresCardStore_CreateMultiple(t *testing.T) {
	// Skip if not in integration test environment
	if !checkIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	t.Parallel() // Enable parallel testing

	// Get a database connection
	db, err := getTestDBForCardStore()
	require.NoError(t, err, "Failed to connect to test database")
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	withTxForCardTest(t, db, func(tx *sql.Tx) {
		// Create stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)
		statsStore := NewPostgresUserCardStatsStore(tx, nil)

		// Create a test user first to satisfy foreign key constraints
		testUser, err := domain.NewUser("test@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo to satisfy foreign key constraints
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo text")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		t.Run("empty_cards_list", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Test with empty list
			err := cardStore.CreateMultiple(ctx, []*domain.Card{})
			assert.NoError(t, err, "CreateMultiple should succeed with empty list")
		})

		t.Run("single_card", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a valid card
			content := json.RawMessage(`{"front":"Test front","back":"Test back"}`)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")

			// Insert the card
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			assert.NoError(t, err, "CreateMultiple should succeed with valid card")

			// Verify the card was created
			retrievedCard, err := cardStore.GetByID(ctx, card.ID)
			assert.NoError(t, err, "GetByID should find the created card")
			assert.Equal(t, card.ID, retrievedCard.ID, "Retrieved card should have same ID")
			assert.Equal(
				t,
				testUser.ID,
				retrievedCard.UserID,
				"Retrieved card should have correct user ID",
			)
			assert.Equal(
				t,
				testMemo.ID,
				retrievedCard.MemoID,
				"Retrieved card should have correct memo ID",
			)
			assert.JSONEq(
				t,
				string(content),
				string(retrievedCard.Content),
				"Retrieved card should have correct content",
			)

			// Verify that user card stats are NOT created anymore
			// This test specifically verifies that the refactored CardStore.CreateMultiple
			// no longer creates associated UserCardStats entries
			_, err = statsStore.Get(ctx, testUser.ID, card.ID)
			assert.Error(t, err, "Should not find stats for created card")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
		})

		t.Run("multiple_cards", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create multiple valid cards
			cards := make([]*domain.Card, 3)
			for i := 0; i < 3; i++ {
				content := json.RawMessage(
					`{"front":"Test front ` + string(
						rune('A'+i),
					) + `","back":"Test back ` + string(
						rune('A'+i),
					) + `"}`,
				)
				card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
				require.NoError(t, err, "Failed to create test card")
				cards[i] = card
			}

			// Insert the cards
			err = cardStore.CreateMultiple(ctx, cards)
			assert.NoError(t, err, "CreateMultiple should succeed with valid cards")

			// Verify each card was created
			for _, card := range cards {
				retrievedCard, err := cardStore.GetByID(ctx, card.ID)
				assert.NoError(t, err, "GetByID should find created card")
				assert.Equal(t, card.ID, retrievedCard.ID, "Retrieved card should have same ID")

				// Verify that user card stats are NOT created anymore
				_, err = statsStore.Get(ctx, testUser.ID, card.ID)
				assert.Error(t, err, "Should not find stats for created card")
				assert.ErrorIs(
					t,
					err,
					store.ErrUserCardStatsNotFound,
					"Error should be ErrUserCardStatsNotFound",
				)
			}
		})

		t.Run("invalid_card", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create an invalid card with empty content
			invalidCard := &domain.Card{
				ID:        uuid.New(),
				UserID:    testUser.ID,
				MemoID:    testMemo.ID,
				Content:   nil, // Invalid - empty content
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			// Insert should fail
			err = cardStore.CreateMultiple(ctx, []*domain.Card{invalidCard})
			assert.Error(t, err, "CreateMultiple should fail with invalid card")
			assert.ErrorIs(t, err, store.ErrInvalidEntity, "Error should be ErrInvalidEntity")
			assert.ErrorContains(t, err, "empty card content", "Error should mention empty content")
		})

		t.Run("non_existent_user", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a card with non-existent user ID
			nonExistentUserID := uuid.New()
			content := json.RawMessage(`{"front":"Test front","back":"Test back"}`)
			card, err := domain.NewCard(nonExistentUserID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")

			// Insert should fail due to foreign key constraint
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			assert.Error(t, err, "CreateMultiple should fail with non-existent user")
			assert.ErrorIs(t, err, store.ErrInvalidEntity, "Error should be ErrInvalidEntity")
		})

		t.Run("non_existent_memo", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a card with non-existent memo ID
			nonExistentMemoID := uuid.New()
			content := json.RawMessage(`{"front":"Test front","back":"Test back"}`)
			card, err := domain.NewCard(testUser.ID, nonExistentMemoID, content)
			require.NoError(t, err, "Failed to create test card")

			// Insert should fail due to foreign key constraint
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			assert.Error(t, err, "CreateMultiple should fail with non-existent memo")
			assert.ErrorIs(t, err, store.ErrInvalidEntity, "Error should be ErrInvalidEntity")
		})

		t.Run("invalid_json_content", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a card with invalid JSON content
			invalidJSON := json.RawMessage(`{"front":"Test front","back":}`) // Malformed JSON
			card := &domain.Card{
				ID:        uuid.New(),
				UserID:    testUser.ID,
				MemoID:    testMemo.ID,
				Content:   invalidJSON,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			// Insert should fail
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			assert.Error(t, err, "CreateMultiple should fail with invalid JSON content")
			assert.ErrorIs(t, err, store.ErrInvalidEntity, "Error should be ErrInvalidEntity")
			assert.ErrorContains(
				t,
				err,
				"invalid card content",
				"Error should mention invalid content",
			)
		})

		t.Run("transaction_rollback", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create 3 cards but make the last one invalid
			cards := make([]*domain.Card, 3)
			cardIDs := make([]uuid.UUID, 3)

			for i := 0; i < 2; i++ {
				content := json.RawMessage(
					`{"front":"Transaction test ` + string(rune('A'+i)) + `","back":"Test back"}`,
				)
				card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
				require.NoError(t, err, "Failed to create test card")
				cards[i] = card
				cardIDs[i] = card.ID
			}

			// Add an invalid card with non-existent user
			nonExistentUserID := uuid.New()
			content := json.RawMessage(`{"front":"Invalid user card","back":"Test back"}`)
			card, err := domain.NewCard(nonExistentUserID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")
			cards[2] = card
			cardIDs[2] = card.ID

			// Insert should fail
			err = cardStore.CreateMultiple(ctx, cards)
			assert.Error(t, err, "CreateMultiple should fail with invalid card in batch")

			// Verify none of the cards were created (transaction rolled back)
			for _, cardID := range cardIDs {
				_, err := cardStore.GetByID(ctx, cardID)
				assert.Error(t, err, "Card should not exist after rollback")
				assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")
			}
		})
	})
}
