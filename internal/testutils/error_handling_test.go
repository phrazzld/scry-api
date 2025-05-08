//go:build skip_testutils_test

package testutils_test

import (
	"io"
	"os"
	"testing"

	"github.com/phrazzld/scry-api/internal/testutils"
)

// errCloser is a simple implementation of io.Closer that returns a predefined error
type errCloser struct {
	err error
}

// Close implements the io.Closer interface
func (e *errCloser) Close() error {
	return e.err
}

// Verify that errCloser implements io.Closer
var _ io.Closer = (*errCloser)(nil)

// TestAssertCloseNoError tests the AssertCloseNoError helper function
func TestAssertCloseNoError(t *testing.T) {
	t.Parallel()

	t.Run("nil closer", func(t *testing.T) {
		t.Parallel()
		// This should not panic
		testutils.AssertCloseNoError(t, nil)
		// No assertion needed - if it doesn't panic, the test passes
	})

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		closer := &errCloser{err: nil}
		// This should not fail the test
		testutils.AssertCloseNoError(t, closer)
		// No assertion needed - if it doesn't fail, the test passes
	})

	// Note: We can't easily test the failure case in a unit test without mocking testing.T,
	// which is outside the scope of this task. For the failure case, we'd need integration
	// tests that actually connect to resources that might fail on Close().
}

// The following test demonstrates using our error handling helpers in a realistic scenario.
// This test setup is similar to what we'll update in the actual test files.
func TestWithErrorHandling(t *testing.T) {
	t.Parallel()

	// Skip if we're not in an environment where we can create actual resources
	// This is similar to how many of our tests check for environment variables
	// before running integration tests.
	if testing.Short() {
		t.Skip("Skipping test that requires creating resources")
	}

	// Create some temp files to test with
	tempDir := t.TempDir()
	configPath := tempDir + "/config.yaml"

	// Create a file that we'll use to test cleanup
	err := os.WriteFile(configPath, []byte("test: config"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try reading the file to make sure it exists
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != "test: config" {
		t.Fatalf("File content doesn't match expected: %s", content)
	}

	// Open a file that we can close with our AssertCloseNoError helper
	file, err := os.Open(configPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	// Here's where we use our helper instead of a traditional defer
	defer testutils.AssertCloseNoError(t, file)

	// Now we can do the actual test without worrying about checking the error from close
	// This is similar to how other tests would use our helper
	content2 := make([]byte, len("test: config"))
	n, err := file.Read(content2)
	if err != nil {
		t.Fatalf("Failed to read from file: %v", err)
	}
	if n != len("test: config") {
		t.Fatalf("Read incorrect number of bytes: %d", n)
	}
	if string(content2) != "test: config" {
		t.Fatalf("File content doesn't match expected: %s", content2)
	}
}
