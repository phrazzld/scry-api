// Package mocks provides centralized mock implementations for testing.
//
// This package contains mock implementations of interfaces used throughout the application,
// facilitating consistent and DRY testing across the codebase. Instead of defining
// inline mocks in individual test files, these standardized mock implementations
// can be reused.
//
// Key Features:
//
//   - Consistent mock behavior across different test packages
//   - Simplified test setup with reusable mock implementations
//   - Reduced duplication of mock logic across test files
//   - Easy maintenance of mock behaviors in a central location
//
// Usage:
//
// Import the mocks package in your test file and create the required mock:
//
//	import "github.com/phrazzld/scry-api/internal/mocks"
//
//	func TestSomething(t *testing.T) {
//	    mockJWTService := &mocks.MockJWTService{
//	        GenerateTokenFn: func(userID string) (string, error) {
//	            return "mocked-token", nil
//	        },
//	    }
//
//	    // Use the mock in your test...
//	}
//
// When adding a new mock to this package:
//  1. Create a new file named after the interface being mocked
//  2. Implement the mock struct with function fields for each interface method
//  3. Document any helper methods or special functionality
//  4. Update existing tests to use the centralized mock implementation
package mocks
