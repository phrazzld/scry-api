package testutils

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
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
