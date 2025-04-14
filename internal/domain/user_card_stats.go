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

// UpdateReview updates the card statistics based on a review outcome.
// It applies the SM-2 algorithm to calculate the new interval and ease factor.
func (s *UserCardStats) UpdateReview(outcome ReviewOutcome) error {
	if !isValidReviewOutcome(outcome) {
		return ErrInvalidReviewOutcome
	}

	now := time.Now().UTC()

	// Update review count and last reviewed time
	s.ReviewCount++
	s.LastReviewedAt = now

	// Apply SM-2 algorithm logic with modifications
	switch outcome {
	case ReviewOutcomeAgain:
		// Reset interval and reduce ease factor
		s.Interval = 0
		s.EaseFactor = max(1.3, s.EaseFactor-0.20)
		s.ConsecutiveCorrect = 0
		s.NextReviewAt = now.Add(time.Minute * 10) // Review again in 10 minutes

	case ReviewOutcomeHard:
		// Small interval increase, slight ease factor reduction
		if s.Interval == 0 {
			s.Interval = 1
		} else {
			s.Interval = int(float64(s.Interval) * 1.2)
		}
		s.EaseFactor = max(1.3, s.EaseFactor-0.15)
		s.ConsecutiveCorrect++
		s.NextReviewAt = now.AddDate(0, 0, s.Interval)

	case ReviewOutcomeGood:
		// Normal interval increase
		if s.Interval == 0 {
			s.Interval = 1
		} else if s.ConsecutiveCorrect == 0 {
			s.Interval = int(float64(s.Interval) * 1.5)
		} else {
			s.Interval = int(float64(s.Interval) * s.EaseFactor)
		}
		s.ConsecutiveCorrect++
		s.NextReviewAt = now.AddDate(0, 0, s.Interval)

	case ReviewOutcomeEasy:
		// Larger interval increase, ease factor boost
		if s.Interval == 0 {
			s.Interval = 2
		} else if s.ConsecutiveCorrect == 0 {
			s.Interval = int(float64(s.Interval) * 2.0)
		} else {
			s.Interval = int(float64(s.Interval) * s.EaseFactor * 1.3)
		}
		s.EaseFactor = min(2.5, s.EaseFactor+0.15)
		s.ConsecutiveCorrect++
		s.NextReviewAt = now.AddDate(0, 0, s.Interval)
	}

	s.UpdatedAt = now
	return nil
}

// PostponeReview pushes the next review time by the specified number of days.
func (s *UserCardStats) PostponeReview(days int) error {
	if days < 1 {
		return errors.New("postpone days must be at least 1")
	}

	s.NextReviewAt = s.NextReviewAt.AddDate(0, 0, days)
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// isValidReviewOutcome checks if the given outcome is a valid ReviewOutcome.
func isValidReviewOutcome(outcome ReviewOutcome) bool {
	switch outcome {
	case ReviewOutcomeAgain, ReviewOutcomeHard, ReviewOutcomeGood, ReviewOutcomeEasy:
		return true
	default:
		return false
	}
}

// min returns the smaller of a and b.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of a and b.
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
