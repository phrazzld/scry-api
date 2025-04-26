package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockFailingCardRepository is a specialized mock that can be configured to fail at specific points
type MockFailingCardRepository struct {
	mock.Mock
	CardStore         store.CardStore
	FailOnCreateCards bool
	dbConn            *sql.DB // Renamed to avoid naming conflict with DB method
}

func (m *MockFailingCardRepository) CreateMultiple(
	ctx context.Context,
	cards []*domain.Card,
) error {
	if m.FailOnCreateCards {
		return errors.New("simulated card creation failure")
	}
	return m.CardStore.CreateMultiple(ctx, cards)
}

func (m *MockFailingCardRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Card, error) {
	return m.CardStore.GetByID(ctx, id)
}

func (m *MockFailingCardRepository) UpdateContent(
	ctx context.Context,
	id uuid.UUID,
	content json.RawMessage,
) error {
	return m.CardStore.UpdateContent(ctx, id, content)
}

func (m *MockFailingCardRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return m.CardStore.Delete(ctx, id)
}

func (m *MockFailingCardRepository) WithTx(tx *sql.Tx) service.CardRepository {
	return &MockFailingCardRepository{
		CardStore:         m.CardStore.WithTx(tx),
		FailOnCreateCards: m.FailOnCreateCards,
		dbConn:            m.dbConn,
	}
}

func (m *MockFailingCardRepository) DB() *sql.DB {
	return m.dbConn
}

// MockFailingStatsRepository is a specialized mock that can be configured to fail at specific points
type MockFailingStatsRepository struct {
	mock.Mock
	StatsStore   store.UserCardStatsStore
	FailOnCreate bool
}

func (m *MockFailingStatsRepository) Create(
	ctx context.Context,
	stats *domain.UserCardStats,
) error {
	if m.FailOnCreate {
		return errors.New("simulated stats creation failure")
	}
	// There's no Create method in the UserCardStatsStore interface, so we'd need to implement it
	// For testing, we'll simulate the Create operation using a direct SQL query
	query := `
		INSERT INTO user_card_stats (user_id, card_id, interval, ease_factor, consecutive_correct,
								   last_reviewed_at, next_review_at, review_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var lastReviewedAt interface{}
	if stats.LastReviewedAt.IsZero() {
		lastReviewedAt = nil
	} else {
		lastReviewedAt = stats.LastReviewedAt
	}

	db := m.StatsStore.(interface{ QueryExecContext() store.DBTX }).QueryExecContext()
	_, err := db.ExecContext(
		ctx,
		query,
		stats.UserID,
		stats.CardID,
		stats.Interval,
		stats.EaseFactor,
		stats.ConsecutiveCorrect,
		lastReviewedAt,
		stats.NextReviewAt,
		stats.ReviewCount,
		stats.CreatedAt,
		stats.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user card stats: %w", err)
	}

	return nil
}

func (m *MockFailingStatsRepository) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	return m.StatsStore.Get(ctx, userID, cardID)
}

func (m *MockFailingStatsRepository) Update(
	ctx context.Context,
	stats *domain.UserCardStats,
) error {
	return m.StatsStore.Update(ctx, stats)
}

func (m *MockFailingStatsRepository) WithTx(tx *sql.Tx) service.StatsRepository {
	return &MockFailingStatsRepository{
		StatsStore:   m.StatsStore.WithTx(tx),
		FailOnCreate: m.FailOnCreate,
	}
}

// No need for helper methods now that we have a proper Create method in the interface

// TestCardService_CreateCards_Atomicity tests that card and stats creation is atomic
func TestCardService_CreateCards_Atomicity(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		logger := slog.Default()

		// Create a user for testing
		userEmail := "card-tx-test@example.com"
		userID := testutils.MustInsertUser(ctx, t, tx, userEmail, bcrypt.MinCost)

		// Create a memo for testing
		memoID := uuid.New()
		memo := &domain.Memo{
			ID:        memoID,
			UserID:    userID,
			Text:      "Test memo for card creation",
			Status:    domain.MemoStatusCompleted,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		_, err := tx.ExecContext(
			ctx,
			"INSERT INTO memos (id, user_id, text, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
			memo.ID,
			memo.UserID,
			memo.Text,
			memo.Status,
			memo.CreatedAt,
			memo.UpdatedAt,
		)
		require.NoError(t, err, "Failed to create test memo")

		// Setup base stores with transaction
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Helper function to create test cards
		createTestCards := func(count int) []*domain.Card {
			cards := make([]*domain.Card, count)
			for i := 0; i < count; i++ {
				content := domain.CardContent{
					Front: fmt.Sprintf("Test card front %d", i),
					Back:  fmt.Sprintf("Test card back %d", i),
				}
				contentBytes, err := json.Marshal(content)
				require.NoError(t, err, "Failed to marshal card content")

				card, err := domain.NewCard(userID, memoID, contentBytes)
				require.NoError(t, err, "Failed to create test card")
				cards[i] = card
			}
			return cards
		}

		t.Run("Transaction_Rollback_On_Card_Failure", func(t *testing.T) {
			// Create a failing repository that fails on card creation
			failingCardRepo := &MockFailingCardRepository{
				CardStore:         cardStore,
				FailOnCreateCards: true, // Fail during card creation
				dbConn:            db,   // Need the real DB for transaction management
			}

			// Create a normal stats repository
			statsRepo := &MockFailingStatsRepository{
				StatsStore:   statsStore,
				FailOnCreate: false,
			}

			// Create service with the failing repository
			cardService, err := service.NewCardService(failingCardRepo, statsRepo, logger)
			require.NoError(t, err, "Failed to create card service")

			// Create test cards
			cards := createTestCards(2)

			// Attempt to create cards - this should fail
			createErr := cardService.CreateCards(ctx, cards)

			// Verify the operation failed
			assert.Error(t, createErr, "Operation should fail")
			assert.Contains(
				t,
				createErr.Error(),
				"simulated card creation failure",
				"Error should be from our mock",
			)

			// Verify no cards were created
			var cardCount int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM cards WHERE user_id = $1 AND memo_id = $2",
				userID, memoID,
			).Scan(&cardCount)
			require.NoError(t, err, "Failed to count cards")
			assert.Equal(
				t,
				0,
				cardCount,
				"No cards should exist in the database due to transaction rollback",
			)

			// Verify no stats were created
			var statsCount int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM user_card_stats WHERE user_id = $1",
				userID,
			).Scan(&statsCount)
			require.NoError(t, err, "Failed to count user card stats")
			assert.Equal(
				t,
				0,
				statsCount,
				"No stats should exist in the database due to transaction rollback",
			)
		})

		t.Run("Transaction_Rollback_On_Stats_Failure", func(t *testing.T) {
			// Create a normal card repository
			cardRepo := &MockFailingCardRepository{
				CardStore:         cardStore,
				FailOnCreateCards: false,
				dbConn:            db, // Need the real DB for transaction management
			}

			// Create a failing stats repository
			statsRepo := &MockFailingStatsRepository{
				StatsStore:   statsStore,
				FailOnCreate: true, // Fail during stats creation
			}

			// Create service with the repositories
			cardService, err := service.NewCardService(cardRepo, statsRepo, logger)
			require.NoError(t, err, "Failed to create card service")

			// Create test cards
			cards := createTestCards(2)

			// Attempt to create cards - this should fail during stats creation
			createErr := cardService.CreateCards(ctx, cards)

			// Verify the operation failed
			assert.Error(t, createErr, "Operation should fail")
			assert.Contains(
				t,
				createErr.Error(),
				"simulated stats creation failure",
				"Error should be from our mock",
			)

			// Verify no cards were created due to transaction rollback
			var cardCount int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM cards WHERE user_id = $1 AND memo_id = $2",
				userID, memoID,
			).Scan(&cardCount)
			require.NoError(t, err, "Failed to count cards")
			assert.Equal(
				t,
				0,
				cardCount,
				"No cards should exist in the database due to transaction rollback",
			)

			// Verify no stats were created
			var statsCount int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM user_card_stats WHERE user_id = $1",
				userID,
			).Scan(&statsCount)
			require.NoError(t, err, "Failed to count user card stats")
			assert.Equal(
				t,
				0,
				statsCount,
				"No stats should exist in the database due to transaction rollback",
			)
		})

		t.Run("Transaction_Commit_On_Success", func(t *testing.T) {
			// Fix the failing repositories to create a successful flow
			cardRepo := &MockFailingCardRepository{
				CardStore:         cardStore,
				FailOnCreateCards: false,
				dbConn:            db,
			}

			// We need a different approach for the stats repository since we're missing Create method
			// Instead of mocking, let's use a direct implementation
			adapter := &statsRepositoryAdapter{
				statsStore: statsStore,
			}

			// Create service with the successful repositories
			cardService, err := service.NewCardService(cardRepo, adapter, logger)
			require.NoError(t, err, "Failed to create card service")

			// Create test cards
			cards := createTestCards(2)

			// Attempt to create cards - this should succeed
			createErr := cardService.CreateCards(ctx, cards)

			// Verify the operation succeeded
			assert.NoError(t, createErr, "Operation should succeed")

			// Verify cards were created
			var cardCount int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM cards WHERE user_id = $1 AND memo_id = $2",
				userID, memoID,
			).Scan(&cardCount)
			require.NoError(t, err, "Failed to count cards")
			assert.Equal(t, len(cards), cardCount, "Cards should exist in the database")

			// Verify stats were created
			var statsCount int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM user_card_stats WHERE user_id = $1",
				userID,
			).Scan(&statsCount)
			require.NoError(t, err, "Failed to count user card stats")
			assert.Equal(t, len(cards), statsCount, "Stats should exist in the database")

			// Check the stats are properly associated with the cards
			for _, card := range cards {
				var count int
				err = tx.QueryRowContext(ctx,
					"SELECT COUNT(*) FROM user_card_stats WHERE user_id = $1 AND card_id = $2",
					userID, card.ID,
				).Scan(&count)
				require.NoError(t, err, "Failed to verify stats for card")
				assert.Equal(t, 1, count, "Each card should have one stats entry")
			}
		})
	})
}

// statsRepositoryAdapter adapts UserCardStatsStore to StatsRepository
type statsRepositoryAdapter struct {
	statsStore store.UserCardStatsStore
}

// Create implements service.StatsRepository.Create
func (a *statsRepositoryAdapter) Create(ctx context.Context, stats *domain.UserCardStats) error {
	// Now we can use the Create method that was added to the interface
	return a.statsStore.Create(ctx, stats)
}

// Get implements service.StatsRepository
func (a *statsRepositoryAdapter) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	return a.statsStore.Get(ctx, userID, cardID)
}

// Update implements service.StatsRepository
func (a *statsRepositoryAdapter) Update(ctx context.Context, stats *domain.UserCardStats) error {
	return a.statsStore.Update(ctx, stats)
}

// WithTx implements service.StatsRepository
func (a *statsRepositoryAdapter) WithTx(tx *sql.Tx) service.StatsRepository {
	return &statsRepositoryAdapter{
		statsStore: a.statsStore.WithTx(tx),
	}
}
