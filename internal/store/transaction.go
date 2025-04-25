// Package store provides abstractions and implementations for data persistence
package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/phrazzld/scry-api/internal/platform/logger"
)

// TxFn is a function that executes within a database transaction.
// It receives the context and a transaction, and returns an error if the operation fails.
// The transaction is committed if the function returns nil, or rolled back if it returns an error.
type TxFn func(ctx context.Context, tx *sql.Tx) error

// RunInTransaction executes the given function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
// The function handles rollbacks in case of panic and logs appropriate information.
func RunInTransaction(ctx context.Context, db *sql.DB, fn TxFn) error {
	// Get logger from context or use default
	log := logger.FromContext(ctx)

	// Begin a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction",
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Set up defer to handle panics and roll back the transaction if needed
	defer func() {
		if p := recover(); p != nil {
			// Attempt to roll back the transaction in case of panic
			txErr := tx.Rollback()
			if txErr != nil {
				log.Error("failed to roll back transaction after panic",
					slog.String("error", txErr.Error()),
					slog.Any("panic", p))
			} else {
				log.Error("rolled back transaction after panic",
					slog.Any("panic", p))
			}
			// Re-panic to maintain the behavior
			// ALLOW-PANIC: Propagating caught panic from transaction
			panic(p)
		}
	}()

	// Execute the provided function within the transaction
	err = fn(ctx, tx)
	if err != nil {
		// If the function returns an error, roll back the transaction
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			log.Error("failed to roll back transaction",
				slog.String("rollback_error", rollbackErr.Error()),
				slog.String("original_error", err.Error()))
			// Return the combined errors to provide complete information
			return fmt.Errorf(
				"error rolling back transaction: %v (original error: %w)",
				rollbackErr,
				err,
			)
		}
		log.Debug("rolled back transaction due to error",
			slog.String("error", err.Error()))
		// Return the original error
		return err
	}

	// If the function executed successfully, commit the transaction
	err = tx.Commit()
	if err != nil {
		log.Error("failed to commit transaction",
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug("transaction committed successfully")
	return nil
}
