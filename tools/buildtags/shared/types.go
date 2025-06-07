package shared

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FileInfo holds information about a Go file and its build tags
type FileInfo struct {
	Path         string
	Tags         []string
	Package      string
	FunctionDefs []string
}

// TagUsage tracks usage of a specific build tag
type TagUsage struct {
	Tag   string
	Files []string
	Count int
}

// Conflict represents a build tag conflict
type Conflict struct {
	Tag         string
	Type        string // "negation", "overlap", "redeclaration"
	Files       []string
	Description string
}

var (
	OldBuildTagRe = regexp.MustCompile(`^// \+build\s+(.+)`)
	NewBuildTagRe = regexp.MustCompile(`^//go:build\s+(.+)`)
	TagTokenRe    = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
)

// ScanDirectory scans a directory for Go files and analyzes their build tags
func ScanDirectory(root string) ([]FileInfo, error) {
	var fileInfos []FileInfo

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and .git directories
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == ".git") {
			return filepath.SkipDir
		}

		// Process only .go files
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			info, err := AnalyzeFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: error analyzing %s: %v\n", path, err)
				return nil
			}
			if info != nil {
				fileInfos = append(fileInfos, *info)
			}
		}

		return nil
	})

	return fileInfos, err
}

// AnalyzeFile analyzes a single Go file for build tags and exported functions
func AnalyzeFile(path string) (*FileInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := &FileInfo{
		Path: path,
		Tags: ExtractBuildTags(string(content)),
	}

	// Parse the file to get package and function definitions
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		// If parsing fails, at least return build tag info
		return info, nil
	}

	info.Package = node.Name.Name

	// Extract function definitions
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.IsExported() {
			info.FunctionDefs = append(info.FunctionDefs, fn.Name.Name)
		}
		return true
	})

	return info, nil
}

// ExtractBuildTags extracts build tags from Go file content
func ExtractBuildTags(content string) []string {
	var tags []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()

		// Stop at package declaration
		if strings.HasPrefix(line, "package ") {
			break
		}

		// Check for old-style build tags
		if matches := OldBuildTagRe.FindStringSubmatch(line); matches != nil {
			tags = append(tags, matches[1])
		}

		// Check for new-style build tags
		if matches := NewBuildTagRe.FindStringSubmatch(line); matches != nil {
			tags = append(tags, matches[1])
		}
	}

	return tags
}

// ParseBuildExpression parses build tag expressions to extract individual tags
func ParseBuildExpression(expr string) []string {
	// Extract all tag tokens from the expression
	matches := TagTokenRe.FindAllString(expr, -1)

	// Deduplicate and handle negations
	tagMap := make(map[string]bool)
	for _, match := range matches {
		// Skip operators
		if match == "build" || match == "ignore" {
			continue
		}

		// Check if this is part of a negation
		negIndex := strings.LastIndex(expr[:strings.Index(expr, match)], "!")
		if negIndex != -1 && strings.Index(expr[negIndex:], match) < 10 {
			tagMap["!"+match] = true
		} else {
			tagMap[match] = true
		}
	}

	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// DetectConflicts analyzes build tags for potential conflicts
func DetectConflicts(fileInfos []FileInfo) []Conflict {
	var conflicts []Conflict

	// Check for negation conflicts
	tagPresence := make(map[string][]string) // tag -> files with tag
	tagNegation := make(map[string][]string) // tag -> files with !tag

	for _, fi := range fileInfos {
		for _, tagExpr := range fi.Tags {
			tags := ParseBuildExpression(tagExpr)
			for _, tag := range tags {
				if strings.HasPrefix(tag, "!") {
					normalizedTag := strings.TrimPrefix(tag, "!")
					tagNegation[normalizedTag] = append(tagNegation[normalizedTag], fi.Path)
				} else {
					tagPresence[tag] = append(tagPresence[tag], fi.Path)
				}
			}
		}
	}

	// Find conflicts
	for tag, posFiles := range tagPresence {
		if negFiles, hasNeg := tagNegation[tag]; hasNeg {
			allFiles := append(posFiles, negFiles...)
			conflicts = append(conflicts, Conflict{
				Tag:         tag,
				Type:        "negation",
				Files:       allFiles,
				Description: fmt.Sprintf("Tag '%s' is both included and excluded", tag),
			})
		}
	}

	return conflicts
}

// ValidateCICompatibility checks for CI compatibility issues
func ValidateCICompatibility(fileInfos []FileInfo) []string {
	var issues []string

	for _, fi := range fileInfos {
		// Check if file defines exported functions
		if len(fi.FunctionDefs) > 0 && len(fi.Tags) > 0 {
			// Check if tags include CI-compatible options
			hasCIFallback := false
			for _, tagExpr := range fi.Tags {
				if strings.Contains(tagExpr, "test_without_external_deps") ||
					strings.Contains(tagExpr, "||") {
					hasCIFallback = true
					break
				}
			}

			if !hasCIFallback && !strings.Contains(fi.Path, "_test.go") {
				issues = append(issues, fmt.Sprintf(
					"File %s defines functions but lacks CI-compatible tags",
					fi.Path,
				))
			}
		}

		// Check for overly complex build expressions
		for _, tagExpr := range fi.Tags {
			complexity := strings.Count(tagExpr, "&&") + strings.Count(tagExpr, "||")
			if complexity > 2 {
				issues = append(issues, fmt.Sprintf(
					"File %s has overly complex build tags: %s",
					fi.Path, tagExpr,
				))
			}
		}
	}

	return issues
}
