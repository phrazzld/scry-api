package srs

import (
	"time"

	"github.com/phrazzld/scry-api/internal/domain"
)

// calculateNewEaseFactor determines the new ease factor based on the review outcome.
//
// The ease factor represents the card's difficulty - higher values mean the card
// is easier and intervals will grow faster. This function adjusts the ease factor
// based on the user's review outcome (Again, Hard, Good, Easy) using predefined
// adjustments from the params.
//
// Parameters:
//   - currentEF: The current ease factor of the card, typically between 1.3 and 2.5
//   - outcome: The user's review outcome (Again, Hard, Good, Easy)
//   - params: Configuration parameters for the SRS algorithm
//
// Returns:
//   - The new ease factor value, clamped between params.MinEaseFactor and params.MaxEaseFactor
//
// Algorithm behavior:
//   - "Again" outcomes significantly decrease ease factor (typically -0.20)
//   - "Hard" outcomes moderately decrease ease factor (typically -0.15)
//   - "Good" outcomes leave ease factor unchanged (0.0)
//   - "Easy" outcomes moderately increase ease factor (typically +0.15)
//   - The result is always clamped to prevent excessively hard or easy cards
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
//
// This function is a core part of the SRS algorithm that calculates how many days
// should pass until the next review of a card. It handles several special cases
// and applies different multipliers based on the review outcome.
//
// Parameters:
//   - currentInterval: The current interval in days
//   - consecutiveCorrect: Number of times the card has been answered correctly in a row
//   - easeFactor: The card's current ease factor (difficulty modifier)
//   - outcome: The user's review outcome (Again, Hard, Good, Easy)
//   - params: Configuration parameters for the SRS algorithm
//
// Returns:
//   - An integer representing the new interval in days
//
// Algorithm behavior:
//   - "Again" outcome: Resets interval to 0 (meaning review in minutes, not days)
//   - First reviews (currentInterval = 0): Uses predefined intervals from params
//   - After a lapse (consecutiveCorrect = 0 but interval > 0): Uses special 1.5 multiplier for "Good" outcome
//   - Normal case for "Good" outcome: Multiplies interval by ease factor
//   - "Hard" outcome: Uses a smaller multiplier (typically 1.2)
//   - "Easy" outcome: Uses a larger multiplier (typically 1.3) plus the ease factor
//
// This implementation follows an SM-2 variant with modifications for better handling
// of lapses and more granular control over interval growth.
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
//
// This function converts the calculated interval into an actual future date/time
// for the next review. It handles the special case of "Again" outcomes, which use
// minutes rather than days for the next review.
//
// Parameters:
//   - interval: The interval in days calculated by calculateNewInterval
//   - outcome: The user's review outcome (Again, Hard, Good, Easy)
//   - now: The current time, usually the time when the review was performed
//   - params: Configuration parameters for the SRS algorithm
//
// Returns:
//   - A time.Time value representing when the card should next be reviewed
//
// Algorithm behavior:
//   - For "Again" outcomes: The card is scheduled for review in params.AgainReviewMinutes
//     (typically 10 minutes) from now
//   - For all other outcomes: The card is scheduled for review after the calculated
//     interval in days from now
//
// This approach ensures failed cards are reviewed again quickly for better learning
// reinforcement, while successful reviews lead to progressively longer intervals.
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
//
// This function orchestrates the full process of calculating the next state of a card
// after a review, following immutability principles by creating a new stats object
// rather than modifying the existing one. It coordinates the calls to the other algorithm
// functions (calculateNewEaseFactor, calculateNewInterval, calculateNextReviewDate).
//
// Parameters:
//   - stats: The current UserCardStats object
//   - outcome: The user's review outcome (Again, Hard, Good, Easy)
//   - now: The current time, usually the time when the review was performed
//   - params: Configuration parameters for the SRS algorithm
//
// Returns:
//   - A new UserCardStats object with updated values
//
// Algorithm behavior:
//   - Creates a complete copy of the original stats to maintain immutability
//   - Increments review count
//   - Updates last reviewed time to now
//   - Calculates new ease factor based on outcome
//   - Updates consecutive correct count (reset on "Again", increment otherwise)
//   - Calculates new interval using current stats and new ease factor
//   - Determines next review date based on the new interval
//   - Updates the updated timestamp to now
//
// This function implements the immutable update pattern - instead of modifying the
// existing stats object, it creates and returns a completely new object. This approach
// is beneficial for tracking history, avoiding side effects, and simplifying testing.
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
