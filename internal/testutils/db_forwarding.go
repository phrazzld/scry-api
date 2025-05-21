//go:build !integration_test_internal && !integration && !test_without_external_deps && !ignored_build_tag_file

// Package testutils provides test utilities and helpers for the application.
// This file has been moved to integration_exports.go with appropriate build tags
// to avoid conflicts with other files.
//
// IMPORTANT: The actual implementation has been moved to integration_exports.go
// to prevent function redeclarations with test_without_external_deps builds.
// This file is kept as a placeholder for reference but is excluded from all builds.
package testutils
