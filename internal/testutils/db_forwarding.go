//go:build ignored_build_tag_file

// Package testutils provides test utilities and helpers for the application.
// This file has been moved to integration_exports.go with appropriate build tags
// to avoid conflicts with other files.
//
// IMPORTANT: The actual implementation has been moved to integration_exports.go
// to prevent function redeclarations with test_without_external_deps builds.
// This file is kept as a placeholder for reference but is excluded from all builds.
//
// Build Tag: This file uses the 'ignored_build_tag_file' tag to ensure it's never
// included in any build. The complex negation pattern has been simplified to a
// single unique tag for clarity.
package testutils
