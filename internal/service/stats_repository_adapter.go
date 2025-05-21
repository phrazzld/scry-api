package service

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
)

// NewStatsRepositoryAdapter creates a new adapter that allows a store.UserCardStatsStore
// to be used where a StatsRepository is expected.
func NewStatsRepositoryAdapter(statsStore store.UserCardStatsStore) StatsRepository {
	return &statsRepositoryAdapter{
		statsStore: statsStore,
	}
}

// statsRepositoryAdapter adapts a store.UserCardStatsStore to the StatsRepository interface
type statsRepositoryAdapter struct {
	statsStore store.UserCardStatsStore
}

// Create implements StatsRepository.Create
func (a *statsRepositoryAdapter) Create(ctx context.Context, stats *domain.UserCardStats) error {
	return a.statsStore.Create(ctx, stats)
}

// Get implements StatsRepository.Get
func (a *statsRepositoryAdapter) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	return a.statsStore.Get(ctx, userID, cardID)
}

// GetForUpdate implements StatsRepository.GetForUpdate
func (a *statsRepositoryAdapter) GetForUpdate(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	return a.statsStore.GetForUpdate(ctx, userID, cardID)
}

// Update implements StatsRepository.Update
func (a *statsRepositoryAdapter) Update(ctx context.Context, stats *domain.UserCardStats) error {
	return a.statsStore.Update(ctx, stats)
}

// WithTx implements StatsRepository.WithTx
func (a *statsRepositoryAdapter) WithTx(tx *sql.Tx) StatsRepository {
	return &statsRepositoryAdapter{
		statsStore: a.statsStore.WithTx(tx),
	}
}
