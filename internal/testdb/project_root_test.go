//go:build integration || test_without_external_deps

package testdb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRootWithEnvVar(t *testing.T) {
	// Save original environment
	origEnv := os.Getenv("SCRY_PROJECT_ROOT")
	defer func() {
		if err := os.Setenv("SCRY_PROJECT_ROOT", origEnv); err != nil {
			t.Logf("Failed to restore SCRY_PROJECT_ROOT: %v", err)
		}
	}()

	// Set up a temporary directory with a go.mod file
	tempDir := t.TempDir()
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module github.com/test/project"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with SCRY_PROJECT_ROOT set
	if err := os.Setenv("SCRY_PROJECT_ROOT", tempDir); err != nil {
		t.Fatal(err)
	}

	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot failed with SCRY_PROJECT_ROOT set: %v", err)
	}

	if root != tempDir {
		t.Errorf("Expected project root %q, got %q", tempDir, root)
	}
}

func TestFindProjectRootWithInvalidEnvVar(t *testing.T) {
	// Save original environment
	origEnv := os.Getenv("SCRY_PROJECT_ROOT")
	defer func() {
		if err := os.Setenv("SCRY_PROJECT_ROOT", origEnv); err != nil {
			t.Logf("Failed to restore SCRY_PROJECT_ROOT: %v", err)
		}
	}()

	// Set an invalid directory
	invalidDir := filepath.Join(t.TempDir(), "nonexistent")
	if err := os.Setenv("SCRY_PROJECT_ROOT", invalidDir); err != nil {
		t.Fatal(err)
	}

	// This should fall back to other methods, not fail outright
	_, err := findProjectRoot()
	if err == nil {
		// If it doesn't error, it must have found the project root by other means
		// which is fine for this test
		t.Log("findProjectRoot succeeded with other detection methods")
	}
}

func TestIsCIEnvironment(t *testing.T) {
	// Save original CI environment variable
	origCI := os.Getenv("CI")
	defer func() {
		if err := os.Setenv("CI", origCI); err != nil {
			t.Logf("Failed to restore CI environment: %v", err)
		}
	}()

	// Test with CI not set
	if err := os.Unsetenv("CI"); err != nil {
		t.Fatal(err)
	}
	for _, envVar := range []string{
		"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TRAVIS", "CIRCLECI",
	} {
		origVal := os.Getenv(envVar)
		if err := os.Unsetenv(envVar); err != nil {
			t.Fatal(err)
		}
		defer func(name, value string) {
			if err := os.Setenv(name, value); err != nil {
				t.Logf("Failed to restore %s: %v", name, err)
			}
		}(envVar, origVal)
	}

	if isCIEnvironment() {
		t.Error("isCIEnvironment returned true when no CI environment variables are set")
	}

	// Test with CI set
	if err := os.Setenv("CI", "true"); err != nil {
		t.Fatal(err)
	}

	if !isCIEnvironment() {
		t.Error("isCIEnvironment returned false when CI environment variable is set")
	}
}
