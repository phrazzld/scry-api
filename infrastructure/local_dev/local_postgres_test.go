package local_dev

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestLocalPostgresSetup verifies the Docker-based local PostgreSQL setup
func TestLocalPostgresSetup(t *testing.T) {
	// Skip if DOCKER_TEST is not set to avoid running during standard test suite
	if os.Getenv("DOCKER_TEST") != "1" {
		t.Skip("Skipping Docker-based PostgreSQL test. Set DOCKER_TEST=1 to run")
	}

	// Set the working directory for docker-compose using filepath.Join for consistency
	workDir := filepath.Join(".")

	// Verify required files exist
	dockerComposeFile := filepath.Join(workDir, "docker-compose.yml")
	if _, err := os.Stat(dockerComposeFile); os.IsNotExist(err) {
		t.Fatalf("Required file not found: %s", dockerComposeFile)
	}

	initScriptFile := filepath.Join(workDir, "init-scripts", "01-init.sql")
	if _, err := os.Stat(initScriptFile); os.IsNotExist(err) {
		t.Fatalf("Required file not found: %s", initScriptFile)
	}

	// Clean up previous container if it exists
	cleanupCmd := exec.Command("docker-compose", "down", "-v")
	cleanupCmd.Dir = workDir
	cleanupOutput, err := cleanupCmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning during cleanup: %v\nOutput: %s", err, string(cleanupOutput))
		// Don't fail the test on cleanup errors
	}

	// Start PostgreSQL container
	startCmd := exec.Command("docker-compose", "up", "-d")
	startCmd.Dir = workDir
	startOutput, err := startCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to start container: %v\nOutput: %s", err, string(startOutput))
	}

	// Defer cleanup
	defer func() {
		cleanupCmd := exec.Command("docker-compose", "down", "-v")
		cleanupCmd.Dir = workDir
		err := cleanupCmd.Run()
		if err != nil {
			t.Logf("Warning: failed to clean up container: %v", err)
		}
	}()

	// Wait for PostgreSQL to be ready
	time.Sleep(3 * time.Second)

	// Test database connection and pgvector extension
	dbURL := "postgres://scryapiuser:local_development_password@localhost:5432/scry?sslmode=disable"
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			t.Logf("Warning: failed to close database connection: %v", err)
		}
	}()

	// Ping the database
	err = db.Ping()
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Check if pgvector extension is enabled
	var extensionExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')").
		Scan(&extensionExists)
	if err != nil {
		t.Fatalf("Failed to check pgvector extension: %v", err)
	}

	if !extensionExists {
		t.Fatal("pgvector extension is not enabled")
	}

	// Export DATABASE_URL for migrations test
	err = os.Setenv("DATABASE_URL", dbURL)
	if err != nil {
		t.Fatalf("Failed to set DATABASE_URL: %v", err)
	}

	// Try running migrations (this would ideally call the migrations code directly)
	// For now, just check if migration table can be created
	// Use schema_migrations to maintain consistency with the rest of the codebase
	_, err = db.Exec(
		"CREATE TABLE IF NOT EXISTS schema_migrations (id SERIAL PRIMARY KEY, version_id BIGINT NOT NULL, is_applied BOOLEAN NOT NULL, tstamp TIMESTAMP WITH TIME ZONE DEFAULT NOW())",
	)
	if err != nil {
		t.Fatalf("Failed to create migration table: %v", err)
	}

	t.Log("Local PostgreSQL setup verified successfully")
}
