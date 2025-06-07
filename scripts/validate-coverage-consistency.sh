#!/bin/bash

# Coverage Validation Script
# This script validates that local coverage calculations match CI expectations
# Usage: ./scripts/validate-coverage-consistency.sh [package1] [package2] ...
# If no packages specified, tests all packages with coverage thresholds

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Coverage thresholds file
COVERAGE_THRESHOLDS="$PROJECT_ROOT/coverage-thresholds.json"

# Temporary directory for test artifacts
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "=========================================="
echo "Coverage Consistency Validation"
echo "=========================================="
echo "Project root: $PROJECT_ROOT"
echo "Go version: $(go version)"
echo "Coverage thresholds: $COVERAGE_THRESHOLDS"
echo "Temp directory: $TEMP_DIR"
echo ""

# Function to run CI-equivalent test command
run_ci_test_command() {
    local package="$1"
    local package_name

    # Sanitize package name for file naming (replace / with -)
    package_name=$(echo "$package" | tr '/' '-')

    echo "Testing package: $package"
    echo "Sanitized name: $package_name"

    # Change to project root
    cd "$PROJECT_ROOT"

    # Run exact CI command
    echo "Running CI-equivalent command..."
    GOTEST_DEBUG=1 GODEBUG=gctrace=0 go test -v -json -race \
        -coverprofile="$TEMP_DIR/coverage-${package_name}.out" \
        -tags=integration,test_without_external_deps \
        "./${package}/..." > "$TEMP_DIR/test-results-${package_name}.json" 2>&1

    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo -e "${RED}❌ Tests failed for $package${NC}"
        echo "Exit code: $exit_code"
        # Show test failures but continue validation
        if [ -f "$TEMP_DIR/test-results-${package_name}.json" ]; then
            echo "Test output available in: $TEMP_DIR/test-results-${package_name}.json"
        fi
        return $exit_code
    fi

    # Extract coverage percentage
    if [ -f "$TEMP_DIR/coverage-${package_name}.out" ]; then
        local coverage_line
        coverage_line=$(go tool cover -func="$TEMP_DIR/coverage-${package_name}.out" | tail -n 1)
        local coverage_percent
        coverage_percent=$(echo "$coverage_line" | awk '{print $NF}' | sed 's/%//')

        echo -e "${GREEN}✅ Tests passed for $package${NC}"
        echo "Coverage: ${coverage_percent}%"

        # Check against threshold
        check_coverage_threshold "$package" "$coverage_percent"

        return 0
    else
        echo -e "${YELLOW}⚠️  No coverage file generated for $package${NC}"
        return 1
    fi
}

# Function to check coverage against thresholds
check_coverage_threshold() {
    local package="$1"
    local coverage="$2"

    if [ ! -f "$COVERAGE_THRESHOLDS" ]; then
        echo -e "${YELLOW}⚠️  Coverage thresholds file not found${NC}"
        return 0
    fi

    # Get threshold for this package
    local threshold
    threshold=$(jq -r ".package_thresholds[\"$package\"] // .default_threshold" "$COVERAGE_THRESHOLDS")

    if [ "$threshold" = "null" ]; then
        threshold=$(jq -r ".default_threshold" "$COVERAGE_THRESHOLDS")
    fi

    echo "Threshold: ${threshold}%"

    # Compare coverage with threshold (using integer comparison)
    local coverage_int=${coverage%.*}  # Remove decimal part
    local threshold_int=${threshold%.*}  # Remove decimal part

    if [ "$coverage_int" -ge "$threshold_int" ]; then
        echo -e "${GREEN}✅ Coverage meets threshold (${coverage}% >= ${threshold}%)${NC}"
    else
        echo -e "${RED}❌ Coverage below threshold (${coverage}% < ${threshold}%)${NC}"
    fi

    echo ""
}

# Function to get packages with coverage thresholds
get_packages_with_thresholds() {
    if [ ! -f "$COVERAGE_THRESHOLDS" ]; then
        echo "Error: Coverage thresholds file not found: $COVERAGE_THRESHOLDS" >&2
        exit 1
    fi

    jq -r '.package_thresholds | keys[]' "$COVERAGE_THRESHOLDS"
}

# Main execution
main() {
    local packages=()

    if [ $# -eq 0 ]; then
        echo "No packages specified, testing all packages with coverage thresholds..."
        echo ""

        # Get all packages from coverage thresholds
        while IFS= read -r package; do
            packages+=("$package")
        done < <(get_packages_with_thresholds)
    else
        packages=("$@")
    fi

    echo "Packages to test: ${packages[*]}"
    echo ""

    local total_packages=${#packages[@]}
    local successful_packages=0
    local failed_packages=0

    for package in "${packages[@]}"; do
        echo "=========================================="
        echo "Testing package: $package"
        echo "=========================================="

        if run_ci_test_command "$package"; then
            ((successful_packages++))
        else
            ((failed_packages++))
        fi

        echo ""
    done

    echo "=========================================="
    echo "Validation Summary"
    echo "=========================================="
    echo "Total packages tested: $total_packages"
    echo -e "Successful: ${GREEN}$successful_packages${NC}"
    echo -e "Failed: ${RED}$failed_packages${NC}"
    echo ""

    # Environment information
    echo "Environment Information:"
    echo "- OS: $(uname -s)"
    echo "- Architecture: $(uname -m)"
    echo "- Go version: $(go version)"
    echo "- Working directory: $(pwd)"
    echo "- User: $(whoami)"
    echo ""

    echo "Validation artifacts available in: $TEMP_DIR"
    echo "- Coverage reports: coverage-*.out"
    echo "- Test results: test-results-*.json"
    echo ""

    if [ $failed_packages -eq 0 ]; then
        echo -e "${GREEN}✅ All coverage validations passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some coverage validations failed.${NC}"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
