package mocks

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
)

// MockCardReviewService implements card_review.CardReviewService for testing
type MockCardReviewService struct {
	// Custom behavior functions
	GetNextCardFn  func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
	SubmitAnswerFn func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error)

	// Default response values
	NextCard     *domain.Card
	UpdatedStats *domain.UserCardStats
	Err          error

	// Call tracking for verification
	GetNextCardCalls struct {
		mu       sync.Mutex
		Count    int
		UserIDs  []uuid.UUID
		Contexts []context.Context
	}

	SubmitAnswerCalls struct {
		mu       sync.Mutex
		Count    int
		UserIDs  []uuid.UUID
		CardIDs  []uuid.UUID
		Answers  []card_review.ReviewAnswer
		Contexts []context.Context
	}
}

// GetNextCard implements the card_review.CardReviewService interface
func (m *MockCardReviewService) GetNextCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	// Track call details for verification
	m.GetNextCardCalls.mu.Lock()
	m.GetNextCardCalls.Count++
	m.GetNextCardCalls.UserIDs = append(m.GetNextCardCalls.UserIDs, userID)
	m.GetNextCardCalls.Contexts = append(m.GetNextCardCalls.Contexts, ctx)
	m.GetNextCardCalls.mu.Unlock()

	// Use custom function if provided
	if m.GetNextCardFn != nil {
		return m.GetNextCardFn(ctx, userID)
	}

	// Return default values
	return m.NextCard, m.Err
}

// SubmitAnswer implements the card_review.CardReviewService interface
func (m *MockCardReviewService) SubmitAnswer(
	ctx context.Context,
	userID uuid.UUID,
	cardID uuid.UUID,
	answer card_review.ReviewAnswer,
) (*domain.UserCardStats, error) {
	// Track call details for verification
	m.SubmitAnswerCalls.mu.Lock()
	m.SubmitAnswerCalls.Count++
	m.SubmitAnswerCalls.UserIDs = append(m.SubmitAnswerCalls.UserIDs, userID)
	m.SubmitAnswerCalls.CardIDs = append(m.SubmitAnswerCalls.CardIDs, cardID)
	m.SubmitAnswerCalls.Answers = append(m.SubmitAnswerCalls.Answers, answer)
	m.SubmitAnswerCalls.Contexts = append(m.SubmitAnswerCalls.Contexts, ctx)
	m.SubmitAnswerCalls.mu.Unlock()

	// Use custom function if provided
	if m.SubmitAnswerFn != nil {
		return m.SubmitAnswerFn(ctx, userID, cardID, answer)
	}

	// Return default values
	return m.UpdatedStats, m.Err
}

// Reset resets the call tracking state for both methods
func (m *MockCardReviewService) Reset() {
	m.GetNextCardCalls.mu.Lock()
	m.GetNextCardCalls.Count = 0
	m.GetNextCardCalls.UserIDs = nil
	m.GetNextCardCalls.Contexts = nil
	m.GetNextCardCalls.mu.Unlock()

	m.SubmitAnswerCalls.mu.Lock()
	m.SubmitAnswerCalls.Count = 0
	m.SubmitAnswerCalls.UserIDs = nil
	m.SubmitAnswerCalls.CardIDs = nil
	m.SubmitAnswerCalls.Answers = nil
	m.SubmitAnswerCalls.Contexts = nil
	m.SubmitAnswerCalls.mu.Unlock()
}

// Functional option pattern for configuring mock

// MockOption is a function type that configures a MockCardReviewService
type MockOption func(*MockCardReviewService)

// WithNextCard sets the default card to return from GetNextCard
func WithNextCard(card *domain.Card) MockOption {
	return func(m *MockCardReviewService) {
		m.NextCard = card
	}
}

// WithUpdatedStats sets the default stats to return from SubmitAnswer
func WithUpdatedStats(stats *domain.UserCardStats) MockOption {
	return func(m *MockCardReviewService) {
		m.UpdatedStats = stats
	}
}

// WithError sets the default error to return from both methods
func WithError(err error) MockOption {
	return func(m *MockCardReviewService) {
		m.Err = err
	}
}

// WithGetNextCardFn sets a custom function for GetNextCard
func WithGetNextCardFn(fn func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)) MockOption {
	return func(m *MockCardReviewService) {
		m.GetNextCardFn = fn
	}
}

// WithSubmitAnswerFn sets a custom function for SubmitAnswer
func WithSubmitAnswerFn(
	fn func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error),
) MockOption {
	return func(m *MockCardReviewService) {
		m.SubmitAnswerFn = fn
	}
}

// NewMockCardReviewService creates a new MockCardReviewService with the given options
func NewMockCardReviewService(opts ...MockOption) *MockCardReviewService {
	mock := &MockCardReviewService{}

	// Apply all options
	for _, opt := range opts {
		opt(mock)
	}

	return mock
}

// Convenience constructors for common test scenarios

// NewMockCardReviewServiceWithNoCardsDue returns a mock that simulates no cards due for review
func NewMockCardReviewServiceWithNoCardsDue() *MockCardReviewService {
	return NewMockCardReviewService(
		WithError(card_review.ErrNoCardsDue),
	)
}

// NewMockCardReviewServiceWithCardNotFound returns a mock that simulates card not found
func NewMockCardReviewServiceWithCardNotFound() *MockCardReviewService {
	return NewMockCardReviewService(
		WithError(card_review.ErrCardNotFound),
	)
}

// NewMockCardReviewServiceWithCardNotOwned returns a mock that simulates card not owned by user
func NewMockCardReviewServiceWithCardNotOwned() *MockCardReviewService {
	return NewMockCardReviewService(
		WithError(card_review.ErrCardNotOwned),
	)
}

// NewMockCardReviewServiceWithInvalidAnswer returns a mock that simulates an invalid answer
func NewMockCardReviewServiceWithInvalidAnswer() *MockCardReviewService {
	return NewMockCardReviewService(
		WithError(card_review.ErrInvalidAnswer),
	)
}
