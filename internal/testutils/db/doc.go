// Package db provides database testing utilities for the Scry API.
//
// This package focuses on database-specific test helpers, such as transaction
// management, connection pooling, and isolation patterns. It aims to simplify
// writing database integration tests by providing a consistent pattern for
// database operations in tests.
//
// Key features:
// - Transaction isolation for parallel test execution
// - Connection pool management with sane defaults for testing
// - Automatic schema management via migrations
// - Convenient helpers for common database operations
//
// For general testing utilities, see the parent testutils package.
package db
