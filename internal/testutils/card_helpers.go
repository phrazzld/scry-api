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

// CardWithStatsOptions holds optional parameters for creating cards with stats
type CardWithStatsOptions struct {
	// The content of the card (front, back, etc.)
	CardContent map[string]interface{}
	// When the card should next be reviewed (defaults to now if not specified)
	NextReviewAt time.Time
	// Custom card ID (defaults to a new UUID if not specified)
	CardID uuid.UUID
	// Optional content for direct JSON setting (overrides CardContent if provided)
	RawContent json.RawMessage
}

// NewTestCard creates a domain.Card instance with test data.
// It does not persist the card to the database.
func NewTestCard(userID, memoID uuid.UUID, options *CardWithStatsOptions) (*domain.Card, error) {
	// Set up default options if none provided
	if options == nil {
		options = &CardWithStatsOptions{}
	}

	// Set default card ID if not provided
	if options.CardID == uuid.Nil {
		options.CardID = uuid.New()
	}

	var content json.RawMessage

	// Use provided raw content or marshal the content map
	if len(options.RawContent) > 0 {
		content = options.RawContent
	} else {
		// Set default content if not provided
		cardContent := options.CardContent
		if cardContent == nil {
			cardContent = map[string]interface{}{
				"front": "Test front",
				"back":  "Test back",
			}
		}

		var err error
		content, err = json.Marshal(cardContent)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal card content: %w", err)
		}
	}

	// Create the card
	card := &domain.Card{
		ID:        options.CardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	return card, nil
}

// NewTestUserCardStats creates a domain.UserCardStats instance with test data.
// It does not persist the stats to the database.
func NewTestUserCardStats(userID, cardID uuid.UUID, options *CardWithStatsOptions) (*domain.UserCardStats, error) {
	// Create the stats
	stats, err := domain.NewUserCardStats(userID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to create stats: %w", err)
	}

	// Set up default options if none provided
	if options == nil {
		options = &CardWithStatsOptions{}
	}

	// Set the next review time if provided
	if !options.NextReviewAt.IsZero() {
		stats.NextReviewAt = options.NextReviewAt
	}

	return stats, nil
}

// MustNewTestCard creates a new test card or fails the test if there's an error
func MustNewTestCard(t *testing.T, userID, memoID uuid.UUID, options *CardWithStatsOptions) *domain.Card {
	t.Helper()
	card, err := NewTestCard(userID, memoID, options)
	require.NoError(t, err, "Failed to create test card")
	return card
}

// MustNewTestUserCardStats creates a new test user card stats or fails the test if there's an error
func MustNewTestUserCardStats(
	t *testing.T,
	userID, cardID uuid.UUID,
	options *CardWithStatsOptions,
) *domain.UserCardStats {
	t.Helper()
	stats, err := NewTestUserCardStats(userID, cardID, options)
	require.NoError(t, err, "Failed to create test user card stats")
	return stats
}

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

// MustInsertCard inserts a card into the database for testing.
// It requires valid userID and memoID that exist in the database.
// Returns the inserted card.
func MustInsertCard(ctx context.Context, t *testing.T, tx store.DBTX, userID, memoID uuid.UUID) *domain.Card {
	t.Helper()

	// Create a test card
	card := MustNewTestCard(t, userID, memoID, nil)

	// Create a card store with transaction context
	cardStore := postgres.NewPostgresCardStore(tx, nil)

	// Insert the card - note that tx is already a transaction context
	// so we don't need to wrap it in RunInTransaction
	err := cardStore.CreateMultiple(ctx, []*domain.Card{card})
	require.NoError(t, err, "Failed to insert test card")

	return card
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
	stats := MustNewTestUserCardStats(t, userID, cardID, nil)

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
