package scripts

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMigrationScript(t *testing.T) {
	// Skip if no database URL is provided
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping migration script test - TEST_DATABASE_URL not set")
	}

	// Find the script path relative to this test file
	scriptPath := "./test-migrations.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("Could not find test-migrations.sh script at %s", scriptPath)
	}

	// Ensure the script is executable
	err := os.Chmod(scriptPath, 0755)
	if err != nil {
		t.Fatalf("Could not make script executable: %v", err)
	}

	// Create command to run the script
	cmd := exec.Command(scriptPath)

	// Set the database URL in the environment
	cmd.Env = append(os.Environ(), "DATABASE_URL="+dbURL)

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for script execution errors
	if err != nil {
		t.Fatalf("Script execution failed: %v\nOutput: %s", err, outputStr)
	}

	// Check for expected output indicating successful migration
	if !strings.Contains(outputStr, "Migration test completed successfully") {
		t.Errorf("Script did not complete successfully. Output: %s", outputStr)
	}
}
