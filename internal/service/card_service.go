package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
)

// CardRepository defines the repository interface for the service layer
type CardRepository interface {
	// CreateMultiple saves multiple cards to the store
	CreateMultiple(ctx context.Context, cards []*domain.Card) error

	// GetByID retrieves a card by its unique ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)

	// WithTx returns a new repository instance that uses the provided transaction
	// This is used for transactional operations
	WithTx(tx *sql.Tx) CardRepository

	// DB returns the underlying database connection
	DB() *sql.DB
}

// StatsRepository defines the repository interface for user card statistics
type StatsRepository interface {
	// Create saves a new UserCardStats to the store
	Create(ctx context.Context, stats *domain.UserCardStats) error

	// Get retrieves user card statistics by the combination of user ID and card ID
	Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

	// Update modifies an existing statistics entry
	Update(ctx context.Context, stats *domain.UserCardStats) error

	// WithTx returns a new repository instance that uses the provided transaction
	WithTx(tx *sql.Tx) StatsRepository
}

// CardService provides card-related operations
type CardService interface {
	// CreateCards creates multiple cards and their associated stats in a single transaction
	// This orchestrates both card and stats creation atomically
	CreateCards(ctx context.Context, cards []*domain.Card) error

	// GetCard retrieves a card by its ID
	GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)
}

// cardServiceImpl implements the CardService interface
type cardServiceImpl struct {
	cardRepo  CardRepository
	statsRepo StatsRepository
	logger    *slog.Logger
}

// NewCardService creates a new CardService
func NewCardService(
	cardRepo CardRepository,
	statsRepo StatsRepository,
	logger *slog.Logger,
) CardService {
	return &cardServiceImpl{
		cardRepo:  cardRepo,
		statsRepo: statsRepo,
		logger:    logger.With(slog.String("component", "card_service")),
	}
}

// CreateCards implements CardService.CreateCards
// It creates multiple cards and their associated stats in a single transaction
func (s *cardServiceImpl) CreateCards(ctx context.Context, cards []*domain.Card) error {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	if len(cards) == 0 {
		log.Debug("no cards to create")
		return nil
	}

	log.Debug("creating cards and stats in transaction",
		slog.Int("card_count", len(cards)))

	// Run all operations in a single transaction for atomicity
	return store.RunInTransaction(ctx, s.cardRepo.DB(), func(ctx context.Context, tx *sql.Tx) error {
		// Get transactional repositories
		txCardRepo := s.cardRepo.WithTx(tx)
		txStatsRepo := s.statsRepo.WithTx(tx)

		// 1. Create the cards within the transaction
		err := txCardRepo.CreateMultiple(ctx, cards)
		if err != nil {
			log.Error("failed to create cards in transaction",
				slog.String("error", err.Error()))
			return fmt.Errorf("failed to create cards: %w", err)
		}

		// 2. Create a UserCardStats entry for each card
		for _, card := range cards {
			// Create a new UserCardStats with default values
			stats, err := domain.NewUserCardStats(card.UserID, card.ID)
			if err != nil {
				log.Error("failed to create user card stats object",
					slog.String("error", err.Error()),
					slog.String("user_id", card.UserID.String()),
					slog.String("card_id", card.ID.String()))
				return fmt.Errorf("failed to create user card stats: %w", err)
			}

			// Save the stats
			err = txStatsRepo.Create(ctx, stats)
			if err != nil {
				log.Error("failed to save user card stats in transaction",
					slog.String("error", err.Error()),
					slog.String("user_id", card.UserID.String()),
					slog.String("card_id", card.ID.String()))
				return fmt.Errorf("failed to save user card stats: %w", err)
			}

			log.Debug("created user card stats",
				slog.String("user_id", card.UserID.String()),
				slog.String("card_id", card.ID.String()))
		}

		log.Info("successfully created cards and stats in transaction",
			slog.Int("card_count", len(cards)))
		return nil
	})
}

// GetCard implements CardService.GetCard
// It retrieves a card by its ID
func (s *cardServiceImpl) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("retrieving card", slog.String("card_id", cardID.String()))

	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		log.Error("failed to retrieve card",
			slog.String("error", err.Error()),
			slog.String("card_id", cardID.String()))
		return nil, fmt.Errorf("failed to retrieve card: %w", err)
	}

	log.Debug("retrieved card successfully",
		slog.String("card_id", cardID.String()),
		slog.String("user_id", card.UserID.String()),
		slog.String("memo_id", card.MemoID.String()))

	return card, nil
}
