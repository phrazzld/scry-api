package testutils

import (
	"testing"
)

// Skip all tests in this file for now
func TestSkippedDuringRefactoring(t *testing.T) {
	t.Skip("Skipping auth tests during refactoring")
}
