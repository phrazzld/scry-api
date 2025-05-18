#!/usr/bin/env bash

# run-ci-checks.sh - Local CI Pipeline Simulation
#
# Purpose: Run key CI pipeline checks locally to catch issues before pushing code
# Usage: ./scripts/run-ci-checks.sh [options]
#
# Options:
#   -h, --help      Show this help message
#   -v, --verbose   Enable verbose output
#   -s, --skip-tests Skip running tests (useful for quick checks)

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script configuration
VERBOSE=false
SKIP_TESTS=false
COVERAGE_THRESHOLD=70

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            grep '^#' "$0" | grep -v '/bin' | cut -c 3-
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -s|--skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Helper functions
print_step() {
    echo -e "\n${GREEN}==> $1${NC}"
}

print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}WARNING: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Ensure we're in the project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

print_step "Starting local CI checks..."

# Track overall status
FAILED_CHECKS=()

# 1. Build verification
print_step "Building main application"
if go build -v ./cmd/server; then
    print_success "Build successful"
else
    print_error "Build failed"
    FAILED_CHECKS+=("build")
fi

# 2. Format check
print_step "Checking code formatting"
if [ -z "$(gofmt -l .)" ]; then
    print_success "Code is properly formatted"
else
    print_warning "Code formatting issues found. Run 'go fmt ./...' to fix."
    if [ "$VERBOSE" = true ]; then
        echo "Files needing formatting:"
        gofmt -l .
    fi
    FAILED_CHECKS+=("formatting")
fi

# 3. Linting
print_step "Running golangci-lint"
if golangci-lint run --verbose --build-tags=test_without_external_deps; then
    print_success "Linting passed"
else
    print_error "Linting failed"
    FAILED_CHECKS+=("linting")
fi

# 4. Testing (unless skipped)
if [ "$SKIP_TESTS" = false ]; then
    print_step "Running tests"
    if go test -v -race -coverprofile=coverage.out -tags=integration ./...; then
        print_success "Tests passed"

        # Check coverage
        print_step "Checking test coverage"
        total_coverage=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}' | sed 's/%//')
        coverage_int=$(echo $total_coverage | cut -d. -f1)

        if [ "$coverage_int" -ge "$COVERAGE_THRESHOLD" ]; then
            print_success "Test coverage: ${total_coverage}% (meets ${COVERAGE_THRESHOLD}% threshold)"
        else
            print_warning "Test coverage: ${total_coverage}% (below ${COVERAGE_THRESHOLD}% threshold)"
            FAILED_CHECKS+=("coverage")
        fi
    else
        print_error "Tests failed"
        FAILED_CHECKS+=("tests")
    fi
else
    print_warning "Tests skipped"
fi

# 5. go mod tidy check
print_step "Checking go.mod tidiness"
cp go.mod go.mod.backup
cp go.sum go.sum.backup
go mod tidy
if diff go.mod go.mod.backup >/dev/null && diff go.sum go.sum.backup >/dev/null; then
    print_success "go.mod is tidy"
else
    print_warning "go.mod is not tidy. Run 'go mod tidy' to fix."
    FAILED_CHECKS+=("go-mod-tidy")
fi
rm go.mod.backup go.sum.backup

# Summary
echo -e "\n${GREEN}==== CI Checks Summary ====${NC}"
if [ ${#FAILED_CHECKS[@]} -eq 0 ]; then
    print_success "All CI checks passed! 🎉"
    exit 0
else
    print_error "The following checks failed:"
    for check in "${FAILED_CHECKS[@]}"; do
        echo "  - $check"
    done
    echo ""
    echo "Please fix these issues before pushing your code."
    exit 1
fi
