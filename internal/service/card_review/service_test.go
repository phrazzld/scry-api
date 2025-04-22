package card_review_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCardRepository is a mock implementation of the CardRepository interface
type MockCardRepository struct {
	mock.Mock
}

func (m *MockCardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}

func (m *MockCardRepository) GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}

func (m *MockCardRepository) WithTx(tx *sql.Tx) card_review.CardRepository {
	args := m.Called(tx)
	return args.Get(0).(card_review.CardRepository)
}

func (m *MockCardRepository) DB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

// MockUserCardStatsRepository is a mock implementation of the UserCardStatsRepository interface
type MockUserCardStatsRepository struct {
	mock.Mock
}

func (m *MockUserCardStatsRepository) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	args := m.Called(ctx, userID, cardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserCardStats), args.Error(1)
}

func (m *MockUserCardStatsRepository) Create(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

func (m *MockUserCardStatsRepository) Update(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

func (m *MockUserCardStatsRepository) WithTx(tx *sql.Tx) card_review.UserCardStatsRepository {
	args := m.Called(tx)
	return args.Get(0).(card_review.UserCardStatsRepository)
}

// MockSRSService is a mock implementation of the srs.Service interface
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

// Helper function to create a test card
func createTestCard(userID uuid.UUID) *domain.Card {
	cardID := uuid.New()
	memoID := uuid.New()
	return &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   []byte(`{"front":"Test Question","back":"Test Answer"}`),
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
}

// TestGetNextCard tests the GetNextCard method of CardReviewService
func TestGetNextCard(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name           string
		userID         uuid.UUID
		setupMock      func(*MockCardRepository, uuid.UUID)
		expectedError  error
		expectedErrMsg string
	}{
		{
			name:   "happy path - card found",
			userID: uuid.New(),
			setupMock: func(repo *MockCardRepository, userID uuid.UUID) {
				card := createTestCard(userID)
				repo.On("GetNextReviewCard", mock.Anything, userID).Return(card, nil)
			},
			expectedError:  nil,
			expectedErrMsg: "",
		},
		{
			name:   "no cards due",
			userID: uuid.New(),
			setupMock: func(repo *MockCardRepository, userID uuid.UUID) {
				repo.On("GetNextReviewCard", mock.Anything, userID).Return(nil, card_review.ErrCardNotFound)
			},
			expectedError:  card_review.ErrNoCardsDue,
			expectedErrMsg: "",
		},
		{
			name:   "repository error",
			userID: uuid.New(),
			setupMock: func(repo *MockCardRepository, userID uuid.UUID) {
				repo.On("GetNextReviewCard", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			expectedError:  nil,
			expectedErrMsg: "failed to get next review card: database error",
		},
		{
			name:   "nil uuid",
			userID: uuid.Nil,
			setupMock: func(repo *MockCardRepository, userID uuid.UUID) {
				repo.On("GetNextReviewCard", mock.Anything, userID).Return(nil, card_review.ErrCardNotFound)
			},
			expectedError:  card_review.ErrNoCardsDue,
			expectedErrMsg: "",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock repositories
			mockCardRepo := new(MockCardRepository)
			mockStatsRepo := new(MockUserCardStatsRepository)
			mockSrsService := new(MockSRSService)
			tc.setupMock(mockCardRepo, tc.userID)

			// Create no-op logger for testing
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Create service with all required dependencies
			service := card_review.NewCardReviewService(
				mockCardRepo,
				mockStatsRepo,
				mockSrsService,
				logger,
			)

			// Call method
			card, err := service.GetNextCard(context.Background(), tc.userID)

			// Verify expectations
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else if tc.expectedErrMsg != "" {
				assert.EqualError(t, err, tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, card)
			}

			// Verify all expectations were met
			mockCardRepo.AssertExpectations(t)
			// No need to verify other mocks for GetNextCard as they aren't used
		})
	}
}
