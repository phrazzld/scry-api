package local_dev

import (
	"database/sql"
	"fmt"
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

	// Find the working directory for docker-compose
	workDir := filepath.Join("..", "local_dev")
	if _, err := os.Stat(filepath.Join(workDir, "docker-compose.yml")); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		err := os.MkdirAll(workDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Generate docker-compose file
		err = generateDockerComposeYml(workDir)
		if err != nil {
			t.Fatalf("Failed to generate docker-compose.yml: %v", err)
		}

		// Generate init script
		err = generateInitScript(workDir)
		if err != nil {
			t.Fatalf("Failed to generate init script: %v", err)
		}
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
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&extensionExists)
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
	_, err = db.Exec(
		"CREATE TABLE IF NOT EXISTS goose_db_version (id SERIAL PRIMARY KEY, version_id BIGINT NOT NULL, is_applied BOOLEAN NOT NULL, tstamp TIMESTAMP WITH TIME ZONE DEFAULT NOW())",
	)
	if err != nil {
		t.Fatalf("Failed to create migration table: %v", err)
	}

	t.Log("Local PostgreSQL setup verified successfully")
}

// Helper function to generate docker-compose.yml
func generateDockerComposeYml(dir string) error {
	dockerComposeContent := `version: '3.8'

services:
  postgres:
    image: pgvector/pgvector:pg15
    environment:
      POSTGRES_DB: scry
      POSTGRES_USER: scryapiuser
      POSTGRES_PASSWORD: local_development_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-scripts:/docker-entrypoint-initdb.d
    command: ["postgres", "-c", "shared_buffers=128MB", "-c", "work_mem=16MB", "-c", "max_connections=50"]

volumes:
  postgres_data:
`

	// Create docker-compose.yml
	err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(dockerComposeContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	return nil
}

// Helper function to generate init script
func generateInitScript(dir string) error {
	// Create init-scripts directory
	initScriptsDir := filepath.Join(dir, "init-scripts")
	err := os.MkdirAll(initScriptsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create init-scripts directory: %w", err)
	}

	// Create init script
	initScriptContent := `-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Add any additional setup required for development
`

	err = os.WriteFile(filepath.Join(initScriptsDir, "01-init.sql"), []byte(initScriptContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write init script: %w", err)
	}

	return nil
}
