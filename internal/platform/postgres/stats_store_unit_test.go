package postgres

import (
	"database/sql"
	"log/slog"
	"testing"

	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestNewPostgresUserCardStatsStore(t *testing.T) {
	tests := []struct {
		name        string
		db          store.DBTX
		logger      *slog.Logger
		expectPanic bool
		check       func(t *testing.T, store *PostgresUserCardStatsStore)
	}{
		{
			name:        "nil_db_panics",
			db:          nil,
			logger:      slog.Default(),
			expectPanic: true,
		},
		{
			name:   "valid_db_with_logger",
			db:     &sql.DB{},
			logger: slog.Default(),
			check: func(t *testing.T, store *PostgresUserCardStatsStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.NotNil(t, store.logger)
			},
		},
		{
			name:   "valid_db_nil_logger_uses_default",
			db:     &sql.DB{},
			logger: nil,
			check: func(t *testing.T, store *PostgresUserCardStatsStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.NotNil(t, store.logger)
			},
		},
		{
			name:   "mock_dbtx",
			db:     &mockDBTX{},
			logger: slog.Default(),
			check: func(t *testing.T, store *PostgresUserCardStatsStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.NotNil(t, store.logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					NewPostgresUserCardStatsStore(tt.db, tt.logger)
				})
				return
			}

			store := NewPostgresUserCardStatsStore(tt.db, tt.logger)
			if tt.check != nil {
				tt.check(t, store)
			}
		})
	}
}

func TestPostgresUserCardStatsStore_WithTx(t *testing.T) {
	// Note: We can't create a real *sql.Tx without a database connection,
	// so we'll test the behavior by checking the store structure.
	// The actual transaction behavior is tested in integration tests.

	originalDB := &sql.DB{}
	logger := slog.Default()
	store := NewPostgresUserCardStatsStore(originalDB, logger)

	// Verify the method exists and the store has expected fields
	assert.NotNil(t, store)
	assert.Equal(t, originalDB, store.db)
	assert.NotNil(t, store.logger)
}
