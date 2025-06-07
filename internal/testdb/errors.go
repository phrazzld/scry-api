//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import (
	"fmt"
	"os"
)

// This file contains error formatting utilities for enhanced diagnostics.

// formatDBConnectionError creates a detailed error message for database connection failures.
// It includes environment variable status, connection details, and troubleshooting guidance.
func formatDBConnectionError(baseErr error, dbURL string) error {
	// Basic environment info
	envInfo := fmt.Sprintf("CI environment: %v\nCurrent working directory: %s",
		isCIEnvironment(), getCurrentDir())

	// Database connection info (safely masked)
	dbInfo := fmt.Sprintf("Database URL used: %s (masked)", maskDatabaseURL(dbURL))

	// Format the comprehensive error message
	errMsg := fmt.Sprintf("Database connection failed: %v\n%s\n%s\n"+
		"Please check:\n"+
		"1. PostgreSQL service is running\n"+
		"2. Credentials and connection string are correct\n"+
		"3. Database exists and is accessible\n"+
		"4. Network connectivity and firewall settings",
		baseErr, dbInfo, envInfo)

	return fmt.Errorf("%s", errMsg)
}

// formatEnvVarError creates a detailed error message when required environment variables are missing.
// It provides guidance on which variables should be set and current environment status.
func formatEnvVarError() error {
	// Check which environment variables are missing
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}
	missingVars := []string{}

	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	// Environment status information
	envInfo := fmt.Sprintf("CI environment: %v\nCurrent working directory: %s",
		isCIEnvironment(), getCurrentDir())

	// Create the error message
	errMsg := fmt.Sprintf("Database connection failed: no database URL available\n"+
		"Required environment variables missing: %v\n%s\n"+
		"Please ensure one of DATABASE_URL, SCRY_TEST_DB_URL, or SCRY_DATABASE_URL is set.",
		missingVars, envInfo)

	return fmt.Errorf("%s", errMsg)
}

// formatProjectRootError creates a detailed error message when the project root cannot be found.
// It includes paths checked, environment variables, and suggested actions.
func formatProjectRootError(checkedPaths []string, checkedEnvVars []string) error {
	dir := getCurrentDir()

	// Get current directory contents for debugging
	dirContents := ""
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		dirContents = fmt.Sprintf("\nCurrent directory contents: %v", names)
	}

	// Create comprehensive error message
	errMsg := fmt.Sprintf("Could not find go.mod in any parent directory.\n"+
		"Checked environment variables: %v\n"+
		"Checked paths: %v\n"+
		"Current directory: %s%s\n"+
		"CI environment: %v\n"+
		"CI environment vars: GITHUB_WORKSPACE=%s, CI_PROJECT_DIR=%s\n"+
		"To fix this, set SCRY_PROJECT_ROOT environment variable to the project root directory",
		checkedEnvVars, checkedPaths, dir, dirContents,
		isCIEnvironment(), os.Getenv("GITHUB_WORKSPACE"), os.Getenv("CI_PROJECT_DIR"))

	return fmt.Errorf("%s", errMsg)
}

// formatMigrationError creates a detailed error message when database migrations fail.
// It includes information about the migrations directory, error details, and suggestions.
func formatMigrationError(baseErr error, migrationsDir string) error {
	// Verify if the migrations directory exists
	dirExists := "exists"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		dirExists = "does not exist"
	}

	// Get list of migration files for debugging if directory exists
	migrationFiles := ""
	if dirExists == "exists" {
		if entries, err := os.ReadDir(migrationsDir); err == nil && len(entries) > 0 {
			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					names = append(names, entry.Name())
				}
			}
			migrationFiles = fmt.Sprintf("\nMigration files: %v", names)
		}
	}

	// Environment information
	envInfo := fmt.Sprintf("CI environment: %v\nCurrent working directory: %s",
		isCIEnvironment(), getCurrentDir())

	// Create comprehensive error message
	errMsg := fmt.Sprintf("Failed to run database migrations: %v\n"+
		"Migrations directory: %s (%s)%s\n%s\n"+
		"Please check:\n"+
		"1. Migrations directory path is correct\n"+
		"2. Migration files exist and are valid\n"+
		"3. Database connection is working\n"+
		"4. Database user has permissions to create tables and modify schema",
		baseErr, migrationsDir, dirExists, migrationFiles, envInfo)

	return fmt.Errorf("%s", errMsg)
}
