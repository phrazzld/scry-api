package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockCardRepository mocks the CardRepository interface
type MockCardRepository struct {
	mock.Mock
}

func (m *MockCardRepository) Create(ctx context.Context, card *domain.Card) error {
	args := m.Called(ctx, card)
	return args.Error(0)
}

func (m *MockCardRepository) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	args := m.Called(ctx, cards)
	return args.Error(0)
}

func (m *MockCardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}

func (m *MockCardRepository) GetByMemoID(
	ctx context.Context,
	memoID uuid.UUID,
) ([]*domain.Card, error) {
	args := m.Called(ctx, memoID)
	return args.Get(0).([]*domain.Card), args.Error(1)
}

func (m *MockCardRepository) GetAllForUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]*domain.Card, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*domain.Card), args.Error(1)
}

func (m *MockCardRepository) GetNextDueCard(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.Card, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}

func (m *MockCardRepository) UpdateContent(
	ctx context.Context,
	id uuid.UUID,
	content json.RawMessage,
) error {
	args := m.Called(ctx, id, content)
	return args.Error(0)
}

func (m *MockCardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCardRepository) UpdateNextReview(
	ctx context.Context,
	tx interface{},
	id uuid.UUID,
	nextReview time.Time,
) error {
	args := m.Called(ctx, tx, id, nextReview)
	return args.Error(0)
}

func (m *MockCardRepository) WithTx(tx *sql.Tx) CardRepository {
	args := m.Called(tx)
	return args.Get(0).(CardRepository)
}

func (m *MockCardRepository) DB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

// MockStatsRepository mocks the StatsRepository interface
type MockStatsRepository struct {
	mock.Mock
}

func (m *MockStatsRepository) Create(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

func (m *MockStatsRepository) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	args := m.Called(ctx, userID, cardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserCardStats), args.Error(1)
}

func (m *MockStatsRepository) GetForUpdate(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	args := m.Called(ctx, userID, cardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserCardStats), args.Error(1)
}

func (m *MockStatsRepository) Update(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

func (m *MockStatsRepository) WithTx(tx *sql.Tx) StatsRepository {
	args := m.Called(tx)
	return args.Get(0).(StatsRepository)
}

// MockSRSService mocks the srs.Service interface
type MockSRSService struct {
	mock.Mock
}

func (m *MockSRSService) CalculateNextReview(
	stats *domain.UserCardStats,
	outcome domain.ReviewOutcome,
	now time.Time,
) (*domain.UserCardStats, error) {
	args := m.Called(stats, outcome, now)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserCardStats), args.Error(1)
}

func (m *MockSRSService) PostponeReview(
	stats *domain.UserCardStats,
	days int,
	now time.Time,
) (*domain.UserCardStats, error) {
	args := m.Called(stats, days, now)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserCardStats), args.Error(1)
}
