package card_review

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
)

// Verify interface compliance at compile time
var _ CardReviewService = (*cardReviewServiceImpl)(nil)

// cardReviewServiceImpl implements the CardReviewService interface.
type cardReviewServiceImpl struct {
	cardStore  store.CardStore
	statsStore store.UserCardStatsStore
	srsService srs.Service
	logger     *slog.Logger
}

// NewCardReviewService creates a new CardReviewService implementation.
// It returns an error if any of the required dependencies are nil.
func NewCardReviewService(
	cardStore store.CardStore,
	statsStore store.UserCardStatsStore,
	srsService srs.Service,
	logger *slog.Logger,
) (CardReviewService, error) {
	// Validate inputs
	if cardStore == nil {
		return nil, domain.NewValidationError("cardStore", "cannot be nil", domain.ErrValidation)
	}
	if statsStore == nil {
		return nil, domain.NewValidationError("statsStore", "cannot be nil", domain.ErrValidation)
	}
	if srsService == nil {
		return nil, domain.NewValidationError("srsService", "cannot be nil", domain.ErrValidation)
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &cardReviewServiceImpl{
		cardStore:  cardStore,
		statsStore: statsStore,
		srsService: srsService,
		logger:     logger.With(slog.String("component", "card_review_service")),
	}, nil
}

// GetNextCard implements CardReviewService.GetNextCard.
// It retrieves the next card due for review for a user.
func (s *cardReviewServiceImpl) GetNextCard(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.Card, error) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("retrieving next review card", slog.String("user_id", userID.String()))

	// Call the store to get the next card due for review
	card, err := s.cardStore.GetNextReviewCard(ctx, userID)
	if err != nil {
		// Map "card not found" errors to service.ErrNoCardsDue
		if errors.Is(err, store.ErrCardNotFound) || errors.Is(err, store.ErrNotFound) {
			log.Debug("no cards due for review", slog.String("user_id", userID.String()))
			return nil, ErrNoCardsDue
		}

		log.Error("failed to get next review card",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return nil, NewGetNextCardError("database error", err)
	}

	log.Debug("successfully retrieved next review card",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()))
	return card, nil
}

// SubmitAnswer implements CardReviewService.SubmitAnswer.
// It processes a user's answer to a flashcard and updates the review schedule.
//
// CONCURRENCY PROTECTION:
// This method uses SELECT FOR UPDATE to acquire a row-level lock on the user's stats
// for the given card. This prevents race conditions that could occur if multiple
// requests try to update the same stats record simultaneously. The lock is acquired
// within a transaction and is held until the transaction is committed or rolled back.
func (s *cardReviewServiceImpl) SubmitAnswer(
	ctx context.Context,
	userID uuid.UUID,
	cardID uuid.UUID,
	answer ReviewAnswer,
) (*domain.UserCardStats, error) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("processing review answer",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.String("outcome", string(answer.Outcome)))

	// Validate answer outcome
	if !isValidOutcome(answer.Outcome) {
		log.Warn("invalid review outcome",
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()),
			slog.String("outcome", string(answer.Outcome)))
		return nil, ErrInvalidAnswer
	}

	// We need to run these operations in a single transaction
	var updatedStats *domain.UserCardStats

	// Use the standard store.RunInTransaction helper for consistent transaction handling
	err := store.RunInTransaction(ctx, s.cardStore.DB(), func(ctx context.Context, tx *sql.Tx) error {
		// Get transactional stores
		txCardStore := s.cardStore.WithTx(tx)
		txStatsStore := s.statsStore.WithTx(tx)

		// First, verify that the card exists
		card, err := txCardStore.GetByID(ctx, cardID)
		if err != nil {
			if errors.Is(err, store.ErrCardNotFound) {
				log.Warn("card not found for review",
					slog.String("user_id", userID.String()),
					slog.String("card_id", cardID.String()))
				return ErrCardNotFound
			}
			return NewSubmitAnswerError("failed to retrieve card", err)
		}

		// Verify that the user owns the card
		if card.UserID != userID {
			log.Warn("user does not own card",
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()),
				slog.String("owner_id", card.UserID.String()))
			return ErrCardNotOwned
		}

		// Get the current stats with a row-level lock to prevent concurrent updates
		stats, err := txStatsStore.GetForUpdate(ctx, userID, cardID)
		if err != nil {
			if errors.Is(err, store.ErrUserCardStatsNotFound) {
				log.Warn("stats not found for card",
					slog.String("user_id", userID.String()),
					slog.String("card_id", cardID.String()))
				// Create new stats with default values
				stats, err = domain.NewUserCardStats(userID, cardID)
				if err != nil {
					return NewSubmitAnswerError("failed to create new stats", err)
				}
			} else {
				return NewSubmitAnswerError("failed to retrieve stats", err)
			}
		}

		// Calculate new review schedule using SRS algorithm
		newStats, err := s.srsService.CalculateNextReview(stats, answer.Outcome, time.Now().UTC())
		if err != nil {
			log.Error("failed to calculate next review",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
			return NewSubmitAnswerError("failed to calculate next review", err)
		}

		// Save or update the stats
		if stats.LastReviewedAt.IsZero() {
			// This is a new card that hasn't been reviewed yet
			err = txStatsStore.Create(ctx, newStats)
			if err != nil {
				return NewSubmitAnswerError("failed to create stats record", err)
			}
		} else {
			// This is an existing card that has been reviewed before
			err = txStatsStore.Update(ctx, newStats)
			if err != nil {
				return NewSubmitAnswerError("failed to update stats record", err)
			}
		}

		// Store the updated stats for the return value
		updatedStats = newStats
		return nil
	})

	if err != nil {
		// If the error is already one of our service errors, pass it through
		if errors.Is(err, ErrCardNotFound) ||
			errors.Is(err, ErrCardNotOwned) ||
			errors.Is(err, ErrInvalidAnswer) {
			return nil, err
		}

		log.Error("failed to submit answer",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		return nil, err // No need to wrap, we're using CustomErrors now
	}

	log.Debug("successfully processed review answer",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.String("outcome", string(answer.Outcome)),
		slog.Float64("ease_factor", updatedStats.EaseFactor),
		slog.Int("interval", updatedStats.Interval),
		slog.Time("next_review_at", updatedStats.NextReviewAt))

	return updatedStats, nil
}

// isValidOutcome checks if the given outcome is valid
func isValidOutcome(outcome domain.ReviewOutcome) bool {
	switch outcome {
	case domain.ReviewOutcomeAgain,
		domain.ReviewOutcomeHard,
		domain.ReviewOutcomeGood,
		domain.ReviewOutcomeEasy:
		return true
	default:
		return false
	}
}
