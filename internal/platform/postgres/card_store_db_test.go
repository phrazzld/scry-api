package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCardStore_DB_WithTx verifies that the DB() method returns a non-nil *sql.DB
// even after calling WithTx, as required by T005.
func TestCardStore_DB_WithTx(t *testing.T) {
	// Create a mock DB and tx
	db := &sql.DB{} // Use an empty DB struct for testing
	var tx *sql.Tx  // nil tx is fine for this test

	// Create a card store
	cardStore := NewPostgresCardStore(db, nil)

	// Verify initial DB() returns the expected connection
	assert.NotNil(t, cardStore.DB(), "DB() should return the SQL DB")
	assert.Equal(t, db, cardStore.DB(), "DB() should return the original DB connection")

	// We're using a nil tx which is fine for this test
	// In a real scenario, tx would be the result of db.Begin()

	// Get a transactional card store
	txCardStore := cardStore.WithTx(tx)

	// Verify DB() still returns the original connection
	assert.NotNil(t, txCardStore.DB(), "DB() should return non-nil after WithTx")
	assert.Equal(t, db, txCardStore.DB(), "DB() should return the original DB connection after WithTx")
}
