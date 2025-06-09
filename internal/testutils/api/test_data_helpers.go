//go:build integration || test_without_external_deps

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/stretchr/testify/require"
)

// CreateTestUser creates a test user in the database within the given transaction
// and returns the user ID.
func CreateTestUser(t *testing.T, tx *sql.Tx) uuid.UUID {
	t.Helper()

	// Generate a unique email for this test to avoid conflicts
	userEmail := "test_" + uuid.New().String() + "@example.com"
	password := "password123456"

	// Create a user store with the transaction
	userStore := postgres.NewPostgresUserStore(tx, 4) // Lower cost for tests

	// Create a new user domain object
	user, err := domain.NewUser(userEmail, password)
	require.NoError(t, err, "Failed to create user domain object")

	// Override timestamps for deterministic testing if needed
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	// Save the user to the database
	err = userStore.Create(context.Background(), user)
	require.NoError(t, err, "Failed to create test user")

	return user.ID
}

// CreateTestCard creates a test card in the database within the given transaction
// and returns the created card.
func CreateTestCard(t *testing.T, tx *sql.Tx, userID uuid.UUID) *domain.Card {
	t.Helper()

	// Create a test memo first (cards must be associated with a memo)
	memoTitle := "Test Memo " + uuid.New().String()
	memo, err := domain.NewMemo(userID, memoTitle)
	require.NoError(t, err, "Failed to create test memo")

	// Save the memo to the database
	memoStore := postgres.NewPostgresMemoStore(tx, nil)
	err = memoStore.Create(context.Background(), memo)
	require.NoError(t, err, "Failed to save test memo")

	// Create card content
	cardContent := map[string]interface{}{
		"front": "What is the capital of France?",
		"back":  "Paris",
	}
	contentBytes, err := json.Marshal(cardContent)
	require.NoError(t, err, "Failed to marshal card content")

	// Create a new card domain object
	card, err := domain.NewCard(userID, memo.ID, contentBytes)
	require.NoError(t, err, "Failed to create test card")

	// Save the card to the database
	cardStore := postgres.NewPostgresCardStore(tx, nil)
	err = cardStore.CreateMultiple(context.Background(), []*domain.Card{card})
	require.NoError(t, err, "Failed to save test card")

	// Create default stats for the card
	stats, err := domain.NewUserCardStats(userID, card.ID)
	require.NoError(t, err, "Failed to create test stats")

	// Save the stats to the database
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
	err = statsStore.Create(context.Background(), stats)
	require.NoError(t, err, "Failed to save test stats")

	return card
}

// GetUserCardStats retrieves user card statistics for a given card and user.
func GetUserCardStats(t *testing.T, tx *sql.Tx, userID, cardID uuid.UUID) *domain.UserCardStats {
	t.Helper()

	statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
	stats, err := statsStore.Get(context.Background(), userID, cardID)
	require.NoError(t, err, "Failed to get user card stats")

	return stats
}
