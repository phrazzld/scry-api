package postgres

import (
	"database/sql"
	"testing"

	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestNewPostgresUserStore(t *testing.T) {
	tests := []struct {
		name       string
		db         store.DBTX
		bcryptCost int
		check      func(t *testing.T, store *PostgresUserStore)
	}{
		{
			name:       "valid_db_with_valid_cost",
			db:         &sql.DB{},
			bcryptCost: 12,
			check: func(t *testing.T, store *PostgresUserStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.Equal(t, 12, store.bcryptCost)
			},
		},
		{
			name:       "valid_db_with_zero_cost_uses_default",
			db:         &sql.DB{},
			bcryptCost: 0,
			check: func(t *testing.T, store *PostgresUserStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.Equal(t, bcrypt.DefaultCost, store.bcryptCost)
			},
		},
		{
			name:       "valid_db_with_cost_too_low_uses_default",
			db:         &sql.DB{},
			bcryptCost: 3,
			check: func(t *testing.T, store *PostgresUserStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.Equal(t, bcrypt.DefaultCost, store.bcryptCost)
			},
		},
		{
			name:       "valid_db_with_cost_too_high_uses_default",
			db:         &sql.DB{},
			bcryptCost: 32,
			check: func(t *testing.T, store *PostgresUserStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.Equal(t, bcrypt.DefaultCost, store.bcryptCost)
			},
		},
		{
			name:       "nil_db_still_creates_store",
			db:         nil,
			bcryptCost: 10,
			check: func(t *testing.T, store *PostgresUserStore) {
				assert.NotNil(t, store)
				assert.Nil(t, store.db)
				assert.Equal(t, 10, store.bcryptCost)
			},
		},
		{
			name:       "mock_dbtx",
			db:         &mockDBTX{},
			bcryptCost: 10,
			check: func(t *testing.T, store *PostgresUserStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
				assert.Equal(t, 10, store.bcryptCost)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewPostgresUserStore(tt.db, tt.bcryptCost)
			if tt.check != nil {
				tt.check(t, store)
			}
		})
	}
}

func TestPostgresUserStore_WithTx(t *testing.T) {
	// Note: We can't create a real *sql.Tx without a database connection,
	// so we'll test the behavior by checking the store structure.
	// The actual transaction behavior is tested in integration tests.

	originalDB := &sql.DB{}
	bcryptCost := 12
	store := NewPostgresUserStore(originalDB, bcryptCost)

	// Verify the method exists and the store has expected fields
	assert.NotNil(t, store)
	assert.Equal(t, originalDB, store.db)
	assert.Equal(t, bcryptCost, store.bcryptCost)
}

func TestPostgresUserStore_DB(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *PostgresUserStore
		expectDB  bool
		expectNil bool
	}{
		{
			name: "returns_sql_db_when_initialized_with_db",
			setup: func() *PostgresUserStore {
				db := &sql.DB{}
				return NewPostgresUserStore(db, 10)
			},
			expectDB: true,
		},
		{
			name: "returns_mock_dbtx_when_initialized_with_mock",
			setup: func() *PostgresUserStore {
				mockDB := &mockDBTX{}
				return NewPostgresUserStore(mockDB, 10)
			},
			expectDB: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setup()
			result := store.DB()

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				if tt.expectDB {
					_, ok := result.(*sql.DB)
					assert.True(t, ok, "expected *sql.DB")
				}
			}
		})
	}
}
