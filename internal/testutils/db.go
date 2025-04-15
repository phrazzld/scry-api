package testutils

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pressly/goose/v3"
)

// SetupTestDatabaseSchema initializes the database schema using project migrations.
// It resets the schema to baseline (by running migrations down to version 0),
// then applies all migrations. This ensures tests run against the canonical schema.
//
// This function should typically be called once in TestMain, rather than for each test.
func SetupTestDatabaseSchema(db *sql.DB) error {
	// Set the goose dialect
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Get the project root directory
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Path to migrations directory
	migrationsDir := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

	// Verify migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	// Set custom logger for goose to avoid unwanted output during tests
	goose.SetLogger(&testGooseLogger{})

	// Reset database schema to baseline
	if err := goose.DownTo(db, migrationsDir, 0); err != nil {
		return fmt.Errorf("failed to reset database schema: %w", err)
	}

	// Apply all migrations
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// ResetTestData truncates all test tables to ensure test isolation.
// This should be called before each test to start with a clean slate.
func ResetTestData(db *sql.DB) error {
	// Use CASCADE to handle foreign key constraints
	_, err := db.Exec("TRUNCATE TABLE users CASCADE")
	if err != nil {
		return fmt.Errorf("failed to truncate users table: %w", err)
	}
	return nil
}

// findProjectRoot attempts to locate the project root directory.
// It works by searching for the go.mod file starting from the current file's directory
// and going up the directory tree.
func findProjectRoot() (string, error) {
	// Get the current file's directory
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	currentDir := filepath.Dir(currentFile)

	// Traverse up to find go.mod
	dir := currentDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached root without finding go.mod
			return "", fmt.Errorf("could not find project root (go.mod file)")
		}
		dir = parentDir
	}
}

// testGooseLogger is a simple implementation of the goose.Logger interface
// that doesn't output anything during tests to keep output clean.
type testGooseLogger struct{}

func (*testGooseLogger) Fatal(v ...interface{}) {
	fmt.Println(v...)
	os.Exit(1)
}

func (*testGooseLogger) Fatalf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(1)
}

func (*testGooseLogger) Print(v ...interface{}) {
	// Silence regular prints during tests
}

func (*testGooseLogger) Println(v ...interface{}) {
	// Silence regular prints during tests
}

func (*testGooseLogger) Printf(format string, v ...interface{}) {
	// Silence regular prints during tests
}

