package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/mock"
)

// Note: We're skipping transaction-based tests in this package since they're better suited
// for integration tests. See card_service_tx_test.go for transaction-based testing.

// MockCardRepository is a mock implementation of the CardRepository
type MockCardRepository struct {
	mock.Mock
}

// CreateMultiple implements CardRepository
func (m *MockCardRepository) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	args := m.Called(ctx, cards)
	return args.Error(0)
}

// GetByID implements CardRepository
func (m *MockCardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	args := m.Called(ctx, id)
	card, _ := args.Get(0).(*domain.Card)
	return card, args.Error(1)
}

// WithTx implements CardRepository
func (m *MockCardRepository) WithTx(tx *sql.Tx) CardRepository {
	args := m.Called(tx)
	return args.Get(0).(CardRepository)
}

// DB implements CardRepository
func (m *MockCardRepository) DB() *sql.DB {
	args := m.Called()
	if db, ok := args.Get(0).(*sql.DB); ok {
		return db
	}
	return nil
}

// MockStatsRepository is a mock implementation of the StatsRepository
type MockStatsRepository struct {
	mock.Mock
}

// Create implements StatsRepository
func (m *MockStatsRepository) Create(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

// Get implements StatsRepository
func (m *MockStatsRepository) Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error) {
	args := m.Called(ctx, userID, cardID)
	stats, _ := args.Get(0).(*domain.UserCardStats)
	return stats, args.Error(1)
}

// Update implements StatsRepository
func (m *MockStatsRepository) Update(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

// WithTx implements StatsRepository
func (m *MockStatsRepository) WithTx(tx *sql.Tx) StatsRepository {
	args := m.Called(tx)
	return args.Get(0).(StatsRepository)
}

func TestCardService_CreateCards(t *testing.T) {
	// Skip test with transaction mocking - this would be tested in an integration test
	t.Skip("Skipping test that requires transaction management")
}

func TestCardService_GetCard(t *testing.T) {
	// Skip test with transaction mocking - this would be tested in an integration test
	t.Skip("Skipping test that requires transaction management")
}
