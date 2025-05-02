package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/require"
)

// CreateTestCard creates a test card in the database within the given transaction
// and returns the created card.
func CreateTestCard(t *testing.T, tx *sql.Tx, userID uuid.UUID) *domain.Card {
	t.Helper()

	// Create a test logger that writes to discarded output
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Create a card store with the transaction
	cardStore := postgres.NewPostgresCardStore(tx, testLogger)

	// Create a memo for the card
	memoStore := postgres.NewPostgresMemoStore(tx, testLogger)
	memo := &domain.Memo{
		ID:        uuid.New(),
		UserID:    userID,
		Text:      "Test memo content",
		Status:    domain.MemoStatusCompleted,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := memoStore.Create(context.Background(), memo)
	require.NoError(t, err, "Failed to create test memo")

	// Create a card
	cardID := uuid.New()
	card := &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memo.ID,
		Content:   json.RawMessage(`{"question": "Test question", "answer": "Test answer"}`),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Create a user_card_stats entry for this card
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, testLogger)
	stats, err := domain.NewUserCardStats(userID, cardID)
	require.NoError(t, err, "Failed to create user card stats object")
	err = statsStore.Create(context.Background(), stats)
	require.NoError(t, err, "Failed to create test user card stats")

	// Save the card
	err = cardStore.CreateMultiple(context.Background(), []*domain.Card{card})
	require.NoError(t, err, "Failed to create test card")

	return card
}

// GetCardByID retrieves a card by its ID from the database within the given transaction.
func GetCardByID(tx *sql.Tx, cardID uuid.UUID) (*domain.Card, error) {
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cardStore := postgres.NewPostgresCardStore(tx, testLogger)
	return cardStore.GetByID(context.Background(), cardID)
}

// GetUserCardStats retrieves user card statistics for a given card and user.
func GetUserCardStats(t *testing.T, tx *sql.Tx, userID, cardID uuid.UUID) *domain.UserCardStats {
	t.Helper()

	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, testLogger)
	stats, err := statsStore.Get(context.Background(), userID, cardID)

	if err != nil && errors.Is(err, store.ErrUserCardStatsNotFound) {
		return nil
	}

	require.NoError(t, err, "Failed to get user card stats")
	return stats
}
