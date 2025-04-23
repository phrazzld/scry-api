package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/require"
)

// CreateTestMemo creates a new valid memo for testing.
// It does not save the memo to the database.
func CreateTestMemo(t *testing.T, userID uuid.UUID) *domain.Memo {
	t.Helper()

	memo, err := domain.NewMemo(
		userID,
		fmt.Sprintf("Test memo content %s", uuid.New().String()[:8]),
	)
	require.NoError(t, err, "Failed to create test memo")
	return memo
}

// MustInsertMemo inserts a memo into the database for testing.
// It requires a valid userID that exists in the database.
// Returns the inserted memo.
func MustInsertMemo(ctx context.Context, t *testing.T, tx store.DBTX, userID uuid.UUID) *domain.Memo {
	t.Helper()

	// Create a test memo
	memo := CreateTestMemo(t, userID)

	// Create a memo store
	memoStore := postgres.NewPostgresMemoStore(tx, nil)

	// Insert the memo
	err := memoStore.Create(ctx, memo)
	require.NoError(t, err, "Failed to insert test memo")

	return memo
}

// CreateTestCard creates a new valid card for testing.
// It does not save the card to the database.
func CreateTestCard(t *testing.T, userID, memoID uuid.UUID) *domain.Card {
	t.Helper()

	content := map[string]interface{}{
		"front": "Test question " + uuid.New().String()[:8],
		"back":  "Test answer " + uuid.New().String()[:8],
	}
	contentJSON, err := json.Marshal(content)
	require.NoError(t, err, "Failed to marshal card content")

	card := &domain.Card{
		ID:        uuid.New(),
		UserID:    userID,
		MemoID:    memoID,
		Content:   contentJSON,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	return card
}

// MustInsertCard inserts a card into the database for testing.
// It requires valid userID and memoID that exist in the database.
// Returns the inserted card.
func MustInsertCard(ctx context.Context, t *testing.T, tx store.DBTX, userID, memoID uuid.UUID) *domain.Card {
	t.Helper()

	// Create a test card
	card := CreateTestCard(t, userID, memoID)

	// Create a card store with transaction context
	cardStore := postgres.NewPostgresCardStore(tx, nil)

	// Insert the card - note that tx is already a transaction context
	// so we don't need to wrap it in RunInTransaction
	err := cardStore.CreateMultiple(ctx, []*domain.Card{card})
	require.NoError(t, err, "Failed to insert test card")

	return card
}

// CreateTestUserCardStats creates a new valid user card stats for testing.
// It does not save the stats to the database.
func CreateTestUserCardStats(t *testing.T, userID, cardID uuid.UUID) *domain.UserCardStats {
	t.Helper()

	// Create a test user card stats object
	stats, err := domain.NewUserCardStats(userID, cardID)
	require.NoError(t, err, "Failed to create test user card stats")

	return stats
}

// MustInsertUserCardStats inserts user card stats into the database for testing.
// It requires valid userID and cardID that exist in the database.
// Returns the inserted stats.
func MustInsertUserCardStats(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	userID, cardID uuid.UUID,
) *domain.UserCardStats {
	t.Helper()

	// Create test stats
	stats := CreateTestUserCardStats(t, userID, cardID)

	// Use store to insert
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
	err := statsStore.Update(ctx, stats)
	require.NoError(t, err, "Failed to insert test user card stats")

	return stats
}

// CountMemos counts the number of memos in the database matching certain criteria.
func CountMemos(ctx context.Context, t *testing.T, tx store.DBTX, whereClause string, args ...interface{}) int {
	t.Helper()

	query := "SELECT COUNT(*) FROM memos"
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err, "Failed to count memos")

	return count
}

// CountCards counts the number of cards in the database matching certain criteria.
func CountCards(ctx context.Context, t *testing.T, tx store.DBTX, whereClause string, args ...interface{}) int {
	t.Helper()

	query := "SELECT COUNT(*) FROM cards"
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err, "Failed to count cards")

	return count
}

// CountUserCardStats counts the number of user card stats in the database matching certain criteria.
func CountUserCardStats(ctx context.Context, t *testing.T, tx store.DBTX, whereClause string, args ...interface{}) int {
	t.Helper()

	query := "SELECT COUNT(*) FROM user_card_stats"
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err, "Failed to count user card stats")

	return count
}

// The helpers from this file have been moved to card_helpers.go to avoid import cycles
// This allows test packages to use the helpers without causing import cycles
