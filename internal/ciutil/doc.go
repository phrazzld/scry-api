// Package ciutil provides utilities for CI and environment-specific functionality.
//
// This package contains standardized functions for detecting the execution environment (CI, local dev),
// accessing environment variables in a consistent way, and providing common utilities for CI-specific
// functionality like project root detection and database URL configuration.
//
// The primary goals of this package are:
// 1. Centralize all CI and environment detection logic
// 2. Standardize access to environment variables used for CI and testing
// 3. Provide a clean API for other packages to interact with environment-specific functionality
//
// By isolating environment detection and standardizing environment variable usage,
// we improve testability, maintainability, and consistency across the codebase.
package ciutil