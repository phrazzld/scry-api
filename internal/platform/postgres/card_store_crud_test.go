package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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

// TestCardStoreCRUDIntegration tests all CRUD operations for the CardStore
func TestCardStoreCRUDIntegration(t *testing.T) {
	// Skip the integration test wrapper if not in integration test environment
	if !cardTestIntegrationEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Run integration tests for each CRUD method
	t.Run("TestPostgresCardStore_GetByID", TestPostgresCardStore_GetByID)
	t.Run("TestPostgresCardStore_UpdateContent", TestPostgresCardStore_UpdateContent)
	t.Run("TestPostgresCardStore_Delete", TestPostgresCardStore_Delete)
}

// TestPostgresCardStore_GetByID tests the GetByID method
func TestPostgresCardStore_GetByID(t *testing.T) {
	// Skip if not in integration test environment
	if !cardTestIntegrationEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := getTestDB(t)
	// t.Cleanup will automatically close the connection

	localWithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("testgetbyid@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for GetByID tests")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		t.Run("existing_card", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a valid card
			content := json.RawMessage(`{"front":"GetByID test front","back":"GetByID test back"}`)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")

			// Insert the card
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			require.NoError(t, err, "Failed to insert test card")

			// Retrieve the card by ID
			retrievedCard, err := cardStore.GetByID(ctx, card.ID)
			assert.NoError(t, err, "GetByID should find the created card")
			assert.NotNil(t, retrievedCard, "Retrieved card should not be nil")
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
		})

		t.Run("non_existent_card", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Generate a random card ID that doesn't exist
			nonExistentCardID := uuid.New()

			// Try to retrieve non-existent card
			_, err := cardStore.GetByID(ctx, nonExistentCardID)
			assert.Error(t, err, "GetByID should return error for non-existent card")
			assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")
		})
	})
}

// TestPostgresCardStore_UpdateContent tests the UpdateContent method
func TestPostgresCardStore_UpdateContent(t *testing.T) {
	// Skip if not in integration test environment
	if !cardTestIntegrationEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := getTestDB(t)
	// t.Cleanup will automatically close the connection

	localWithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("testupdatecontent@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for UpdateContent tests")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		t.Run("successful_update", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a valid card
			originalContent := json.RawMessage(`{"front":"Original front","back":"Original back"}`)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, originalContent)
			require.NoError(t, err, "Failed to create test card")

			// Insert the card
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			require.NoError(t, err, "Failed to insert test card")

			// Record the original updated_at time
			originalCard, err := cardStore.GetByID(ctx, card.ID)
			require.NoError(t, err, "Failed to get original card")
			originalUpdatedAt := originalCard.UpdatedAt

			// Wait a moment to ensure the timestamps will be different
			time.Sleep(10 * time.Millisecond)

			// Update the card content
			newContent := json.RawMessage(`{"front":"Updated front","back":"Updated back"}`)
			err = cardStore.UpdateContent(ctx, card.ID, newContent)
			assert.NoError(t, err, "UpdateContent should succeed for valid card")

			// Retrieve the updated card
			updatedCard, err := cardStore.GetByID(ctx, card.ID)
			assert.NoError(t, err, "GetByID should find the updated card")
			assert.JSONEq(
				t,
				string(newContent),
				string(updatedCard.Content),
				"Card should have updated content",
			)

			// Verify updated_at was updated
			assert.True(
				t,
				updatedCard.UpdatedAt.After(originalUpdatedAt),
				"UpdatedAt should be later than original timestamp",
			)
		})

		t.Run("invalid_json_content", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a valid card
			originalContent := json.RawMessage(`{"front":"Original front","back":"Original back"}`)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, originalContent)
			require.NoError(t, err, "Failed to create test card")

			// Insert the card
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			require.NoError(t, err, "Failed to insert test card")

			// Try to update with invalid JSON
			invalidJSON := []byte(`{"front":"Invalid JSON,"back":"Missing quote"}`)
			err = cardStore.UpdateContent(ctx, card.ID, invalidJSON)
			assert.Error(t, err, "UpdateContent should fail with invalid JSON")
			assert.ErrorIs(t, err, store.ErrInvalidEntity, "Error should be ErrInvalidEntity")

			// Verify the content wasn't changed
			unchangedCard, err := cardStore.GetByID(ctx, card.ID)
			assert.NoError(t, err, "GetByID should find the original card")
			assert.JSONEq(
				t,
				string(originalContent),
				string(unchangedCard.Content),
				"Card content should not be changed after failed update",
			)
		})

		t.Run("non_existent_card", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Generate a random card ID that doesn't exist
			nonExistentCardID := uuid.New()

			// Try to update non-existent card
			newContent := json.RawMessage(`{"front":"Updated front","back":"Updated back"}`)
			err := cardStore.UpdateContent(ctx, nonExistentCardID, newContent)
			assert.Error(t, err, "UpdateContent should return error for non-existent card")
			assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")
		})
	})
}

// TestPostgresCardStore_Delete tests the Delete method
func TestPostgresCardStore_Delete(t *testing.T) {
	// Skip if not in integration test environment
	if !cardTestIntegrationEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := getTestDB(t)
	// t.Cleanup will automatically close the connection

	localWithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)
		statsStore := NewPostgresUserCardStatsStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("testdelete@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for Delete tests")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		t.Run("successful_delete", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a valid card
			content := json.RawMessage(`{"front":"Delete test front","back":"Delete test back"}`)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")

			// Insert the card
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			require.NoError(t, err, "Failed to insert test card")

			// Create stats for the card since CreateMultiple no longer does this automatically
			stats, err := domain.NewUserCardStats(testUser.ID, card.ID)
			require.NoError(t, err, "Failed to create stats for test card")
			err = statsStore.Create(ctx, stats)
			require.NoError(t, err, "Failed to insert stats for test card")

			// Verify the card exists
			_, err = cardStore.GetByID(ctx, card.ID)
			require.NoError(t, err, "Card should exist before deletion")

			// Verify the stats exist
			_, err = statsStore.Get(ctx, testUser.ID, card.ID)
			require.NoError(t, err, "Stats should exist before card deletion")

			// Delete the card
			err = cardStore.Delete(ctx, card.ID)
			assert.NoError(t, err, "Delete should succeed for existing card")

			// Verify the card no longer exists
			_, err = cardStore.GetByID(ctx, card.ID)
			assert.Error(t, err, "Card should not exist after deletion")
			assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")

			// Verify the stats were cascade deleted
			_, err = statsStore.Get(ctx, testUser.ID, card.ID)
			assert.Error(t, err, "Stats should not exist after card deletion")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
		})

		t.Run("non_existent_card", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Generate a random card ID that doesn't exist
			nonExistentCardID := uuid.New()

			// Try to delete non-existent card
			err := cardStore.Delete(ctx, nonExistentCardID)
			assert.Error(t, err, "Delete should return error for non-existent card")
			assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")
		})

		t.Run("duplicate_delete", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Create a valid card
			content := json.RawMessage(`{"front":"Duplicate delete test front","back":"Duplicate delete test back"}`)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")

			// Insert the card
			err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
			require.NoError(t, err, "Failed to insert test card")

			// Delete the card
			err = cardStore.Delete(ctx, card.ID)
			assert.NoError(t, err, "First delete should succeed")

			// Try to delete the same card again
			err = cardStore.Delete(ctx, card.ID)
			assert.Error(t, err, "Second delete should fail")
			assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")
		})
	})
}
