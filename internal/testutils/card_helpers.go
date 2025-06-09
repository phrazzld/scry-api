//go:build (!compatibility && ignore_redeclarations) || test_without_external_deps

package testutils

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/require"
)

// MustInsertMemo inserts a memo into the database for testing.
//
// This helper:
// - Creates a memo with default test values and the provided userID
// - Inserts the memo into the database using the provided transaction
// - Returns the inserted memo object with all fields populated
// - Fails the test with a descriptive error if insertion fails
//
// It requires a valid userID that exists in the database and a transaction context.
// The function is particularly useful for setting up test data where memos are needed.
//
// Example:
//
//	// Insert a memo for a specific user
//	memo := testutils.MustInsertMemo(ctx, t, tx, userID)
//
//	// The memo ID can then be used to create related objects
//	card := testutils.MustInsertCard(ctx, t, tx, userID, memo.ID)
func MustInsertMemo(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	userID uuid.UUID,
) *domain.Memo {
	t.Helper()

	// Create a test memo using the new helper
	memo := MustCreateMemoForTest(t, WithMemoUserID(userID))

	// Create a memo store
	memoStore := postgres.NewPostgresMemoStore(tx, nil)

	// Insert the memo
	err := memoStore.Create(ctx, memo)
	require.NoError(t, err, "Failed to insert test memo")

	return memo
}

// MustInsertCard inserts a card into the database for testing.
//
// This helper:
// - Creates a card with default test values, linked to the provided userID and memoID
// - Inserts the card into the database using the provided transaction
// - Returns the inserted card object with all fields populated
// - Fails the test with a descriptive error if insertion fails
//
// The function requires valid userID and memoID that already exist in the database.
// It uses the PostgresCardStore.CreateMultiple method internally.
//
// Example:
//
//	// Insert a card linked to a specific user and memo
//	card := testutils.MustInsertCard(ctx, t, tx, userID, memo.ID)
//
//	// The card ID can then be used for further operations
//	stats := testutils.MustInsertUserCardStats(ctx, t, tx, userID, card.ID)
func MustInsertCard(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	userID, memoID uuid.UUID,
) *domain.Card {
	t.Helper()

	// Create a test card using the new helper
	card := MustCreateCardForTest(t,
		WithCardUserID(userID),
		WithCardMemoID(memoID),
	)

	// Create a card store with transaction context
	cardStore := postgres.NewPostgresCardStore(tx, nil)

	// Insert the card - note that tx is already a transaction context
	// so we don't need to wrap it in RunInTransaction
	err := cardStore.CreateMultiple(ctx, []*domain.Card{card})
	require.NoError(t, err, "Failed to insert test card")

	return card
}

// MustInsertUserCardStats inserts user card stats into the database for testing.
//
// This helper:
// - Creates user card stats with default test values, linked to the provided userID and cardID
// - Inserts the stats into the database using the provided transaction
// - Returns the inserted stats object with all fields populated
// - Fails the test with a descriptive error if insertion fails
//
// The function requires valid userID and cardID that already exist in the database.
// It uses the PostgresUserCardStatsStore.Update method internally since stats are
// typically upserted rather than created.
//
// Example:
//
//	// Insert stats for a specific user and card
//	stats := testutils.MustInsertUserCardStats(ctx, t, tx, userID, card.ID)
//
//	// Use stats for testing review functionality
//	assert.Equal(t, 1, stats.ReviewCount)
func MustInsertUserCardStats(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	userID, cardID uuid.UUID,
) *domain.UserCardStats {
	t.Helper()

	// Create test stats using the new helper
	stats := MustCreateStatsForTest(t,
		WithStatsUserID(userID),
		WithStatsCardID(cardID),
	)

	// Use store to insert
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
	err := statsStore.Update(ctx, stats)
	require.NoError(t, err, "Failed to insert test user card stats")

	return stats
}

// CountMemos counts the number of memos in the database matching certain criteria.
//
// This function is useful for test assertions to verify that database operations
// have the expected effect on the number of records. It executes a SQL COUNT query
// with the provided WHERE clause and arguments.
//
// Example:
//
//	// Count memos for a specific user
//	count := testutils.CountMemos(ctx, t, tx, "user_id = $1", userID)
//	assert.Equal(t, 1, count, "Should have exactly 1 memo for this user")
//
//	// Count pending memos
//	pendingCount := testutils.CountMemos(ctx, t, tx, "status = $1", domain.MemoStatusPending)
//	assert.Equal(t, 2, pendingCount, "Should have 2 pending memos")
func CountMemos(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	whereClause string,
	args ...interface{},
) int {
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
//
// This function is useful for test assertions to verify that database operations
// have the expected effect on the number of card records. It executes a SQL COUNT query
// with the provided WHERE clause and arguments.
//
// Example:
//
//	// Count cards for a specific memo
//	count := testutils.CountCards(ctx, t, tx, "memo_id = $1", memoID)
//	assert.Equal(t, 5, count, "Should have generated 5 cards for this memo")
//
//	// Count cards for a specific user created in the last hour
//	recentCount := testutils.CountCards(ctx, t, tx,
//	    "user_id = $1 AND created_at > $2",
//	    userID, time.Now().Add(-1*time.Hour))
func CountCards(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	whereClause string,
	args ...interface{},
) int {
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
//
// This function is useful for test assertions to verify that database operations
// have the expected effect on the number of user card stats records. It executes a SQL
// COUNT query with the provided WHERE clause and arguments.
//
// Example:
//
//	// Count stats for a specific user
//	count := testutils.CountUserCardStats(ctx, t, tx, "user_id = $1", userID)
//	assert.Equal(t, 10, count, "Should have stats for 10 cards for this user")
//
//	// Count stats for cards due for review
//	dueCount := testutils.CountUserCardStats(ctx, t, tx,
//	    "next_review_at < $1", time.Now())
//	assert.Equal(t, 3, dueCount, "Should have 3 cards due for review")
func CountUserCardStats(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	whereClause string,
	args ...interface{},
) int {
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
