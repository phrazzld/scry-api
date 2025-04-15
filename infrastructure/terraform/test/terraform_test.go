package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
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

	// TODO: Use the connection string to test database connectivity and run migrations
	// This would require importing the database/sql package and executing some queries
	// as well as potentially running the migration code.
	// Also verify that the database_password variable works correctly by attempting
	// to authenticate with the specified password.
}
