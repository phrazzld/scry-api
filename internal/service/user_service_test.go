package service_test

import (
	"testing"
)

// Note: We're skipping transaction-based tests in this package since they're better suited
// for integration tests. See cmd/server/*_test.go for transaction-based testing.

func TestUserService_UpdateUserEmail(t *testing.T) {
	// Skip test with transaction mocking - this would be tested in an integration test
	t.Skip("Skipping test that requires transaction management")
}

func TestUserService_UpdateUserPassword(t *testing.T) {
	// Skip test with transaction mocking - this would be tested in an integration test
	t.Skip("Skipping test that requires transaction management")
}
