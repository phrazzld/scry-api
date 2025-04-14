package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ReviewOutcome represents the result of a card review
type ReviewOutcome string

// Possible review outcome values
const (
	ReviewOutcomeAgain ReviewOutcome = "again"
	ReviewOutcomeHard  ReviewOutcome = "hard"
	ReviewOutcomeGood  ReviewOutcome = "good"
	ReviewOutcomeEasy  ReviewOutcome = "easy"
)

// Common validation errors for UserCardStats
var (
	ErrEmptyStatsUserID     = errors.New("user card stats user ID cannot be empty")
	ErrEmptyStatsCardID     = errors.New("user card stats card ID cannot be empty")
	ErrInvalidInterval      = errors.New("interval must be greater than or equal to 0")
	ErrInvalidEaseFactor    = errors.New("ease factor must be greater than 1.0")
	ErrInvalidReviewOutcome = errors.New("invalid review outcome")
)

// UserCardStats tracks a user's spaced repetition statistics for a specific card.
// It implements the SM-2 algorithm with some modifications for determining review intervals.
type UserCardStats struct {
	UserID             uuid.UUID `json:"user_id"`
	CardID             uuid.UUID `json:"card_id"`
	Interval           int       `json:"interval"`            // Current interval in days
	EaseFactor         float64   `json:"ease_factor"`         // Ease factor (1.3-2.5 typically)
	ConsecutiveCorrect int       `json:"consecutive_correct"` // Count of consecutive correct answers
	LastReviewedAt     time.Time `json:"last_reviewed_at"`    // When the card was last reviewed
	NextReviewAt       time.Time `json:"next_review_at"`      // When the card should be reviewed next
	ReviewCount        int       `json:"review_count"`        // Total number of reviews
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// NewUserCardStats creates new statistics for a user and card with default values.
// Initial settings are configured for immediate review of new cards.
func NewUserCardStats(userID, cardID uuid.UUID) (*UserCardStats, error) {
	now := time.Now().UTC()
	stats := &UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           0,
		EaseFactor:         2.5, // Default ease factor
		ConsecutiveCorrect: 0,
		LastReviewedAt:     time.Time{}, // Zero time
		NextReviewAt:       now,         // Card is available for review immediately
		ReviewCount:        0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := stats.Validate(); err != nil {
		return nil, err
	}

	return stats, nil
}

// Validate checks if the UserCardStats has valid data.
// Returns an error if any field fails validation.
func (s *UserCardStats) Validate() error {
	if s.UserID == uuid.Nil {
		return ErrEmptyStatsUserID
	}

	if s.CardID == uuid.Nil {
		return ErrEmptyStatsCardID
	}

	if s.Interval < 0 {
		return ErrInvalidInterval
	}

	if s.EaseFactor <= 1.0 {
		return ErrInvalidEaseFactor
	}

	return nil
}

// Note: Mutable methods UpdateReview and PostponeReview have been removed.
// Use srs.Service.CalculateNextReview and srs.Service.PostponeReview instead,
// which follow immutability principles by returning new instances rather than
// modifying existing ones.
