package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
)

// CardServiceError is a custom error type for card service errors.
type CardServiceError struct {
	Operation string
	Message   string
	Err       error
}

// Error implements the error interface for CardServiceError.
func (e *CardServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("card service %s failed: %s: %v", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("card service %s failed: %s", e.Operation, e.Message)
}

// Unwrap returns the wrapped error to support errors.Is/errors.As.
func (e *CardServiceError) Unwrap() error {
	return e.Err
}

// NewCardServiceError creates a new CardServiceError.
func NewCardServiceError(operation, message string, err error) *CardServiceError {
	return &CardServiceError{
		Operation: operation,
		Message:   message,
		Err:       err,
	}
}

// CardRepository defines the repository interface for the service layer
type CardRepository interface {
	// CreateMultiple saves multiple cards to the store
	CreateMultiple(ctx context.Context, cards []*domain.Card) error

	// GetByID retrieves a card by its unique ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)

	// UpdateContent modifies an existing card's content field
	UpdateContent(ctx context.Context, id uuid.UUID, content json.RawMessage) error

	// Delete removes a card from the store by its ID
	Delete(ctx context.Context, id uuid.UUID) error

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

	// GetForUpdate retrieves user card statistics with a row-level lock
	GetForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

	// Update modifies an existing statistics entry
	Update(ctx context.Context, stats *domain.UserCardStats) error

	// WithTx returns a new repository instance that uses the provided transaction
	WithTx(tx *sql.Tx) StatsRepository
}

// CardService provides card-related operations for managing flashcards
// in the spaced repetition system.
type CardService interface {
	// CreateCards creates multiple cards and their associated stats in a single transaction.
	// This orchestrates both card and stats creation atomically to ensure data consistency.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request correlation
	//   - cards: Slice of domain.Card objects to create
	//
	// Returns:
	//   - nil: If all cards and stats were created successfully
	//   - error: Any error that occurred during creation, with transaction rollback
	CreateCards(ctx context.Context, cards []*domain.Card) error

	// GetCard retrieves a card by its ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request correlation
	//   - cardID: UUID of the card to retrieve
	//
	// Returns:
	//   - (*domain.Card, nil): The card if found
	//   - (nil, error): CardServiceError wrapping store.ErrCardNotFound or other error
	GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)

	// UpdateCardContent modifies an existing card's content.
	// It validates that the user is the owner of the card before updating.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request correlation
	//   - userID: UUID of the user attempting to update the card
	//   - cardID: UUID of the card to update
	//   - content: New JSON content for the card
	//
	// Returns:
	//   - nil: If the update was successful
	//   - ErrNotOwned: If the userID doesn't match the card's owner
	//   - CardServiceError: Wrapping store.ErrCardNotFound or other error
	UpdateCardContent(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error

	// DeleteCard removes a card and its associated user_card_stats entries.
	// It validates that the user is the owner of the card before deleting.
	// The deletion relies on database CASCADE DELETE constraints for associated records.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request correlation
	//   - userID: UUID of the user attempting to delete the card
	//   - cardID: UUID of the card to delete
	//
	// Returns:
	//   - nil: If the deletion was successful
	//   - ErrNotOwned: If the userID doesn't match the card's owner
	//   - CardServiceError: Wrapping store.ErrCardNotFound or other error
	DeleteCard(ctx context.Context, userID, cardID uuid.UUID) error

	// PostponeCard extends the time until the next review for a card.
	// It validates that the user is the owner of the card before postponing
	// and performs the postpone operation atomically within a transaction.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request correlation
	//   - userID: UUID of the user attempting to postpone the card
	//   - cardID: UUID of the card to postpone
	//   - days: Number of days to postpone the review (must be >= 1)
	//
	// Returns:
	//   - (*domain.UserCardStats, nil): Updated stats with new NextReviewAt date
	//   - (nil, ErrNotOwned): If the userID doesn't match the card's owner
	//   - (nil, ErrInvalidDays): If days is less than 1
	//   - (nil, ErrStatsNotFound): If stats for the card couldn't be found
	//   - (nil, CardServiceError): Wrapping store.ErrCardNotFound or other error
	PostponeCard(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error)
}

// cardServiceImpl implements the CardService interface
type cardServiceImpl struct {
	cardRepo   CardRepository
	statsRepo  StatsRepository
	srsService srs.Service
	logger     *slog.Logger
}

// NewCardService creates a new CardService
// It returns an error if any of the required dependencies are nil.
func NewCardService(
	cardRepo CardRepository,
	statsRepo StatsRepository,
	srsService srs.Service,
	logger *slog.Logger,
) (CardService, error) {
	// Validate dependencies
	if cardRepo == nil {
		return nil, domain.NewValidationError("cardRepo", "cannot be nil", domain.ErrValidation)
	}
	if statsRepo == nil {
		return nil, domain.NewValidationError("statsRepo", "cannot be nil", domain.ErrValidation)
	}
	if srsService == nil {
		return nil, domain.NewValidationError("srsService", "cannot be nil", domain.ErrValidation)
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &cardServiceImpl{
		cardRepo:   cardRepo,
		statsRepo:  statsRepo,
		srsService: srsService,
		logger:     logger.With(slog.String("component", "card_service")),
	}, nil
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
	return store.RunInTransaction(
		ctx,
		s.cardRepo.DB(),
		func(ctx context.Context, tx *sql.Tx) error {
			// Get transactional repositories
			txCardRepo := s.cardRepo.WithTx(tx)
			txStatsRepo := s.statsRepo.WithTx(tx)

			// 1. Create the cards within the transaction
			err := txCardRepo.CreateMultiple(ctx, cards)
			if err != nil {
				log.Error("failed to create cards in transaction",
					slog.String("error", err.Error()))
				return NewCardServiceError("create_cards", "failed to save cards", err)
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
					return NewCardServiceError("create_cards", "failed to create stats object", err)
				}

				// Save the stats
				err = txStatsRepo.Create(ctx, stats)
				if err != nil {
					log.Error("failed to save user card stats in transaction",
						slog.String("error", err.Error()),
						slog.String("user_id", card.UserID.String()),
						slog.String("card_id", card.ID.String()))
					return NewCardServiceError("create_cards", "failed to save stats", err)
				}

				log.Debug("created user card stats",
					slog.String("user_id", card.UserID.String()),
					slog.String("card_id", card.ID.String()))
			}

			log.Info("successfully created cards and stats in transaction",
				slog.Int("card_count", len(cards)))
			return nil
		},
	)
}

// GetCard implements CardService.GetCard
// It retrieves a card by its ID
func (s *cardServiceImpl) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Make sure we have a valid logger
	if log == nil {
		// Use a default logger if needed
		log = slog.Default()
	}

	log.Debug("retrieving card", slog.String("card_id", cardID.String()))

	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		// Make sure we have a valid logger before logging errors
		if log != nil {
			log.Error("failed to retrieve card",
				slog.String("error", err.Error()),
				slog.String("card_id", cardID.String()))
		}

		// Check for specific error types
		if store.IsNotFoundError(err) {
			return nil, NewCardServiceError("get_card", "card not found", store.ErrCardNotFound)
		}

		return nil, NewCardServiceError("get_card", "failed to retrieve card", err)
	}

	log.Debug("retrieved card successfully",
		slog.String("card_id", cardID.String()),
		slog.String("user_id", card.UserID.String()),
		slog.String("memo_id", card.MemoID.String()))

	return card, nil
}

// UpdateCardContent implements CardService.UpdateCardContent
// It validates that the user is the owner of the card before updating its content
func (s *cardServiceImpl) UpdateCardContent(
	ctx context.Context,
	userID, cardID uuid.UUID,
	content json.RawMessage,
) error {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Make sure we have a valid logger
	if log == nil {
		// Use a default logger if needed
		log = slog.Default()
	}

	log.Debug("updating card content",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))

	// 1. Fetch the card to verify ownership
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		// Make sure we have a valid logger before logging errors
		if log != nil {
			log.Error("failed to retrieve card for content update",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
		}

		// Check for specific error types
		if store.IsNotFoundError(err) {
			return NewCardServiceError("update_card_content", "card not found", store.ErrCardNotFound)
		}

		return NewCardServiceError("update_card_content", "failed to retrieve card", err)
	}

	// 2. Validate ownership
	if card.UserID != userID {
		log.Error("unauthorized attempt to update card content",
			slog.String("requested_user_id", userID.String()),
			slog.String("actual_owner_id", card.UserID.String()),
			slog.String("card_id", cardID.String()))
		return NewCardServiceError("update_card_content", "card is owned by another user", ErrNotOwned)
	}

	// 3. Update the card content
	err = s.cardRepo.UpdateContent(ctx, cardID, content)
	if err != nil {
		log.Error("failed to update card content",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		return NewCardServiceError("update_card_content", "failed to update card content", err)
	}

	log.Debug("card content updated successfully",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))

	return nil
}

// DeleteCard implements CardService.DeleteCard
// It validates that the user is the owner of the card before deleting it
func (s *cardServiceImpl) DeleteCard(ctx context.Context, userID, cardID uuid.UUID) error {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Make sure we have a valid logger
	if log == nil {
		// Use a default logger if needed
		log = slog.Default()
	}

	log.Debug("deleting card",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))

	// 1. Fetch the card to verify ownership
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		// Make sure we have a valid logger before logging errors
		if log != nil {
			log.Error("failed to retrieve card for deletion",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
		}

		// Check for specific error types
		if store.IsNotFoundError(err) {
			return NewCardServiceError("delete_card", "card not found", store.ErrCardNotFound)
		}

		return NewCardServiceError("delete_card", "failed to retrieve card", err)
	}

	// 2. Validate ownership
	if card.UserID != userID {
		log.Error("unauthorized attempt to delete card",
			slog.String("requested_user_id", userID.String()),
			slog.String("actual_owner_id", card.UserID.String()),
			slog.String("card_id", cardID.String()))
		return NewCardServiceError("delete_card", "card is owned by another user", ErrNotOwned)
	}

	// 3. Delete the card
	err = s.cardRepo.Delete(ctx, cardID)
	if err != nil {
		log.Error("failed to delete card",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		return NewCardServiceError("delete_card", "failed to delete card", err)
	}

	log.Debug("card deleted successfully",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))

	return nil
}

// PostponeCard implements CardService.PostponeCard
// It validates that the user is the owner of the card before postponing the review
// and performs the operation in a transaction to ensure atomicity.
func (s *cardServiceImpl) PostponeCard(
	ctx context.Context,
	userID, cardID uuid.UUID,
	days int,
) (*domain.UserCardStats, error) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Make sure we have a valid logger
	if log == nil {
		// Use a default logger if needed
		log = slog.Default()
	}

	log.Debug("postponing card review",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.Int("days", days))

	// Validate days parameter first to fail fast
	if days < 1 {
		// Make sure we have a valid logger before logging errors
		if log != nil {
			log.Error("invalid days value for postpone",
				slog.Int("days", days),
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
		}
		return nil, NewCardServiceError("postpone_card", "days must be at least 1", srs.ErrInvalidDays)
	}

	// 1. Fetch the card to verify ownership
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		// Make sure we have a valid logger before logging errors
		if log != nil {
			log.Error("failed to retrieve card for postpone",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
		}

		// Check for specific error types
		if store.IsNotFoundError(err) {
			return nil, NewCardServiceError("postpone_card", "card not found", store.ErrCardNotFound)
		}

		return nil, NewCardServiceError("postpone_card", "failed to retrieve card", err)
	}

	// 2. Validate ownership
	if card.UserID != userID {
		log.Error("unauthorized attempt to postpone card review",
			slog.String("requested_user_id", userID.String()),
			slog.String("actual_owner_id", card.UserID.String()),
			slog.String("card_id", cardID.String()))
		return nil, NewCardServiceError("postpone_card", "card is owned by another user", ErrNotOwned)
	}

	// Initialize a variable to hold the updated stats
	var updatedStats *domain.UserCardStats

	// 3. Run the postpone operation in a transaction for atomicity
	err = store.RunInTransaction(
		ctx,
		s.cardRepo.DB(),
		func(ctx context.Context, tx *sql.Tx) error {
			// Get transactional repositories
			txStatsRepo := s.statsRepo.WithTx(tx)

			// 3.1 Get current stats with a row lock (FOR UPDATE)
			stats, err := txStatsRepo.GetForUpdate(ctx, userID, cardID)
			if err != nil {
				log.Error("failed to retrieve stats for update",
					slog.String("error", err.Error()),
					slog.String("user_id", userID.String()),
					slog.String("card_id", cardID.String()))

				// Check for specific error types
				if store.IsNotFoundError(err) {
					return NewCardServiceError("postpone_card", "user card statistics not found", ErrStatsNotFound)
				}

				return NewCardServiceError("postpone_card", "failed to retrieve stats", err)
			}

			// 3.2 Calculate new review date using SRS service
			now := time.Now().UTC()
			newStats, err := s.srsService.PostponeReview(stats, days, now)
			if err != nil {
				log.Error("failed to calculate postponed review",
					slog.String("error", err.Error()),
					slog.String("user_id", userID.String()),
					slog.String("card_id", cardID.String()),
					slog.Int("days", days))
				return NewCardServiceError("postpone_card", "failed to calculate postponed review", err)
			}

			// 3.3 Update stats in database
			err = txStatsRepo.Update(ctx, newStats)
			if err != nil {
				log.Error("failed to update stats with postponed review",
					slog.String("error", err.Error()),
					slog.String("user_id", userID.String()),
					slog.String("card_id", cardID.String()))
				return NewCardServiceError("postpone_card", "failed to update stats", err)
			}

			// Store the updated stats so we can return them after the transaction
			updatedStats = newStats
			return nil
		},
	)

	if err != nil {
		return nil, err // Error already wrapped and logged in transaction
	}

	log.Debug("card review successfully postponed",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.Int("days", days),
		slog.Time("next_review_at", updatedStats.NextReviewAt))

	return updatedStats, nil
}
