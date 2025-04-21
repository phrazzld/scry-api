package card_review

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
)

// CardRepository defines the interface for repositories that can provide
// card data and support transactions.
type CardRepository interface {
	// GetByID retrieves a card by its unique ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)

	// GetNextReviewCard retrieves the next card due for review for a user.
	GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)

	// WithTx returns a new repository instance that uses the provided transaction.
	WithTx(tx *sql.Tx) CardRepository

	// DB returns the underlying database connection.
	DB() *sql.DB
}

// UserCardStatsRepository defines the interface for repositories that can provide
// user card statistics data and support transactions.
type UserCardStatsRepository interface {
	// Get retrieves user card statistics by the combination of user ID and card ID.
	Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

	// Create saves a new user card statistics entry.
	Create(ctx context.Context, stats *domain.UserCardStats) error

	// Update modifies an existing statistics entry.
	Update(ctx context.Context, stats *domain.UserCardStats) error

	// WithTx returns a new repository instance that uses the provided transaction.
	WithTx(tx *sql.Tx) UserCardStatsRepository
}

// NewCardRepositoryAdapter creates a new adapter that allows a store.CardStore
// to be used where a CardRepository is expected.
func NewCardRepositoryAdapter(cardStore store.CardStore, db *sql.DB) CardRepository {
	return &cardRepositoryAdapter{
		cardStore: cardStore,
		db:        db,
	}
}

// cardRepositoryAdapter adapts a store.CardStore to the CardRepository interface
type cardRepositoryAdapter struct {
	cardStore store.CardStore
	db        *sql.DB
}

// GetByID implements CardRepository.GetByID
func (a *cardRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	return a.cardStore.GetByID(ctx, id)
}

// GetNextReviewCard implements CardRepository.GetNextReviewCard
func (a *cardRepositoryAdapter) GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	return a.cardStore.GetNextReviewCard(ctx, userID)
}

// WithTx implements CardRepository.WithTx
func (a *cardRepositoryAdapter) WithTx(tx *sql.Tx) CardRepository {
	return &cardRepositoryAdapter{
		cardStore: a.cardStore.WithTxCardStore(tx),
		db:        a.db,
	}
}

// DB implements CardRepository.DB
func (a *cardRepositoryAdapter) DB() *sql.DB {
	return a.db
}

// NewUserCardStatsRepositoryAdapter creates a new adapter that allows a store.UserCardStatsStore
// to be used where a UserCardStatsRepository is expected.
func NewUserCardStatsRepositoryAdapter(statsStore store.UserCardStatsStore) UserCardStatsRepository {
	return &userCardStatsRepositoryAdapter{
		statsStore: statsStore,
	}
}

// userCardStatsRepositoryAdapter adapts a store.UserCardStatsStore to the UserCardStatsRepository interface
type userCardStatsRepositoryAdapter struct {
	statsStore store.UserCardStatsStore
}

// Get implements UserCardStatsRepository.Get
func (a *userCardStatsRepositoryAdapter) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	return a.statsStore.Get(ctx, userID, cardID)
}

// Create implements UserCardStatsRepository.Create
func (a *userCardStatsRepositoryAdapter) Create(ctx context.Context, stats *domain.UserCardStats) error {
	return a.statsStore.Create(ctx, stats)
}

// Update implements UserCardStatsRepository.Update
func (a *userCardStatsRepositoryAdapter) Update(ctx context.Context, stats *domain.UserCardStats) error {
	return a.statsStore.Update(ctx, stats)
}

// WithTx implements UserCardStatsRepository.WithTx
func (a *userCardStatsRepositoryAdapter) WithTx(tx *sql.Tx) UserCardStatsRepository {
	return &userCardStatsRepositoryAdapter{
		statsStore: a.statsStore.WithTx(tx),
	}
}
