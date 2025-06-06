//go:build integration && !test_without_external_deps

package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/require"
)

// CreateTestMemo creates a test memo with the provided user ID.
// This is a compatibility function used in tests.
func CreateTestMemo(t *testing.T, userID uuid.UUID) *domain.Memo {
	t.Helper()

	// Create a test memo with default values
	return &domain.Memo{
		ID:        uuid.New(),
		UserID:    userID,
		Text:      "Test memo content " + uuid.New().String()[:8],
		Status:    domain.MemoStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// MustInsertMemo inserts a memo into the database for testing.
//
// This helper:
// - Creates a memo with default test values and the provided userID
// - Inserts the memo into the database using the provided transaction
// - Returns the inserted memo object with all fields populated
// - Fails the test with a descriptive error if insertion fails
func MustInsertMemo(
	ctx context.Context,
	t *testing.T,
	tx store.DBTX,
	userID uuid.UUID,
) *domain.Memo {
	t.Helper()

	// Create a test memo using the CreateTestMemo helper
	memo := CreateTestMemo(t, userID)

	// Create a memo store
	memoStore := postgres.NewPostgresMemoStore(tx, nil)

	// Insert the memo
	err := memoStore.Create(ctx, memo)
	require.NoError(t, err, "Failed to insert test memo")

	return memo
}

// CountMemos counts the number of memos in the database matching certain criteria.
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
