package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phrazzld/scry-api/tools/buildtags/shared"
)

func TestValidateBuildTags(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create a file with valid build tags
	validFile := filepath.Join(tmpDir, "valid.go")
	validContent := `//go:build integration || test_without_external_deps

package main

func ExportedFunction() {
	// function body
}
`
	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatalf("Failed to create valid test file: %v", err)
	}

	// Test validation passes for valid files
	if err := validateBuildTags(tmpDir); err != nil {
		t.Errorf("Expected validation to pass, but got error: %v", err)
	}
}

func TestScanForTaggedFiles(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create files with different tag patterns
	tests := []struct {
		filename string
		content  string
		hasTag   bool
	}{
		{
			"tagged.go",
			"//go:build integration\n\npackage main\n",
			true,
		},
		{
			"old_style.go",
			"// +build integration\n\npackage main\n",
			true,
		},
		{
			"no_tags.go",
			"package main\n",
			false,
		},
	}

	expectedTaggedCount := 0
	for _, tt := range tests {
		filepath := filepath.Join(tmpDir, tt.filename)
		if err := os.WriteFile(filepath, []byte(tt.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", tt.filename, err)
		}
		if tt.hasTag {
			expectedTaggedCount++
		}
	}

	// Scan for tagged files
	taggedFiles, err := scanForTaggedFiles(tmpDir)
	if err != nil {
		t.Fatalf("Failed to scan for tagged files: %v", err)
	}

	if len(taggedFiles) != expectedTaggedCount {
		t.Errorf("Expected %d tagged files, got %d", expectedTaggedCount, len(taggedFiles))
	}
}

func TestBuildTagConflictDetection(t *testing.T) {
	// Create temp directory with conflicting files
	tmpDir := t.TempDir()

	// Create file with positive tag
	posFile := filepath.Join(tmpDir, "positive.go")
	posContent := `//go:build integration

package main

func PositiveFunction() {}
`
	if err := os.WriteFile(posFile, []byte(posContent), 0644); err != nil {
		t.Fatalf("Failed to create positive test file: %v", err)
	}

	// Create file with negative tag
	negFile := filepath.Join(tmpDir, "negative.go")
	negContent := `//go:build !integration

package main

func NegativeFunction() {}
`
	if err := os.WriteFile(negFile, []byte(negContent), 0644); err != nil {
		t.Fatalf("Failed to create negative test file: %v", err)
	}

	// Validate should detect conflict
	err := validateBuildTags(tmpDir)
	if err == nil {
		t.Error("Expected validation to fail due to build tag conflict, but it passed")
	}

	// Check that conflict was detected by shared package
	fileInfos, err := shared.ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	conflicts := shared.DetectConflicts(fileInfos)
	if len(conflicts) == 0 {
		t.Error("Expected to detect conflicts, but none were found")
	}

	// Check that the conflict is for the "integration" tag
	found := false
	for _, conflict := range conflicts {
		if conflict.Tag == "integration" && conflict.Type == "negation" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find integration tag negation conflict")
	}
}
