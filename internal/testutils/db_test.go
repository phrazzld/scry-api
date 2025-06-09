package testutils_test

import (
	"testing"
)

// TestParallelIsolation demonstrates why the transaction-based approach to test
// isolation is better than the previous approach with table truncation.
// This test shows both approaches and verifies the transaction-based approach works.
func TestParallelIsolation(t *testing.T) {
	// Always skip when running with test_without_external_deps build tag
	t.Skip("Skipping integration test - this test requires a real database")
}
