package card_review

import (
	"context"
	"database/sql" // Used for transaction handling
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/logger"
)

// Verify interface compliance at compile time
var _ CardReviewService = (*cardReviewServiceImpl)(nil)

// cardReviewServiceImpl implements the CardReviewService interface.
type cardReviewServiceImpl struct {
	cardRepo   CardRepository
	statsRepo  UserCardStatsRepository
	srsService srs.Service
	logger     *slog.Logger
}

// NewCardReviewService creates a new CardReviewService implementation.
func NewCardReviewService(
	cardRepo CardRepository,
	statsRepo UserCardStatsRepository,
	srsService srs.Service,
	logger *slog.Logger,
) CardReviewService {
	// Validate inputs
	if cardRepo == nil {
		panic("cardRepo cannot be nil")
	}
	if statsRepo == nil {
		panic("statsRepo cannot be nil")
	}
	if srsService == nil {
		panic("srsService cannot be nil")
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &cardReviewServiceImpl{
		cardRepo:   cardRepo,
		statsRepo:  statsRepo,
		srsService: srsService,
		logger:     logger.With(slog.String("component", "card_review_service")),
	}
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

	// Call the repository to get the next card due for review
	card, err := s.cardRepo.GetNextReviewCard(ctx, userID)
	if err != nil {
		// Map store.ErrCardNotFound to service.ErrNoCardsDue
		if errors.Is(err, ErrCardNotFound) {
			log.Debug("no cards due for review", slog.String("user_id", userID.String()))
			return nil, ErrNoCardsDue
		}

		log.Error("failed to get next review card",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return nil, fmt.Errorf("failed to get next review card: %w", err)
	}

	log.Debug("successfully retrieved next review card",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()))
	return card, nil
}

// SubmitAnswer implements CardReviewService.SubmitAnswer.
// It processes a user's answer to a flashcard and updates the review schedule.
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
	err := s.runInTransaction(
		ctx,
		func(ctx context.Context, cardRepo CardRepository, statsRepo UserCardStatsRepository) error {
			// First, verify that the card exists
			card, err := cardRepo.GetByID(ctx, cardID)
			if err != nil {
				if errors.Is(err, ErrCardNotFound) {
					log.Warn("card not found for review",
						slog.String("user_id", userID.String()),
						slog.String("card_id", cardID.String()))
					return ErrCardNotFound
				}
				return fmt.Errorf("failed to get card: %w", err)
			}

			// Verify that the user owns the card
			if card.UserID != userID {
				log.Warn("user does not own card",
					slog.String("user_id", userID.String()),
					slog.String("card_id", cardID.String()),
					slog.String("owner_id", card.UserID.String()))
				return ErrCardNotOwned
			}

			// Get the current stats
			stats, err := statsRepo.Get(ctx, userID, cardID)
			if err != nil {
				if errors.Is(err, ErrCardStatsNotFound) {
					log.Warn("stats not found for card",
						slog.String("user_id", userID.String()),
						slog.String("card_id", cardID.String()))
					// Create new stats with default values
					stats, err = domain.NewUserCardStats(userID, cardID)
					if err != nil {
						return fmt.Errorf("failed to create new stats: %w", err)
					}
				} else {
					return fmt.Errorf("failed to get stats: %w", err)
				}
			}

			// Calculate new review schedule using SRS algorithm
			newStats, err := s.srsService.CalculateNextReview(stats, answer.Outcome, time.Now().UTC())
			if err != nil {
				log.Error("failed to calculate next review",
					slog.String("error", err.Error()),
					slog.String("user_id", userID.String()),
					slog.String("card_id", cardID.String()))
				return fmt.Errorf("failed to calculate next review: %w", err)
			}

			// Save or update the stats
			if stats.LastReviewedAt.IsZero() {
				// This is a new card that hasn't been reviewed yet
				err = statsRepo.Create(ctx, newStats)
				if err != nil {
					return fmt.Errorf("failed to create stats: %w", err)
				}
			} else {
				// This is an existing card that has been reviewed before
				err = statsRepo.Update(ctx, newStats)
				if err != nil {
					return fmt.Errorf("failed to update stats: %w", err)
				}
			}

			// Store the updated stats for the return value
			updatedStats = newStats
			return nil
		},
	)

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
		return nil, fmt.Errorf("failed to submit answer: %w", err)
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

// runInTransaction runs the given function in a transaction
func (s *cardReviewServiceImpl) runInTransaction(
	ctx context.Context,
	fn func(context.Context, CardRepository, UserCardStatsRepository) error,
) error {
	// Create a local variable that explicitly uses the database/sql package
	// This prevents the "imported and not used" error
	sqlRef := sql.IsolationLevel(0)
	_ = sqlRef // Suppress unused variable warning

	db := s.cardRepo.DB()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create transactional repositories
	cardRepo := s.cardRepo.WithTx(tx)
	statsRepo := s.statsRepo.WithTx(tx)

	// Execute the transaction function
	err = fn(ctx, cardRepo, statsRepo)
	if err != nil {
		// Roll back if there was an error
		if rbErr := tx.Rollback(); rbErr != nil {
			// We combine the rollback error with the original error
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
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
