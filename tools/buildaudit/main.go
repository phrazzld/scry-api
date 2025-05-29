package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phrazzld/scry-api/tools/buildtags/shared"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
		os.Exit(1)
	}

	rootDir := os.Args[1]
	fileInfos, err := shared.ScanDirectory(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	// Generate report
	fmt.Println("# Build Tag Audit Report")
	fmt.Println()

	// Summary
	fmt.Printf("## Summary\n")
	fmt.Printf("- Total Go files scanned: %d\n", len(fileInfos))
	filesWithTags := 0
	for _, fi := range fileInfos {
		if len(fi.Tags) > 0 {
			filesWithTags++
		}
	}
	fmt.Printf("- Files with build tags: %d\n", filesWithTags)
	fmt.Println()

	// Tag usage statistics
	tagUsage := analyzeTagUsage(fileInfos)
	fmt.Println("## Build Tag Usage")
	fmt.Println()
	fmt.Println("| Tag | Count | Files |")
	fmt.Println("|-----|-------|-------|")
	for _, tu := range tagUsage {
		fileList := strings.Join(tu.Files, ", ")
		if len(fileList) > 50 {
			fileList = fileList[:47] + "..."
		}
		fmt.Printf("| %s | %d | %s |\n", tu.Tag, tu.Count, fileList)
	}
	fmt.Println()

	// Detect conflicts
	conflicts := shared.DetectConflicts(fileInfos)
	if len(conflicts) > 0 {
		fmt.Println("## Potential Conflicts")
		fmt.Println()
		for _, conflict := range conflicts {
			fmt.Printf("### %s\n", conflict.Tag)
			fmt.Printf("- Type: %s\n", conflict.Type)
			fmt.Printf("- Description: %s\n", conflict.Description)
			fmt.Printf("- Files: %s\n", strings.Join(conflict.Files, ", "))
			fmt.Println()
		}
	}

	// CI compatibility warnings
	ciIssues := shared.ValidateCICompatibility(fileInfos)
	if len(ciIssues) > 0 {
		fmt.Println("## CI Compatibility Warnings")
		fmt.Println()
		for _, issue := range ciIssues {
			fmt.Printf("- %s\n", issue)
		}
		fmt.Println()
	}

	// Package-specific analysis
	packageAnalysis := analyzeByPackage(fileInfos)
	fmt.Println("## Package Analysis")
	fmt.Println()
	for pkg, files := range packageAnalysis {
		if len(files) > 1 {
			fmt.Printf("### %s\n", pkg)
			fmt.Printf("Files with different build tags:\n")
			for _, fi := range files {
				if len(fi.Tags) > 0 {
					fmt.Printf("- %s: %s\n", fi.Path, strings.Join(fi.Tags, ", "))
				}
			}
			fmt.Println()
		}
	}
}

func analyzeTagUsage(fileInfos []shared.FileInfo) []shared.TagUsage {
	tagCount := make(map[string]*shared.TagUsage)

	for _, fi := range fileInfos {
		for _, tagExpr := range fi.Tags {
			tags := shared.ParseBuildExpression(tagExpr)
			for _, tag := range tags {
				// Normalize tag (remove negation for counting)
				normalizedTag := strings.TrimPrefix(tag, "!")

				if tu, exists := tagCount[normalizedTag]; exists {
					tu.Count++
					tu.Files = append(tu.Files, filepath.Base(fi.Path))
				} else {
					tagCount[normalizedTag] = &shared.TagUsage{
						Tag:   normalizedTag,
						Count: 1,
						Files: []string{filepath.Base(fi.Path)},
					}
				}
			}
		}
	}

	// Convert to slice and sort
	usage := make([]shared.TagUsage, 0, len(tagCount))
	for _, tu := range tagCount {
		usage = append(usage, *tu)
	}
	sort.Slice(usage, func(i, j int) bool {
		return usage[i].Count > usage[j].Count
	})

	return usage
}

func analyzeByPackage(fileInfos []shared.FileInfo) map[string][]shared.FileInfo {
	packages := make(map[string][]shared.FileInfo)

	for _, fi := range fileInfos {
		if fi.Package != "" {
			packages[fi.Package] = append(packages[fi.Package], fi)
		}
	}

	return packages
}
