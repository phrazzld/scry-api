package postgres

import (
	"database/sql"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWithTxCardStore tests the deprecated WithTxCardStore method
func TestWithTxCardStore(t *testing.T) {
	// Create a card store
	db := &sql.DB{}
	logger := slog.Default()
	store := NewPostgresCardStore(db, logger)

	// Create a mock transaction (we can't create a real one without DB connection)
	// But we can test that the method exists and behaves consistently with WithTx
	tx := &sql.Tx{}

	// The deprecated method should exist and return a CardStore interface
	// We're testing that it delegates to WithTx correctly
	result := store.WithTxCardStore(tx)
	assert.NotNil(t, result)

	// Verify it returns the same type as WithTx would
	resultWithTx := store.WithTx(tx)
	assert.NotNil(t, resultWithTx)

	// Both should return CardStore interfaces (structural test)
	assert.IsType(t, result, resultWithTx)
}

// TestMemoStoreWithTx tests the WithTx method for memo store
func TestMemoStoreWithTx(t *testing.T) {
	// Create a memo store
	db := &sql.DB{}
	logger := slog.Default()
	store := NewPostgresMemoStore(db, logger)

	// Create a mock transaction
	tx := &sql.Tx{}

	// Test WithTx returns a new store instance
	result := store.WithTx(tx)
	assert.NotNil(t, result)

	// Verify it returns a MemoStore interface
	_, ok := result.(*PostgresMemoStore)
	assert.True(t, ok, "WithTx should return a PostgresMemoStore instance")

	// The new store should use the transaction as its db
	resultStore := result.(*PostgresMemoStore)
	assert.Equal(t, tx, resultStore.db, "WithTx store should use the provided transaction")
	assert.Equal(t, store.logger, resultStore.logger, "WithTx store should preserve the logger")
}

// TestStatsStoreWithTx tests the WithTx method for stats store
func TestStatsStoreWithTx(t *testing.T) {
	// Create a stats store
	db := &sql.DB{}
	logger := slog.Default()
	store := NewPostgresUserCardStatsStore(db, logger)

	// Create a mock transaction
	tx := &sql.Tx{}

	// Test WithTx returns a new store instance
	result := store.WithTx(tx)
	assert.NotNil(t, result)

	// Verify it returns a UserCardStatsStore interface
	_, ok := result.(*PostgresUserCardStatsStore)
	assert.True(t, ok, "WithTx should return a PostgresUserCardStatsStore instance")

	// The new store should use the transaction as its db
	resultStore := result.(*PostgresUserCardStatsStore)
	assert.Equal(t, tx, resultStore.db, "WithTx store should use the provided transaction")
	assert.Equal(t, store.logger, resultStore.logger, "WithTx store should preserve the logger")
}

// TestTaskStoreWithTx tests the WithTx method for task store
func TestTaskStoreWithTx(t *testing.T) {
	// Create a task store
	db := &sql.DB{}
	store := NewPostgresTaskStore(db)

	// Create a mock transaction
	tx := &sql.Tx{}

	// Test WithTx returns a new store instance
	result := store.WithTx(tx)
	assert.NotNil(t, result)

	// Verify it returns a TaskStore interface
	_, ok := result.(*PostgresTaskStore)
	assert.True(t, ok, "WithTx should return a PostgresTaskStore instance")

	// The new store should use the transaction as its db
	resultStore := result.(*PostgresTaskStore)
	assert.Equal(t, tx, resultStore.db, "WithTx store should use the provided transaction")
}

// TestUserStoreWithTx tests the WithTx method for user store
func TestUserStoreWithTx(t *testing.T) {
	// Create a user store
	db := &sql.DB{}
	bcryptCost := 10
	store := NewPostgresUserStore(db, bcryptCost)

	// Create a mock transaction
	tx := &sql.Tx{}

	// Test WithTx returns a new store instance
	result := store.WithTx(tx)
	assert.NotNil(t, result)

	// Verify it returns a UserStore interface
	_, ok := result.(*PostgresUserStore)
	assert.True(t, ok, "WithTx should return a PostgresUserStore instance")

	// The new store should use the transaction as its db
	resultStore := result.(*PostgresUserStore)
	assert.Equal(t, tx, resultStore.db, "WithTx store should use the provided transaction")
	assert.Equal(t, store.bcryptCost, resultStore.bcryptCost, "WithTx store should preserve bcrypt cost")
}
