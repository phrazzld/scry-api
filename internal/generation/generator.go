package generation

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// Generator defines the interface for generating flashcards from text.
// This interface serves as a boundary between the application core and
// external AI/LLM services, following the hexagonal architecture pattern.
type Generator interface {
	// GenerateCards creates flashcards based on the provided memo text and user ID.
	// It returns a slice of Card domain objects or an error if generation fails.
	//
	// Parameters:
	//   - ctx: Context for the operation, which can be used for cancellation
	//   - memoText: The content of the memo to generate cards from
	//   - userID: The UUID of the user who owns the memo
	//
	// Returns:
	//   - A slice of domain.Card pointers representing the generated flashcards
	//   - An error if the generation fails for any reason (see errors.go for specific types)
	GenerateCards(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error)
}
