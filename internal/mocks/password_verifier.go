package mocks

import "errors"

// MockPasswordVerifier implements auth.PasswordVerifier for testing
type MockPasswordVerifier struct {
	// ShouldSucceed determines whether the password comparison should succeed
	ShouldSucceed bool

	// CompareFn allows for custom comparison logic in tests
	CompareFn func(hashedPassword, password string) error

	// CompareCalledWith stores the arguments passed to Compare for verification
	CompareCalledWith struct {
		HashedPassword string
		Password       string
	}

	// CompareCallCount tracks how many times Compare was called
	CompareCallCount int
}

// Compare implements the auth.PasswordVerifier interface
func (m *MockPasswordVerifier) Compare(hashedPassword, password string) error {
	// Record call details for test verification
	m.CompareCalledWith.HashedPassword = hashedPassword
	m.CompareCalledWith.Password = password
	m.CompareCallCount++

	// Use custom function if provided
	if m.CompareFn != nil {
		return m.CompareFn(hashedPassword, password)
	}

	// Default implementation based on ShouldSucceed flag
	if m.ShouldSucceed {
		return nil // Successful comparison
	}
	return errors.New("password mismatch") // Failed comparison
}
