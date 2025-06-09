//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
// It maintains a clean dependency structure by only depending on store interfaces
// and standard database packages, not on specific implementations.
package testdb

import (
	"time"
)

// TestTimeout defines a default timeout for test database operations.
const TestTimeout = 5 * time.Second
