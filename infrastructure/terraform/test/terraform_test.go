package test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver
)

func TestTerraformDatabaseInfrastructure(t *testing.T) {
	// Skip this test unless explicitly enabled with TERRATEST_ENABLED=1
	if os.Getenv("TERRATEST_ENABLED") != "1" {
		t.Skip("Skipping infrastructure tests. Set TERRATEST_ENABLED=1 to run")
	}

	// Make sure DO token is available
	doToken := os.Getenv("DO_TOKEN")
	if doToken == "" {
		t.Fatal("DO_TOKEN environment variable must be set")
	}

	// Path to terraform code
	terraformDir := filepath.Join("..", "..")

	// Configure terraform options with variables for testing
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: terraformDir,
		Vars: map[string]interface{}{
			"do_token":          doToken,
			"cluster_name":      "scry-db-test",
			"database_name":     "scry_test",
			"node_size":         "db-s-1vcpu-1gb",   // Smallest size for testing
			"node_count":        1,                  // Single node for testing
			"database_password": "TestPassword123!", // Test password
		},
	})

	// Destroy resources once tests complete
	defer terraform.Destroy(t, terraformOptions)

	// Run terraform init and apply
	terraform.InitAndApply(t, terraformOptions)

	// Verify outputs exist
	dbHost := terraform.Output(t, terraformOptions, "database_host")
	if dbHost == "" {
		t.Fatal("Expected database_host output to be set")
	}

	dbPort := terraform.Output(t, terraformOptions, "database_port")
	if dbPort == "" {
		t.Fatal("Expected database_port output to be set")
	}

	dbName := terraform.Output(t, terraformOptions, "database_name")
	if dbName == "" {
		t.Fatal("Expected database_name output to be set")
	}

	// Test connection string format
	connectionString := terraform.OutputRequired(t, terraformOptions, "connection_string")
	if len(connectionString) < 20 { // Basic sanity check on connection string length
		t.Fatalf("Connection string appears invalid: %s", connectionString)
	}

	// Verify database connectivity
	t.Log("Attempting to connect to database using connection string")

	// Open connection to the database
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			t.Logf("Warning: failed to close database connection: %v", err)
		}
	}()

	// Set connection pool parameters
	db.SetMaxOpenConns(2)                  // Limit connections for test
	db.SetMaxIdleConns(1)                  // Keep a single connection ready
	db.SetConnMaxLifetime(time.Minute * 5) // Recreate connections after 5 minutes

	// Ping the database with timeout
	t.Log("Pinging database to verify connectivity")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
	t.Log("Successfully pinged database")

	// Execute a simple query to verify database functionality
	t.Log("Executing simple query to verify database functionality")
	var version string
	err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	t.Logf("Database version: %s", version)

	// Verify pgvector extension is available
	t.Log("Checking if pgvector extension is available")
	var extensionExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'vector')").
		Scan(&extensionExists)
	if err != nil {
		t.Fatalf("Failed to check pgvector extension availability: %v", err)
	}

	if !extensionExists {
		t.Fatal("pgvector extension is not available in the database")
	}
	t.Log("pgvector extension is available")
}
