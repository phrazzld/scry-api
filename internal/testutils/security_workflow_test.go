//go:build test_without_external_deps

package testutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecurityWorkflowExists(t *testing.T) {
	t.Parallel()

	// Check that the security workflow file exists
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "security.yml")
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Fatal("Security workflow file does not exist at expected path:", workflowPath)
	}

	// Read the workflow file to ensure it's not empty
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read security workflow file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Security workflow file is empty")
	}

	// Basic sanity checks for required content
	contentStr := string(content)

	requiredElements := []string{
		"name: Security Checks",
		"codeql:",
		"github/codeql-action/init@v3",
		"github/codeql-action/analyze@v3",
		"govulncheck:",
		"dependency-review:",
		"GO_VERSION:",
	}

	for _, element := range requiredElements {
		if !contains(contentStr, element) {
			t.Errorf("Security workflow missing required element: %s", element)
		}
	}

	t.Log("✅ Security workflow file exists and contains required elements")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
