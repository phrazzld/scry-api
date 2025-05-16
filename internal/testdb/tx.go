//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"
)

// This file contains transaction management utilities for test isolation.

// WithTx runs the provided function within a database transaction.
// The transaction is automatically rolled back after the function completes,
// ensuring test isolation. This allows tests to make database modifications
// without persisting them, enabling parallel test execution.
// It provides enhanced error handling and diagnostics for transaction failures.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Verify database connection is active
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		dbURL := GetTestDatabaseURL()

		// Log more diagnostic information in CI environments
		if isCIEnvironment() {
			fmt.Printf("CI Debug: Database ping failed with error: %v\n", err)
			fmt.Printf("CI Debug: Database URL (masked): %s\n", maskDatabaseURL(dbURL))

			// Try to diagnose connection issues
			parsedURL, parseErr := url.Parse(dbURL)
			if parseErr == nil && parsedURL.User != nil {
				username := parsedURL.User.Username()
				fmt.Printf("CI Debug: Attempting connection with user: %s\n", username)
			}

			// Try a simpler query to check connectivity
			queryCtx, queryCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer queryCancel()
			var one int
			if queryErr := db.QueryRowContext(queryCtx, "SELECT 1").Scan(&one); queryErr != nil {
				fmt.Printf("CI Debug: Simple 'SELECT 1' query also failed: %v\n", queryErr)
			} else {
				fmt.Printf("CI Debug: Simple 'SELECT 1' query succeeded with result: %d\n", one)
			}
		}

		errDetail := formatDBConnectionError(err, dbURL)
		t.Fatalf("Database connection failed before transaction: %v", errDetail)
	}

	// Start a transaction with timeout context
	txCtx, txCancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout
	defer txCancel()

	tx, err := db.BeginTx(txCtx, nil)
	if err != nil {
		// Additional diagnostics in CI
		if isCIEnvironment() {
			fmt.Printf("CI Debug: Transaction start failed: %v\n", err)
			stats := db.Stats()
			fmt.Printf("CI Debug: Current connection stats: MaxOpen=%d, Open=%d, InUse=%d, Idle=%d\n",
				stats.MaxOpenConnections, stats.OpenConnections, stats.InUse, stats.Idle)
		}

		t.Fatalf(
			"Failed to begin transaction: %v\nThis may indicate database connectivity issues or resource constraints",
			err,
		)
	}

	// Add transaction metadata for debugging if available
	if tx != nil {
		// Some drivers support querying transaction state
		t.Logf("Transaction started successfully")

		// In CI, verify transaction is working with a simple query
		if isCIEnvironment() {
			var one int
			if err := tx.QueryRow("SELECT 1").Scan(&one); err != nil {
				t.Logf("Warning: Test transaction may be unstable - simple query failed: %v", err)
			}
		}
	}

	// Ensure rollback happens after test completes or fails
	defer func() {
		if r := recover(); r != nil {
			// If there was a panic, try to roll back the transaction before re-panicking
			rollbackErr := tx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				t.Logf("Warning: failed to rollback transaction after panic: %v", rollbackErr)
			}
			// Re-panic with the original error
			// ALLOW-PANIC
			panic(r)
		}

		// Normal rollback path
		err := tx.Rollback()
		// sql.ErrTxDone is expected if tx is already committed or rolled back
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Logf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Execute the test function with the transaction
	fn(t, tx)
}

// RunInTx executes the given function within a transaction.
// The transaction is automatically rolled back after the function completes.
// This function is an alias for WithTx maintained for backward compatibility.
func RunInTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Simply call WithTx to avoid code duplication
	WithTx(t, db, fn)
}
