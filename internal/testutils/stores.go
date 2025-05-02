//go:build !compatibility && ignore_redeclarations

// This file provides test utilities for creating and managing stores.
// It should be used in preference to the compatibility.go file where possible.

package testutils

import (
	"database/sql"
	"log/slog"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
)

// TestStores holds all store implementations for testing.
// This allows tests to easily access all stores sharing a single transaction.
type TestStores struct {
	UserStore         store.UserStore
	MemoStore         store.MemoStore
	CardStore         store.CardStore
	UserCardStatStore store.UserCardStatsStore
	TaskStore         task.TaskStore
}

// CreateTestStores creates all store implementations using a shared transaction.
// This ensures all operations within the test use the same transaction,
// maintaining isolation and supporting automatic rollback.
//
// Usage:
//
//	testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
//	    stores := testutils.CreateTestStores(tx)
//	    // Use stores.UserStore, stores.MemoStore, etc. in your tests
//	})
func CreateTestStores(tx store.DBTX) TestStores {
	// Create a logger that discards output for tests
	logger := slog.Default()

	// BCrypt cost set to 4 for faster tests
	bcryptCost := 4

	return TestStores{
		UserStore:         postgres.NewPostgresUserStore(tx, bcryptCost),
		MemoStore:         postgres.NewPostgresMemoStore(tx, logger),
		CardStore:         postgres.NewPostgresCardStore(tx, logger),
		UserCardStatStore: postgres.NewPostgresUserCardStatsStore(tx, logger),
		TaskStore:         postgres.NewPostgresTaskStore(tx),
	}
}

// CreateCardRepositoryAdapter creates a card repository adapter for testing
func CreateCardRepositoryAdapter(cardStore store.CardStore, db store.DBTX) service.CardRepository {
	sqlDB, ok := db.(*sql.DB)
	if !ok {
		// If it's not a *sql.DB, it must be our TxDB wrapper with a transaction
		// For testing, just pass nil as the DB since we're using transactions
		return service.NewCardRepositoryAdapter(cardStore, nil)
	}
	return service.NewCardRepositoryAdapter(cardStore, sqlDB)
}

// CreateStatsRepositoryAdapter creates a stats repository adapter for testing
func CreateStatsRepositoryAdapter(statsStore store.UserCardStatsStore) service.StatsRepository {
	return service.NewStatsRepositoryAdapter(statsStore)
}

// CreateTestJWTService creates a JWT service with test settings
// It uses a predefined secret suitable for testing purposes.
// This is not secure for production use.
func CreateTestJWTService() (auth.JWTService, error) {
	// Standard test JWT settings
	// This uses a fixed secret that is suitable for testing but should never be used in production
	authConfig := config.AuthConfig{
		JWTSecret:                   "test-jwt-secret-thatis32characterslong",
		TokenLifetimeMinutes:        60,   // 60 minutes
		RefreshTokenLifetimeMinutes: 1440, // 24 hours
	}

	return auth.NewJWTService(authConfig)
}
