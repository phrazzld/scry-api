package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/store"
)

// Test NewCardService constructor validation
func TestNewCardService(t *testing.T) {
	tests := []struct {
		name        string
		cardRepo    CardRepository
		statsRepo   StatsRepository
		srsService  srs.Service
		logger      *slog.Logger
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil cardRepo",
			cardRepo:    nil,
			statsRepo:   &mockStatsRepository{},
			srsService:  &mockSRSService{},
			logger:      slog.Default(),
			expectError: true,
			errorMsg:    "cardRepo",
		},
		{
			name:        "nil statsRepo",
			cardRepo:    &mockCardRepository{},
			statsRepo:   nil,
			srsService:  &mockSRSService{},
			logger:      slog.Default(),
			expectError: true,
			errorMsg:    "statsRepo",
		},
		{
			name:        "nil srsService",
			cardRepo:    &mockCardRepository{},
			statsRepo:   &mockStatsRepository{},
			srsService:  nil,
			logger:      slog.Default(),
			expectError: true,
			errorMsg:    "srsService",
		},
		{
			name:        "nil logger uses default",
			cardRepo:    &mockCardRepository{},
			statsRepo:   &mockStatsRepository{},
			srsService:  &mockSRSService{},
			logger:      nil,
			expectError: false,
		},
		{
			name:        "all dependencies provided",
			cardRepo:    &mockCardRepository{},
			statsRepo:   &mockStatsRepository{},
			srsService:  &mockSRSService{},
			logger:      slog.Default(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewCardService(tt.cardRepo, tt.statsRepo, tt.srsService, tt.logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

// Test CreateCards method
func TestCardService_CreateCards(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	memoID := uuid.New()

	tests := []struct {
		name           string
		cards          []*domain.Card
		cardRepoError  error
		statsRepoError error
		expectError    bool
		errorContains  string
	}{
		{
			name:        "empty cards slice",
			cards:       []*domain.Card{},
			expectError: false,
		},
		{
			name: "successful creation",
			cards: []*domain.Card{
				mustCreateCard(t, userID, memoID, `{"front":"test1","back":"back1"}`),
				mustCreateCard(t, userID, memoID, `{"front":"test2","back":"back2"}`),
			},
			expectError: false,
		},
		{
			name: "card creation fails",
			cards: []*domain.Card{
				mustCreateCard(t, userID, memoID, `{"front":"test","back":"back"}`),
			},
			cardRepoError: errors.New("database error"),
			expectError:   true,
			errorContains: "failed to save cards",
		},
		{
			name: "stats creation fails",
			cards: []*domain.Card{
				mustCreateCard(t, userID, memoID, `{"front":"test","back":"back"}`),
			},
			statsRepoError: errors.New("stats error"),
			expectError:    true,
			errorContains:  "failed to save stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockDB := &sql.DB{} // This will be nil but won't be called for empty cards
			cardRepo := &mockCardRepository{
				createMultipleError: tt.cardRepoError,
				dbReturn:            mockDB,
			}
			statsRepo := &mockStatsRepository{
				createError: tt.statsRepoError,
			}
			srsService := &mockSRSService{}
			logger := slog.Default()

			service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
			require.NoError(t, err)

			// Skip transaction tests if we would need real DB connection
			if len(tt.cards) > 0 {
				// For unit tests with transactions, we'll test the error cases differently
				// by mocking the transaction behavior directly in the repository
				t.Skip("Skipping transaction test - requires integration test environment")
			}

			// Execute
			err = service.CreateCards(ctx, tt.cards)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify method calls
			if len(tt.cards) > 0 && tt.cardRepoError == nil {
				assert.True(t, cardRepo.createMultipleCalled)
			}
		})
	}
}

// Test GetCard method
func TestCardService_GetCard(t *testing.T) {
	ctx := context.Background()
	cardID := uuid.New()
	userID := uuid.New()
	memoID := uuid.New()

	tests := []struct {
		name          string
		cardID        uuid.UUID
		repoError     error
		repoCard      *domain.Card
		expectError   bool
		errorContains string
	}{
		{
			name:   "successful retrieval",
			cardID: cardID,
			repoCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			expectError: false,
		},
		{
			name:          "card not found",
			cardID:        cardID,
			repoError:     store.ErrCardNotFound,
			expectError:   true,
			errorContains: "card not found",
		},
		{
			name:          "database error",
			cardID:        cardID,
			repoError:     errors.New("database connection failed"),
			expectError:   true,
			errorContains: "failed to retrieve card",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			cardRepo := &mockCardRepository{
				getByIDError: tt.repoError,
				getByIDCard:  tt.repoCard,
			}
			statsRepo := &mockStatsRepository{}
			srsService := &mockSRSService{}
			logger := slog.Default()

			service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
			require.NoError(t, err)

			// Execute
			card, err := service.GetCard(ctx, tt.cardID)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, card)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				var cardSvcErr *CardServiceError
				assert.True(t, errors.As(err, &cardSvcErr))
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, card)
				assert.Equal(t, tt.repoCard, card)
			}

			// Verify method calls
			assert.True(t, cardRepo.getByIDCalled)
		})
	}
}

// Test UpdateCardContent method
func TestCardService_UpdateCardContent(t *testing.T) {
	ctx := context.Background()
	cardID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()
	memoID := uuid.New()
	content := json.RawMessage(`{"front":"updated","back":"content"}`)

	tests := []struct {
		name          string
		userID        uuid.UUID
		cardID        uuid.UUID
		content       json.RawMessage
		getError      error
		getCard       *domain.Card
		updateError   error
		expectError   bool
		errorContains string
		expectedErr   error
	}{
		{
			name:    "successful update",
			userID:  userID,
			cardID:  cardID,
			content: content,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			expectError: false,
		},
		{
			name:          "card not found",
			userID:        userID,
			cardID:        cardID,
			content:       content,
			getError:      store.ErrCardNotFound,
			expectError:   true,
			errorContains: "card not found",
		},
		{
			name:          "database error on get",
			userID:        userID,
			cardID:        cardID,
			content:       content,
			getError:      errors.New("database error"),
			expectError:   true,
			errorContains: "failed to retrieve card",
		},
		{
			name:    "not card owner",
			userID:  otherUserID,
			cardID:  cardID,
			content: content,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID, // Different from requester
				MemoID: memoID,
			},
			expectError:   true,
			errorContains: "owned by another user",
			expectedErr:   ErrNotOwned,
		},
		{
			name:    "update fails",
			userID:  userID,
			cardID:  cardID,
			content: content,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			updateError:   errors.New("update failed"),
			expectError:   true,
			errorContains: "failed to update card content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			cardRepo := &mockCardRepository{
				getByIDError:       tt.getError,
				getByIDCard:        tt.getCard,
				updateContentError: tt.updateError,
			}
			statsRepo := &mockStatsRepository{}
			srsService := &mockSRSService{}
			logger := slog.Default()

			service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
			require.NoError(t, err)

			// Execute
			err = service.UpdateCardContent(ctx, tt.userID, tt.cardID, tt.content)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr))
				}

				var cardSvcErr *CardServiceError
				assert.True(t, errors.As(err, &cardSvcErr))
			} else {
				assert.NoError(t, err)
				assert.True(t, cardRepo.updateContentCalled)
			}

			// Verify get was called
			assert.True(t, cardRepo.getByIDCalled)
		})
	}
}

// Test DeleteCard method
func TestCardService_DeleteCard(t *testing.T) {
	ctx := context.Background()
	cardID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()
	memoID := uuid.New()

	tests := []struct {
		name          string
		userID        uuid.UUID
		cardID        uuid.UUID
		getError      error
		getCard       *domain.Card
		deleteError   error
		expectError   bool
		errorContains string
		expectedErr   error
	}{
		{
			name:   "successful deletion",
			userID: userID,
			cardID: cardID,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			expectError: false,
		},
		{
			name:          "card not found",
			userID:        userID,
			cardID:        cardID,
			getError:      store.ErrCardNotFound,
			expectError:   true,
			errorContains: "card not found",
		},
		{
			name:          "database error on get",
			userID:        userID,
			cardID:        cardID,
			getError:      errors.New("database error"),
			expectError:   true,
			errorContains: "failed to retrieve card",
		},
		{
			name:   "not card owner",
			userID: otherUserID,
			cardID: cardID,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID, // Different from requester
				MemoID: memoID,
			},
			expectError:   true,
			errorContains: "owned by another user",
			expectedErr:   ErrNotOwned,
		},
		{
			name:   "delete fails",
			userID: userID,
			cardID: cardID,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			deleteError:   errors.New("delete failed"),
			expectError:   true,
			errorContains: "failed to delete card",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			cardRepo := &mockCardRepository{
				getByIDError: tt.getError,
				getByIDCard:  tt.getCard,
				deleteError:  tt.deleteError,
			}
			statsRepo := &mockStatsRepository{}
			srsService := &mockSRSService{}
			logger := slog.Default()

			service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
			require.NoError(t, err)

			// Execute
			err = service.DeleteCard(ctx, tt.userID, tt.cardID)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr))
				}

				var cardSvcErr *CardServiceError
				assert.True(t, errors.As(err, &cardSvcErr))
			} else {
				assert.NoError(t, err)
				assert.True(t, cardRepo.deleteCalled)
			}

			// Verify get was called
			assert.True(t, cardRepo.getByIDCalled)
		})
	}
}

// Test PostponeCard method
func TestCardService_PostponeCard(t *testing.T) {
	ctx := context.Background()
	cardID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()
	memoID := uuid.New()
	days := 7

	stats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 0,
		NextReviewAt:       time.Now().UTC().Add(24 * time.Hour),
		ReviewCount:        0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	tests := []struct {
		name             string
		userID           uuid.UUID
		cardID           uuid.UUID
		days             int
		getCardError     error
		getCard          *domain.Card
		getStatsError    error
		getStats         *domain.UserCardStats
		srsError         error
		updateStatsError error
		expectError      bool
		errorContains    string
		expectedErr      error
	}{
		{
			name:   "successful postpone",
			userID: userID,
			cardID: cardID,
			days:   days,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			getStats:    stats,
			expectError: false,
		},
		{
			name:          "invalid days",
			userID:        userID,
			cardID:        cardID,
			days:          0,
			expectError:   true,
			errorContains: "days must be at least 1",
			expectedErr:   srs.ErrInvalidDays,
		},
		{
			name:          "card not found",
			userID:        userID,
			cardID:        cardID,
			days:          days,
			getCardError:  store.ErrCardNotFound,
			expectError:   true,
			errorContains: "card not found",
		},
		{
			name:   "not card owner",
			userID: otherUserID,
			cardID: cardID,
			days:   days,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID, // Different from requester
				MemoID: memoID,
			},
			expectError:   true,
			errorContains: "owned by another user",
			expectedErr:   ErrNotOwned,
		},
		{
			name:   "stats not found",
			userID: userID,
			cardID: cardID,
			days:   days,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			getStatsError: store.ErrUserCardStatsNotFound,
			expectError:   true,
			errorContains: "user card statistics not found",
			expectedErr:   ErrStatsNotFound,
		},
		{
			name:   "srs calculation fails",
			userID: userID,
			cardID: cardID,
			days:   days,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			getStats:      stats,
			srsError:      errors.New("srs error"),
			expectError:   true,
			errorContains: "failed to calculate postponed review",
		},
		{
			name:   "update stats fails",
			userID: userID,
			cardID: cardID,
			days:   days,
			getCard: &domain.Card{
				ID:     cardID,
				UserID: userID,
				MemoID: memoID,
			},
			getStats:         stats,
			updateStatsError: errors.New("update error"),
			expectError:      true,
			errorContains:    "failed to update stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			cardRepo := &mockCardRepository{
				getByIDError: tt.getCardError,
				getByIDCard:  tt.getCard,
			}
			statsRepo := &mockStatsRepository{
				getForUpdateError: tt.getStatsError,
				getForUpdateStats: tt.getStats,
				updateError:       tt.updateStatsError,
			}
			srsService := &mockSRSService{
				postponeError: tt.srsError,
			}
			logger := slog.Default()

			service, err := NewCardService(cardRepo, statsRepo, srsService, logger)
			require.NoError(t, err)

			// Skip transaction-based tests that need real DB
			// Also skip tests that would make it past the card validation
			if !tt.expectError || tt.getCard != nil {
				t.Skip("Skipping postpone test that requires transaction - requires integration test environment")
			}

			// Execute
			result, err := service.PostponeCard(ctx, tt.userID, tt.cardID, tt.days)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				if tt.expectedErr != nil {
					assert.True(t, errors.Is(err, tt.expectedErr))
				}

				var cardSvcErr *CardServiceError
				assert.True(t, errors.As(err, &cardSvcErr))
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// Test CardServiceError methods
func TestCardServiceError(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		tests := []struct {
			name      string
			operation string
			message   string
			err       error
			expected  string
		}{
			{
				name:      "with underlying error",
				operation: "create",
				message:   "validation failed",
				err:       errors.New("invalid input"),
				expected:  "card service create failed: validation failed: invalid input",
			},
			{
				name:      "without underlying error",
				operation: "delete",
				message:   "not found",
				err:       nil,
				expected:  "card service delete failed: not found",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := &CardServiceError{
					Operation: tt.operation,
					Message:   tt.message,
					Err:       tt.err,
				}

				assert.Equal(t, tt.expected, err.Error())
			})
		}
	})

	t.Run("Unwrap method", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		err := &CardServiceError{
			Operation: "test",
			Message:   "test message",
			Err:       underlyingErr,
		}

		assert.Equal(t, underlyingErr, err.Unwrap())

		// Test with nil error
		err.Err = nil
		assert.Nil(t, err.Unwrap())
	})
}

// Test NewCardServiceError constructor
func TestNewCardServiceError(t *testing.T) {
	operation := "test_operation"
	message := "test message"
	underlyingErr := errors.New("underlying error")

	err := NewCardServiceError(operation, message, underlyingErr)

	assert.Equal(t, operation, err.Operation)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, underlyingErr, err.Err)
}

// Helper function to create test cards
func mustCreateCard(t *testing.T, userID, memoID uuid.UUID, content string) *domain.Card {
	card, err := domain.NewCard(userID, memoID, json.RawMessage(content))
	require.NoError(t, err)
	return card
}

// Mock implementations for testing

type mockCardRepository struct {
	// Method call tracking
	createMultipleCalled bool
	getByIDCalled        bool
	updateContentCalled  bool
	deleteCalled         bool
	withTxCalled         bool
	dbCalled             bool

	// Return values
	createMultipleError error
	getByIDError        error
	getByIDCard         *domain.Card
	updateContentError  error
	deleteError         error
	withTxReturn        CardRepository
	dbReturn            *sql.DB
}

func (m *mockCardRepository) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	m.createMultipleCalled = true
	return m.createMultipleError
}

func (m *mockCardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	m.getByIDCalled = true
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	return m.getByIDCard, nil
}

func (m *mockCardRepository) UpdateContent(ctx context.Context, id uuid.UUID, content json.RawMessage) error {
	m.updateContentCalled = true
	return m.updateContentError
}

func (m *mockCardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.deleteCalled = true
	return m.deleteError
}

func (m *mockCardRepository) WithTx(tx *sql.Tx) CardRepository {
	m.withTxCalled = true
	if m.withTxReturn != nil {
		return m.withTxReturn
	}
	return &mockCardRepository{}
}

func (m *mockCardRepository) DB() *sql.DB {
	m.dbCalled = true
	return m.dbReturn
}

type mockStatsRepository struct {
	// Method call tracking
	createCalled       bool
	getCalled          bool
	getForUpdateCalled bool
	updateCalled       bool
	withTxCalled       bool

	// Return values
	createError       error
	getError          error
	getStats          *domain.UserCardStats
	getForUpdateError error
	getForUpdateStats *domain.UserCardStats
	updateError       error
	withTxReturn      StatsRepository
}

func (m *mockStatsRepository) Create(ctx context.Context, stats *domain.UserCardStats) error {
	m.createCalled = true
	return m.createError
}

func (m *mockStatsRepository) Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error) {
	m.getCalled = true
	if m.getError != nil {
		return nil, m.getError
	}
	return m.getStats, nil
}

func (m *mockStatsRepository) GetForUpdate(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	m.getForUpdateCalled = true
	if m.getForUpdateError != nil {
		return nil, m.getForUpdateError
	}
	return m.getForUpdateStats, nil
}

func (m *mockStatsRepository) Update(ctx context.Context, stats *domain.UserCardStats) error {
	m.updateCalled = true
	return m.updateError
}

func (m *mockStatsRepository) WithTx(tx *sql.Tx) StatsRepository {
	m.withTxCalled = true
	if m.withTxReturn != nil {
		return m.withTxReturn
	}
	return &mockStatsRepository{}
}

type mockSRSService struct {
	// Return values
	postponeError error
}

func (m *mockSRSService) CalculateNextReview(
	stats *domain.UserCardStats,
	outcome domain.ReviewOutcome,
	now time.Time,
) (*domain.UserCardStats, error) {
	// Simple implementation for testing
	return stats, nil
}

func (m *mockSRSService) PostponeReview(
	stats *domain.UserCardStats,
	days int,
	now time.Time,
) (*domain.UserCardStats, error) {
	if m.postponeError != nil {
		return nil, m.postponeError
	}

	if days < 1 {
		return nil, srs.ErrInvalidDays
	}

	if stats == nil {
		return nil, srs.ErrNilStats
	}

	// Create updated stats
	newStats := &domain.UserCardStats{
		UserID:             stats.UserID,
		CardID:             stats.CardID,
		Interval:           stats.Interval,
		EaseFactor:         stats.EaseFactor,
		ConsecutiveCorrect: stats.ConsecutiveCorrect,
		LastReviewedAt:     stats.LastReviewedAt,
		NextReviewAt:       stats.NextReviewAt.AddDate(0, 0, days),
		ReviewCount:        stats.ReviewCount,
		CreatedAt:          stats.CreatedAt,
		UpdatedAt:          now,
	}

	return newStats, nil
}
