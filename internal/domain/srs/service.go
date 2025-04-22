package srs

import (
	"errors"
	"time"

	"github.com/phrazzld/scry-api/internal/domain"
)

// Common errors
var (
	ErrNilStats       = errors.New("user card stats cannot be nil")
	ErrInvalidOutcome = errors.New("invalid review outcome")
	ErrInvalidDays    = errors.New("postpone days must be at least 1")
)

// Service defines the interface for SRS algorithm operations
type Service interface {
	// CalculateNextReview computes new stats based on a review outcome
	CalculateNextReview(
		stats *domain.UserCardStats,
		outcome domain.ReviewOutcome,
		now time.Time,
	) (*domain.UserCardStats, error)

	// PostponeReview pushes the next review time forward by a specified number of days
	PostponeReview(
		stats *domain.UserCardStats,
		days int,
		now time.Time,
	) (*domain.UserCardStats, error)
}

// defaultService is the standard implementation of the Service interface
type defaultService struct {
	params *Params
}

// NewDefaultService creates a new SRS service with default parameters
// This function will never return an error as it uses default parameters, but
// maintains the error return for consistency with other service constructors.
func NewDefaultService() (Service, error) {
	return &defaultService{
		params: NewDefaultParams(),
	}, nil
}

// NewServiceWithParams creates a new SRS service with custom parameters
// It returns an error if the provided parameters are nil.
func NewServiceWithParams(params *Params) (Service, error) {
	if params == nil {
		return nil, errors.New("params cannot be nil")
	}

	return &defaultService{
		params: params,
	}, nil
}

// CalculateNextReview implements the Service interface for calculating updated stats
func (s *defaultService) CalculateNextReview(
	stats *domain.UserCardStats,
	outcome domain.ReviewOutcome,
	now time.Time,
) (*domain.UserCardStats, error) {
	// Validate inputs
	if stats == nil {
		return nil, ErrNilStats
	}

	if !isValidOutcome(outcome) {
		return nil, ErrInvalidOutcome
	}

	// Use the pure calculation function to get new stats
	newStats := calculateNextStats(stats, outcome, now, s.params)

	return newStats, nil
}

// PostponeReview implements the Service interface for postponing reviews
func (s *defaultService) PostponeReview(
	stats *domain.UserCardStats,
	days int,
	now time.Time,
) (*domain.UserCardStats, error) {
	// Validate inputs
	if stats == nil {
		return nil, ErrNilStats
	}

	if days < 1 {
		return nil, ErrInvalidDays
	}

	// Create a copy of the original stats
	newStats := &domain.UserCardStats{
		UserID:             stats.UserID,
		CardID:             stats.CardID,
		Interval:           stats.Interval,
		EaseFactor:         stats.EaseFactor,
		ConsecutiveCorrect: stats.ConsecutiveCorrect,
		LastReviewedAt:     stats.LastReviewedAt,
		NextReviewAt:       stats.NextReviewAt,
		ReviewCount:        stats.ReviewCount,
		CreatedAt:          stats.CreatedAt,
		UpdatedAt:          now, // Update the updated timestamp
	}

	// Postpone the next review
	newStats.NextReviewAt = stats.NextReviewAt.AddDate(0, 0, days)

	return newStats, nil
}

// isValidOutcome checks if the given outcome is valid
func isValidOutcome(outcome domain.ReviewOutcome) bool {
	switch outcome {
	case domain.ReviewOutcomeAgain,
		domain.ReviewOutcomeHard,
		domain.ReviewOutcomeGood,
		domain.ReviewOutcomeEasy:
		return true
	default:
		return false
	}
}
