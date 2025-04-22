package card_review_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCardStore is a mock implementation of the store.CardStore interface
type MockCardStore struct {
	mock.Mock
}

func (m *MockCardStore) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	args := m.Called(ctx, cards)
	return args.Error(0)
}

func (m *MockCardStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}

func (m *MockCardStore) UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error {
	args := m.Called(ctx, id, content)
	return args.Error(0)
}

func (m *MockCardStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCardStore) GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}

func (m *MockCardStore) WithTx(tx *sql.Tx) store.CardStore {
	args := m.Called(tx)
	return args.Get(0).(store.CardStore)
}

// WithTxCardStore is deprecated. Use WithTx instead.
func (m *MockCardStore) WithTxCardStore(tx *sql.Tx) store.CardStore {
	return m.WithTx(tx)
}

func (m *MockCardStore) DB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

// MockUserCardStatsStore is a mock implementation of the store.UserCardStatsStore interface
type MockUserCardStatsStore struct {
	mock.Mock
}

func (m *MockUserCardStatsStore) Create(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

func (m *MockUserCardStatsStore) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	args := m.Called(ctx, userID, cardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserCardStats), args.Error(1)
}

func (m *MockUserCardStatsStore) GetForUpdate(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	args := m.Called(ctx, userID, cardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserCardStats), args.Error(1)
}

func (m *MockUserCardStatsStore) Update(ctx context.Context, stats *domain.UserCardStats) error {
	args := m.Called(ctx, stats)
	return args.Error(0)
}

func (m *MockUserCardStatsStore) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	args := m.Called(ctx, userID, cardID)
	return args.Error(0)
}

func (m *MockUserCardStatsStore) WithTx(tx *sql.Tx) store.UserCardStatsStore {
	args := m.Called(tx)
	return args.Get(0).(store.UserCardStatsStore)
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

// TestNewCardReviewService tests the service constructor with various inputs
func TestNewCardReviewService(t *testing.T) {
	// Setup
	mockCardStore := new(MockCardStore)
	mockStatsStore := new(MockUserCardStatsStore)
	mockSrsService := new(MockSRSService)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Test cases
	t.Run("valid dependencies", func(t *testing.T) {
		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, logger)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("nil card store", func(t *testing.T) {
		service, err := card_review.NewCardReviewService(nil, mockStatsStore, mockSrsService, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "cardStore cannot be nil")
	})

	t.Run("nil stats store", func(t *testing.T) {
		service, err := card_review.NewCardReviewService(mockCardStore, nil, mockSrsService, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "statsStore cannot be nil")
	})

	t.Run("nil SRS service", func(t *testing.T) {
		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, nil, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "srsService cannot be nil")
	})

	t.Run("nil logger", func(t *testing.T) {
		service, err := card_review.NewCardReviewService(mockCardStore, mockStatsStore, mockSrsService, nil)
		assert.NoError(t, err)
		assert.NotNil(t, service)
		// Logger should be defaulted, not error
	})
}

// TestGetNextCard tests the GetNextCard method of CardReviewService
func TestGetNextCard(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name           string
		userID         uuid.UUID
		setupMock      func(*MockCardStore, uuid.UUID)
		expectedError  error
		expectedErrMsg string
	}{
		{
			name:   "happy path - card found",
			userID: uuid.New(),
			setupMock: func(store *MockCardStore, userID uuid.UUID) {
				card := createTestCard(userID)
				store.On("GetNextReviewCard", mock.Anything, userID).Return(card, nil)
			},
			expectedError:  nil,
			expectedErrMsg: "",
		},
		{
			name:   "no cards due",
			userID: uuid.New(),
			setupMock: func(store *MockCardStore, userID uuid.UUID) {
				store.On("GetNextReviewCard", mock.Anything, userID).Return(nil, fmt.Errorf("card not found"))
			},
			expectedError:  card_review.ErrNoCardsDue,
			expectedErrMsg: "",
		},
		{
			name:   "repository error",
			userID: uuid.New(),
			setupMock: func(store *MockCardStore, userID uuid.UUID) {
				store.On("GetNextReviewCard", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			expectedError:  nil,
			expectedErrMsg: "failed to get next review card: database error",
		},
		{
			name:   "nil uuid",
			userID: uuid.Nil,
			setupMock: func(store *MockCardStore, userID uuid.UUID) {
				store.On("GetNextReviewCard", mock.Anything, userID).Return(nil, fmt.Errorf("card not found"))
			},
			expectedError:  card_review.ErrNoCardsDue,
			expectedErrMsg: "",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock stores
			mockCardStore := new(MockCardStore)
			mockStatsStore := new(MockUserCardStatsStore)
			mockSrsService := new(MockSRSService)
			tc.setupMock(mockCardStore, tc.userID)

			// Create no-op logger for testing
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Create service with all required dependencies
			service, err := card_review.NewCardReviewService(
				mockCardStore,
				mockStatsStore,
				mockSrsService,
				logger,
			)
			assert.NoError(t, err)
			assert.NotNil(t, service)

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
			mockCardStore.AssertExpectations(t)
			// No need to verify other mocks for GetNextCard as they aren't used
		})
	}
}

// TestSubmitAnswer tests the SubmitAnswer method of CardReviewService
func TestSubmitAnswer(t *testing.T) {
	// Only test invalid answer case since we can't easily mock RunInTransaction
	// without a mocking library
	invalidAnswer := card_review.ReviewAnswer{Outcome: "invalid"}
	userID := uuid.New()
	cardID := uuid.New()

	// Create mocks
	mockCardStore := new(MockCardStore)
	mockStatsStore := new(MockUserCardStatsStore)
	mockSrsService := new(MockSRSService)

	// Create service
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service, err := card_review.NewCardReviewService(
		mockCardStore,
		mockStatsStore,
		mockSrsService,
		logger,
	)
	assert.NoError(t, err)

	// Test invalid answer case
	_, err = service.SubmitAnswer(context.Background(), userID, cardID, invalidAnswer)
	assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)
}
