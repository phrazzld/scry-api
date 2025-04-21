package card_review

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// ReviewAnswer represents a user's answer to a flashcard review.
type ReviewAnswer struct {
	Outcome domain.ReviewOutcome `json:"outcome"` // The outcome selected by the user
}

// CardReviewService provides methods for reviewing flashcards
// using a spaced repetition algorithm.
type CardReviewService interface {
	// GetNextCard retrieves the next card due for review for a user.
	// It uses the spaced repetition algorithm to determine which card
	// should be reviewed next based on review history and scheduling.
	//
	// Parameters:
	//   - ctx: Context for the operation, which can include correlation ID and cancellation
	//   - userID: UUID of the user requesting the next card
	//
	// Returns:
	//   - (*domain.Card, nil): The next card due for review if one exists
	//   - (nil, ErrNoCardsDue): If the user has no cards due for review
	//   - (nil, error): Any other error, typically from the database or validation
	//
	// Error Handling:
	//   - Returns ErrNoCardsDue when the user has no cards due for review
	//   - Database errors are logged and wrapped with appropriate service-level errors
	//
	// This method is a thin wrapper around the store layer and does not modify any data.
	GetNextCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)

	// SubmitAnswer processes a user's answer for a flashcard and updates the
	// review schedule based on the spaced repetition algorithm.
	//
	// This method performs several operations within a single transaction:
	// 1. Verifies the card exists and belongs to the user
	// 2. Updates the user's statistics for the card using the SRS algorithm
	// 3. Calculates the next review date based on the answer outcome
	//
	// Parameters:
	//   - ctx: Context for the operation, which can include correlation ID and cancellation
	//   - userID: UUID of the user submitting the answer
	//   - cardID: UUID of the card being reviewed
	//   - answer: ReviewAnswer containing the outcome (again, hard, good, easy)
	//
	// Returns:
	//   - (*domain.UserCardStats, nil): Updated user card statistics
	//   - (nil, ErrCardNotFound): If the card does not exist
	//   - (nil, ErrCardNotOwned): If the user does not own the card
	//   - (nil, error): Any other error, typically from validation or the database
	//
	// Error Handling:
	//   - Returns ErrCardNotFound when the card does not exist
	//   - Returns ErrCardNotOwned when the user does not own the card
	//   - Returns ErrInvalidAnswer when the outcome is invalid
	//   - Database errors are logged and wrapped with appropriate service-level errors
	//
	// This method modifies data and MUST be executed within a transaction for
	// proper atomicity and data consistency.
	SubmitAnswer(
		ctx context.Context,
		userID uuid.UUID,
		cardID uuid.UUID,
		answer ReviewAnswer,
	) (*domain.UserCardStats, error)
}

// Common error types for CardReviewService
var (
	// ErrNoCardsDue indicates that the user has no cards due for review.
	ErrNoCardsDue = errors.New("no cards due for review")

	// ErrCardNotFound indicates that the card does not exist.
	ErrCardNotFound = errors.New("card not found")

	// ErrCardStatsNotFound indicates that the card statistics do not exist.
	ErrCardStatsNotFound = errors.New("card stats not found")

	// ErrCardNotOwned indicates that the user does not own the card.
	ErrCardNotOwned = errors.New("unauthorized access: card not owned by user")

	// ErrInvalidAnswer indicates an invalid answer was provided.
	ErrInvalidAnswer = errors.New("invalid answer")
)
