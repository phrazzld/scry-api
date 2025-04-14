package srs

import (
	"time"

	"github.com/phrazzld/scry-api/internal/domain"
)

// calculateNewEaseFactor determines the new ease factor based on the review outcome.
func calculateNewEaseFactor(
	currentEF float64,
	outcome domain.ReviewOutcome,
	params *Params,
) float64 {
	// Apply the adjustment for the given outcome
	adjustment := params.EaseFactorAdjustment[outcome]
	newEF := currentEF + adjustment

	// Ensure ease factor stays within configured limits
	if newEF < params.MinEaseFactor {
		newEF = params.MinEaseFactor
	}
	if newEF > params.MaxEaseFactor {
		newEF = params.MaxEaseFactor
	}

	return newEF
}

// calculateNewInterval determines the new interval based on the review outcome and current stats.
func calculateNewInterval(
	currentInterval int,
	consecutiveCorrect int,
	easeFactor float64,
	outcome domain.ReviewOutcome,
	params *Params,
) int {
	// Handle "Again" outcome - reset interval
	if outcome == domain.ReviewOutcomeAgain {
		return 0
	}

	// Special case for first review or after a reset
	if currentInterval == 0 {
		// Use pre-configured initial intervals for first reviews
		return params.FirstReviewIntervals[outcome]
	}

	// After a lapse (consecutiveCorrect is 0 but interval > 0),
	// use a special modifier for the Good outcome
	if consecutiveCorrect == 0 && outcome == domain.ReviewOutcomeGood {
		return int(float64(currentInterval) * 1.5)
	}

	// Normal case: apply the outcome-specific modifier and ease factor
	var modifier float64
	if outcome == domain.ReviewOutcomeGood {
		// For Good outcome, use the ease factor directly
		modifier = easeFactor
	} else {
		// For other outcomes, use the configured modifier
		modifier = params.IntervalModifier[outcome]

		// For Easy outcome, also multiply by ease factor
		if outcome == domain.ReviewOutcomeEasy {
			modifier *= easeFactor
		}
	}

	return int(float64(currentInterval) * modifier)
}

// calculateNextReviewDate determines when the card should next be reviewed.
func calculateNextReviewDate(
	interval int,
	outcome domain.ReviewOutcome,
	now time.Time,
	params *Params,
) time.Time {
	// For "Again" outcome, review again in a few minutes
	if outcome == domain.ReviewOutcomeAgain {
		return now.Add(time.Duration(params.AgainReviewMinutes) * time.Minute)
	}

	// For other outcomes, review after the calculated interval
	return now.AddDate(0, 0, interval)
}

// calculateNextStats creates a new UserCardStats with updated values based on the review outcome.
func calculateNextStats(
	stats *domain.UserCardStats,
	outcome domain.ReviewOutcome,
	now time.Time,
	params *Params,
) *domain.UserCardStats {
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
		UpdatedAt:          stats.UpdatedAt,
	}

	// Increment review count
	newStats.ReviewCount++

	// Update last reviewed time
	newStats.LastReviewedAt = now

	// Calculate new ease factor
	newStats.EaseFactor = calculateNewEaseFactor(stats.EaseFactor, outcome, params)

	// Update consecutive correct count
	if outcome == domain.ReviewOutcomeAgain {
		newStats.ConsecutiveCorrect = 0
	} else {
		newStats.ConsecutiveCorrect++
	}

	// Calculate new interval
	newStats.Interval = calculateNewInterval(
		stats.Interval,
		stats.ConsecutiveCorrect,
		newStats.EaseFactor,
		outcome,
		params,
	)

	// Calculate next review date
	newStats.NextReviewAt = calculateNextReviewDate(newStats.Interval, outcome, now, params)

	// Update the updated timestamp
	newStats.UpdatedAt = now

	return newStats
}
