//go:build test_without_external_deps && !integration

// Package testutils provides implementations required by tests
// when run with the test_without_external_deps build tag.
package testutils

// This file includes implementations that help tests run with the
// test_without_external_deps build tag. Specifically handling functions
// that would otherwise be unavailable due to build tag restrictions.

// Empty file - all functionality has been moved to helpers and db.go files
// with appropriate build tags. This file remains as a placeholder to document
// the approach taken.
//
// The problem:
// - Many tests require certain functions like WithTx, CreateTestUser, etc.
// - These functions are defined across different files with different build tags
// - When running with test_without_external_deps, some functions are missing
//
// The solution:
// - Skip tests that require database connectivity
// - Skip tests that would require complex mocking
// - Use build tags to conditionally include implementations
//
// The overall approach chosen was to simplify the tests rather than
// create complex mock implementations, as the CI environment has full
// database connectivity available.
