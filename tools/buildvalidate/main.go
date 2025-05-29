package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phrazzld/scry-api/tools/buildtags/shared"
)

// validateBuildTags validates build tags for conflicts and compatibility
func validateBuildTags(rootDir string) error {
	fileInfos, err := shared.ScanDirectory(rootDir)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	// Check for conflicts
	conflicts := shared.DetectConflicts(fileInfos)
	if len(conflicts) > 0 {
		fmt.Printf("Found %d build tag conflicts:\n", len(conflicts))
		for _, conflict := range conflicts {
			fmt.Printf("  - %s (%s): %s\n", conflict.Tag, conflict.Type, conflict.Description)
			for _, file := range conflict.Files {
				fmt.Printf("    %s\n", file)
			}
		}
		return fmt.Errorf("build tag conflicts detected")
	}

	// Check CI compatibility
	ciIssues := shared.ValidateCICompatibility(fileInfos)
	if len(ciIssues) > 0 {
		fmt.Printf("Found %d CI compatibility issues:\n", len(ciIssues))
		for _, issue := range ciIssues {
			fmt.Printf("  - %s\n", issue)
		}
		return fmt.Errorf("CI compatibility issues detected")
	}

	fmt.Println("Build tag validation passed")
	return nil
}

// scanForTaggedFiles scans a directory and returns files with build tags
func scanForTaggedFiles(root string) ([]string, error) {
	var taggedFiles []string
	buildTagRe := regexp.MustCompile(`^(//\s*\+build|//go:build)\s+`)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and .git directories
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == ".git") {
			return filepath.SkipDir
		}

		// Process only .go files
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			scanner := bufio.NewScanner(strings.NewReader(string(content)))
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "package ") {
					break
				}
				if buildTagRe.MatchString(line) {
					taggedFiles = append(taggedFiles, path)
					break
				}
			}
		}

		return nil
	})

	return taggedFiles, err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [directory]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  validate <dir>  - Validate build tags in directory\n")
		fmt.Fprintf(os.Stderr, "  list <dir>      - List files with build tags\n")
		os.Exit(1)
	}

	command := os.Args[1]
	var rootDir string
	if len(os.Args) > 2 {
		rootDir = os.Args[2]
	} else {
		rootDir = "."
	}

	switch command {
	case "validate":
		if err := validateBuildTags(rootDir); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}
	case "list":
		files, err := scanForTaggedFiles(rootDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning files: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Files with build tags (%d):\n", len(files))
		for _, file := range files {
			fmt.Printf("  %s\n", file)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}
