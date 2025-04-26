package mocks

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
)

// MockGenerator implements generation.Generator for testing
type MockGenerator struct {
	// GenerateCardsFn allows test cases to mock the GenerateCards behavior
	GenerateCardsFn func(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error)

	// Default response values
	Cards []*domain.Card
	Err   error

	// Call tracking for verification
	GenerateCardsCalls struct {
		// mu protects the call tracking state for concurrent test cases
		mu sync.Mutex

		// Count tracks how many times GenerateCards was called
		Count int

		// MemoTexts contains all memo texts passed to GenerateCards calls
		MemoTexts []string

		// UserIDs contains all userIDs passed to GenerateCards calls
		UserIDs []uuid.UUID

		// Contexts contains all contexts passed to GenerateCards calls
		Contexts []context.Context
	}
}

// GenerateCards implements the generation.Generator interface
func (m *MockGenerator) GenerateCards(
	ctx context.Context,
	memoText string,
	userID uuid.UUID,
) ([]*domain.Card, error) {
	// Track call details for verification
	m.GenerateCardsCalls.mu.Lock()
	m.GenerateCardsCalls.Count++
	m.GenerateCardsCalls.MemoTexts = append(m.GenerateCardsCalls.MemoTexts, memoText)
	m.GenerateCardsCalls.UserIDs = append(m.GenerateCardsCalls.UserIDs, userID)
	m.GenerateCardsCalls.Contexts = append(m.GenerateCardsCalls.Contexts, ctx)
	m.GenerateCardsCalls.mu.Unlock()

	// Use custom function if provided
	if m.GenerateCardsFn != nil {
		return m.GenerateCardsFn(ctx, memoText, userID)
	}

	// Return default values
	return m.Cards, m.Err
}

// NewMockGeneratorWithCards creates a MockGenerator that returns the specified cards
func NewMockGeneratorWithCards(cards []*domain.Card) *MockGenerator {
	return &MockGenerator{
		Cards: cards,
	}
}

// NewMockGeneratorWithError creates a MockGenerator that returns the specified error
func NewMockGeneratorWithError(err error) *MockGenerator {
	return &MockGenerator{
		Err: err,
	}
}

// NewMockGeneratorWithDefaultCards creates a MockGenerator with sample cards
func NewMockGeneratorWithDefaultCards(memoID, userID uuid.UUID) *MockGenerator {
	// Create some default card content
	card1Content := domain.CardContent{
		Front: "What is hexagonal architecture?",
		Back:  "An architectural pattern that isolates the domain from external concerns.",
		Tags:  []string{"architecture", "design"},
	}
	card2Content := domain.CardContent{
		Front: "What is Dependency Inversion?",
		Back:  "A principle where high-level modules don't depend on low-level modules; both depend on abstractions.",
		Tags:  []string{"design", "SOLID"},
	}

	// Convert to JSON
	content1, _ := json.Marshal(card1Content)
	content2, _ := json.Marshal(card2Content)

	// Create cards
	card1, _ := domain.NewCard(userID, memoID, content1)
	card2, _ := domain.NewCard(userID, memoID, content2)

	// Return mock generator with these cards
	return &MockGenerator{
		Cards: []*domain.Card{card1, card2},
	}
}

// MockGeneratorThatFails creates a MockGenerator that simulates a generation failure
func MockGeneratorThatFails() *MockGenerator {
	return &MockGenerator{
		Err: generation.ErrGenerationFailed,
	}
}

// MockGeneratorWithTransientFailure creates a MockGenerator that simulates a transient failure
func MockGeneratorWithTransientFailure() *MockGenerator {
	return &MockGenerator{
		Err: generation.ErrTransientFailure,
	}
}

// MockGeneratorWithContentBlocked creates a MockGenerator that simulates content being blocked
func MockGeneratorWithContentBlocked() *MockGenerator {
	return &MockGenerator{
		Err: generation.ErrContentBlocked,
	}
}

// Reset resets the call tracking state
func (m *MockGenerator) Reset() {
	m.GenerateCardsCalls.mu.Lock()
	defer m.GenerateCardsCalls.mu.Unlock()

	m.GenerateCardsCalls.Count = 0
	m.GenerateCardsCalls.MemoTexts = nil
	m.GenerateCardsCalls.UserIDs = nil
	m.GenerateCardsCalls.Contexts = nil
}
