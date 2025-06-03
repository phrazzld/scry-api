//go:build test_without_external_deps

package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInTransaction_Success(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Expect the transaction to begin, and then commit
	mock.ExpectBegin()
	mock.ExpectCommit()

	// Create a function that will succeed
	successFn := func(ctx context.Context, tx *sql.Tx) error {
		return nil
	}

	// Execute the transaction
	err = RunInTransaction(context.Background(), db, successFn)
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunInTransaction_FunctionError(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Expect the transaction to begin, then rollback due to error
	mock.ExpectBegin()
	mock.ExpectRollback()

	// Create a function that will fail
	expectedErr := errors.New("function failed")
	failFn := func(ctx context.Context, tx *sql.Tx) error {
		return expectedErr
	}

	// Execute the transaction
	err = RunInTransaction(context.Background(), db, failFn)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunInTransaction_BeginTransactionError(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Expect the transaction begin to fail
	expectedErr := errors.New("begin transaction failed")
	mock.ExpectBegin().WillReturnError(expectedErr)

	// Create a simple function
	fn := func(ctx context.Context, tx *sql.Tx) error {
		return nil
	}

	// Execute the transaction
	err = RunInTransaction(context.Background(), db, fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.ErrorIs(t, err, expectedErr)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunInTransaction_CommitError(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Expect the transaction to begin and then fail to commit
	mock.ExpectBegin()
	expectedErr := errors.New("commit failed")
	mock.ExpectCommit().WillReturnError(expectedErr)

	// Create a function that will succeed
	successFn := func(ctx context.Context, tx *sql.Tx) error {
		return nil
	}

	// Execute the transaction
	err = RunInTransaction(context.Background(), db, successFn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit transaction")
	assert.ErrorIs(t, err, expectedErr)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunInTransaction_RollbackError(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Expect the transaction to begin, then fail to rollback
	mock.ExpectBegin()
	functionErr := errors.New("function failed")
	rollbackErr := errors.New("rollback failed")
	mock.ExpectRollback().WillReturnError(rollbackErr)

	// Create a function that will fail
	failFn := func(ctx context.Context, tx *sql.Tx) error {
		return functionErr
	}

	// Execute the transaction
	err = RunInTransaction(context.Background(), db, failFn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error rolling back transaction")
	assert.Contains(t, err.Error(), "rollback failed")
	assert.Contains(t, err.Error(), "original error")
	assert.ErrorIs(t, err, functionErr)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunInTransaction_Panic(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Expect the transaction to begin, then rollback due to panic
	mock.ExpectBegin()
	mock.ExpectRollback()

	// Create a function that will panic
	panicFn := func(ctx context.Context, tx *sql.Tx) error {
		panic("test panic")
	}

	// Execute the transaction and expect it to panic
	assert.Panics(t, func() {
		_ = RunInTransaction(context.Background(), db, panicFn)
	})

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunInTransaction_PanicWithRollbackError(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Expect the transaction to begin, then fail to rollback after panic
	mock.ExpectBegin()
	rollbackErr := errors.New("rollback failed")
	mock.ExpectRollback().WillReturnError(rollbackErr)

	// Create a function that will panic
	panicFn := func(ctx context.Context, tx *sql.Tx) error {
		panic("test panic")
	}

	// Execute the transaction and expect it to panic
	assert.Panics(t, func() {
		_ = RunInTransaction(context.Background(), db, panicFn)
	})

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTxFn tests the TxFn type to ensure it's properly defined
func TestTxFn(t *testing.T) {
	// Create a simple TxFn
	var fn TxFn = func(ctx context.Context, tx *sql.Tx) error {
		return nil
	}

	// Verify it can be called
	assert.NotNil(t, fn)

	// Test with a mock transaction
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)

	err = fn(context.Background(), tx)
	assert.NoError(t, err)
}
