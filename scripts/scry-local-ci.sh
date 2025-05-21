#!/usr/bin/env bash
# scry-local-ci.sh - Local CI Pipeline Simulation
#
# Purpose: Comprehensive script to simulate CI checks locally, helping developers
# catch issues before pushing code.
#
# This script runs the same checks as the CI pipeline, including:
# - Pre-flight environment validations
# - Code formatting verification
# - Linting with golangci-lint
# - Build verification
# - Database migration checks (when database is available)
# - Test execution with various configurations
# - Coverage analysis
#
# Usage:
#   ./scripts/scry-local-ci.sh [options]
#
# Options:
#   -h, --help              Show this help message
#   -v, --verbose           Enable detailed output
#   -q, --quick             Run only essential checks (faster)
#   -f, --fix               Auto-fix issues where possible
#   --skip-build            Skip build verification
#   --skip-lint             Skip linting checks
#   --skip-format           Skip formatting checks
#   --skip-tests            Skip all tests
#   --skip-migrations       Skip database migration checks
#   --skip-coverage         Skip code coverage analysis
#   --skip-pre-flight       Skip pre-flight environment checks
#   --with-db               Run database-dependent checks (requires DB connection)
#   --ci-mode               Run as if in CI environment (useful for debugging)

# Enable strict error handling
set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERBOSE=false
QUICK_MODE=false
FIX_MODE=false
CI_MODE=false
WITH_DB=false
START_TIME=$(date +%s)
COVERAGE_THRESHOLD=70

# Tracks which checks to run (default: all)
RUN_BUILD=true
RUN_LINT=true
RUN_FORMAT=true
RUN_TESTS=true
RUN_MIGRATIONS=true
RUN_COVERAGE=true
RUN_PRE_FLIGHT=true

# Track check results
FAILED_CHECKS=()
PASSED_CHECKS=()
SKIPPED_CHECKS=()

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

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
        -q|--quick)
            QUICK_MODE=true
            shift
            ;;
        -f|--fix)
            FIX_MODE=true
            shift
            ;;
        --ci-mode)
            CI_MODE=true
            shift
            ;;
        --with-db)
            WITH_DB=true
            shift
            ;;
        --skip-build)
            RUN_BUILD=false
            shift
            ;;
        --skip-lint)
            RUN_LINT=false
            shift
            ;;
        --skip-format)
            RUN_FORMAT=false
            shift
            ;;
        --skip-tests)
            RUN_TESTS=false
            shift
            ;;
        --skip-migrations)
            RUN_MIGRATIONS=false
            shift
            ;;
        --skip-coverage)
            RUN_COVERAGE=false
            shift
            ;;
        --skip-pre-flight)
            RUN_PRE_FLIGHT=false
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Helper functions for output
print_step() {
    echo -e "\n${BLUE}==> ${BOLD}$1${NC}"
}

print_substep() {
    echo -e "    ${BLUE}-> $1${NC}"
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

print_info() {
    if [ "$VERBOSE" = true ]; then
        echo -e "    $1"
    fi
}

log_skipped() {
    local check_name="$1"
    echo -e "${YELLOW}SKIPPED: $check_name${NC}"
    SKIPPED_CHECKS+=("$check_name")
}

log_failed() {
    local check_name="$1"
    echo -e "${RED}FAILED: $check_name${NC}" >&2
    FAILED_CHECKS+=("$check_name")
}

log_passed() {
    local check_name="$1"
    echo -e "${GREEN}PASSED: $check_name${NC}"
    PASSED_CHECKS+=("$check_name")
}

run_check() {
    local check_name="$1"
    local check_flag="$2"
    local check_func="$3"

    if [ "$check_flag" = false ]; then
        log_skipped "$check_name"
        return 0
    fi

    print_step "Running $check_name"
    local start_time=$(date +%s)

    if $check_func; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_passed "$check_name (${duration}s)"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_failed "$check_name (${duration}s)"
        return 1
    fi
}

# Check 1: Pre-flight - verify environment and dependencies
check_pre_flight() {
    print_substep "Verifying development environment"

    # Check Go installation
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        print_info "Visit https://golang.org/doc/install for installation instructions"
        return 1
    else
        local go_version=$(go version | awk '{print $3}')
        print_info "Found Go $go_version"
    fi

    # Check golangci-lint
    if ! command -v golangci-lint &> /dev/null; then
        print_error "golangci-lint is not installed or not in PATH"
        print_info "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        return 1
    else
        local lint_version=$(golangci-lint --version | head -n 1)
        print_info "Found $lint_version"
    fi

    # Check for essential environment variables
    if [ "$WITH_DB" = true ] || [ "$CI_MODE" = true ]; then
        print_substep "Checking database configuration"

        # Set DB environment variables for CI mode
        if [ "$CI_MODE" = true ]; then
            export DATABASE_URL="postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
            export SCRY_TEST_DB_URL="$DATABASE_URL"
            export SCRY_DATABASE_URL="$DATABASE_URL"
            print_info "CI mode: Using standard test database URL"
        fi

        # Check database URL
        if [ -z "${DATABASE_URL:-}" ] && [ -z "${SCRY_TEST_DB_URL:-}" ] && [ -z "${SCRY_DATABASE_URL:-}" ]; then
            print_warning "No database URL environment variable is set"
            print_info "Set DATABASE_URL, SCRY_TEST_DB_URL, or SCRY_DATABASE_URL for database checks"

            if [ "$CI_MODE" = true ]; then
                return 1
            fi
        else
            print_info "Database URL environment variable is set"
        fi

        # Check PostgreSQL client if running with DB
        if [ "$WITH_DB" = true ]; then
            if ! command -v psql &> /dev/null; then
                print_warning "PostgreSQL client (psql) not installed - some DB checks may fail"
                print_info "Install PostgreSQL client tools for your platform"
            else
                print_info "Found PostgreSQL client"
            fi
        fi
    fi

    # Check for go.mod (validate we're in a Go module)
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        print_error "go.mod not found! Are you in the project root?"
        return 1
    fi

    # Check required config files
    if [ ! -f "$PROJECT_ROOT/.golangci.yml" ]; then
        print_warning "golangci-lint config (.golangci.yml) not found"
    fi

    print_success "Pre-flight checks completed"
    return 0
}

# Check 2: Format check - verify code formatting
check_format() {
    print_substep "Checking code formatting"

    # Create a temporary file for gofmt output
    local gofmt_output_file=$(mktemp)
    trap 'rm -f "$gofmt_output_file"' EXIT

    # Run gofmt to find formatting issues
    gofmt -l . > "$gofmt_output_file"

    # Check if any files need formatting
    if [ -s "$gofmt_output_file" ]; then
        print_error "Code formatting issues found"
        echo "Files needing formatting:"
        cat "$gofmt_output_file"

        # Attempt to fix if in fix mode
        if [ "$FIX_MODE" = true ]; then
            print_substep "Fixing formatting issues with gofmt -w"
            xargs gofmt -w < "$gofmt_output_file"
            print_success "Formatting fixed"
            return 0
        fi

        print_info "Run 'go fmt ./...' to fix formatting issues"
        return 1
    fi

    print_success "Code is properly formatted"
    return 0
}

# Check 3: Linting - check code with golangci-lint
check_lint() {
    print_substep "Running golangci-lint"

    local lint_args="run"

    # Add verbosity if requested
    if [ "$VERBOSE" = true ]; then
        lint_args="$lint_args --verbose"
    fi

    # Add fix flag if requested
    if [ "$FIX_MODE" = true ]; then
        lint_args="$lint_args --fix"
    fi

    # Add build tags for test compatibility
    lint_args="$lint_args --build-tags=test_without_external_deps"

    # Run linter
    if golangci-lint $lint_args; then
        print_success "Linting passed"
        return 0
    else
        print_error "Linting failed"
        print_info "Review the issues above and fix them"
        return 1
    fi
}

# Check 4: Build verification - ensure the code builds
check_build() {
    print_substep "Building main application"

    if go build -v ./cmd/server; then
        print_success "Build successful"
        return 0
    else
        print_error "Build failed"
        return 1
    fi
}

# Check 5: Database migration checks
check_migrations() {
    if [ "$WITH_DB" = false ]; then
        print_warning "Skipping migration checks (--with-db not specified)"
        return 0
    fi

    print_substep "Checking database migrations"

    # Minimal required environment variables
    export SCRY_AUTH_JWT_SECRET="${SCRY_AUTH_JWT_SECRET:-test-secret-for-local-ci}"
    export SCRY_LLM_GEMINI_API_KEY="${SCRY_LLM_GEMINI_API_KEY:-test-key-for-local-ci}"
    export SCRY_LLM_PROMPT_TEMPLATE_PATH="${SCRY_LLM_PROMPT_TEMPLATE_PATH:-prompts/flashcard_template.txt}"

    # Try to connect to the database
    print_info "Testing database connection"
    if ! "${SCRIPT_DIR}/wait-for-db.sh" --timeout 10; then
        print_error "Failed to connect to database"
        print_info "Check your database URL and that PostgreSQL is running"
        return 1
    fi

    # Run migration status check
    print_info "Checking migration status"
    if go run ./cmd/server -migrate=status; then
        print_success "Migration status check passed"
    else
        print_error "Migration status check failed"
        return 1
    fi

    # Run migration validation if in non-quick mode
    if [ "$QUICK_MODE" = false ]; then
        print_info "Validating migrations"
        if go run ./cmd/server -validate-migrations; then
            print_success "Migration validation passed"
        else
            print_error "Migration validation failed"
            return 1
        fi
    else
        print_info "Skipping detailed migration validation in quick mode"
    fi

    return 0
}

# Check 6: Tests - run unit and integration tests
check_tests() {
    print_substep "Running tests"

    # Set up test arguments
    local test_args="-v"

    # Add race detection if not in quick mode
    if [ "$QUICK_MODE" = false ]; then
        test_args="$test_args -race"
    fi

    # Add coverage profile if running coverage check
    if [ "$RUN_COVERAGE" = true ]; then
        test_args="$test_args -coverprofile=coverage.out"
    fi

    # Add build tags
    local build_tags="test_without_external_deps"
    if [ "$WITH_DB" = true ]; then
        build_tags="$build_tags,integration"
    fi
    test_args="$test_args -tags=$build_tags"

    # Set CGO_ENABLED for database tests if using DB
    if [ "$WITH_DB" = true ]; then
        export CGO_ENABLED=1
    fi

    # Run tests
    if go test $test_args ./...; then
        print_success "All tests passed"
        return 0
    else
        print_error "Tests failed"
        return 1
    fi
}

# Check 7: Coverage analysis
check_coverage() {
    if [ "$RUN_TESTS" = false ]; then
        print_warning "Skipping coverage check (tests were skipped)"
        return 0
    fi

    if [ ! -f "coverage.out" ]; then
        print_warning "No coverage file found. Did tests run with -coverprofile?"
        return 0
    fi

    print_substep "Analyzing test coverage"

    # Get overall coverage
    local total_coverage=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}' | sed 's/%//')
    local coverage_int=$(echo $total_coverage | cut -d. -f1)

    echo "Total test coverage: ${total_coverage}%"

    # Check if coverage threshold is met
    if [ "$coverage_int" -ge "$COVERAGE_THRESHOLD" ]; then
        print_success "Coverage threshold met (${total_coverage}% >= ${COVERAGE_THRESHOLD}%)"
        return 0
    else
        print_error "Coverage below threshold (${total_coverage}% < ${COVERAGE_THRESHOLD}%)"
        print_info "Add more tests to increase coverage"
        return 1
    fi
}

# Function to run go mod tidy check
check_go_mod_tidy() {
    print_substep "Checking go.mod tidiness"

    # Create backup files
    cp go.mod go.mod.backup
    cp go.sum go.sum.backup

    # Run go mod tidy
    go mod tidy

    # Compare with backups
    if diff go.mod go.mod.backup >/dev/null && diff go.sum go.sum.backup >/dev/null; then
        print_success "go.mod is tidy"
        rm go.mod.backup go.sum.backup
        return 0
    else
        print_warning "go.mod is not tidy. Run 'go mod tidy' to fix."
        # Restore backups
        mv go.mod.backup go.mod
        mv go.sum.backup go.sum
        return 1
    fi
}

# Main execution function
run_ci_checks() {
    local error_count=0
    local start_time=$(date +%s)

    # Change to project root
    cd "$PROJECT_ROOT"

    print_step "Starting local CI checks"
    echo "Mode: $([ "$QUICK_MODE" = true ] && echo "Quick" || echo "Full")"
    echo "Fix mode: $([ "$FIX_MODE" = true ] && echo "Enabled" || echo "Disabled")"
    echo "Database checks: $([ "$WITH_DB" = true ] && echo "Enabled" || echo "Disabled")"

    # Run checks in sequence, tracking failures

    # Pre-flight checks
    run_check "Pre-flight Checks" "$RUN_PRE_FLIGHT" check_pre_flight || error_count=$((error_count + 1))

    # Format checks
    run_check "Format Check" "$RUN_FORMAT" check_format || error_count=$((error_count + 1))

    # Linting
    run_check "Linting" "$RUN_LINT" check_lint || error_count=$((error_count + 1))

    # go.mod tidy check
    run_check "go mod tidy Check" true check_go_mod_tidy || error_count=$((error_count + 1))

    # Build verification
    run_check "Build Verification" "$RUN_BUILD" check_build || error_count=$((error_count + 1))

    # Migration checks
    run_check "Migration Checks" "$RUN_MIGRATIONS" check_migrations || error_count=$((error_count + 1))

    # Tests
    run_check "Tests" "$RUN_TESTS" check_tests || error_count=$((error_count + 1))

    # Coverage analysis
    run_check "Coverage Analysis" "$RUN_COVERAGE" check_coverage || error_count=$((error_count + 1))

    # Calculate total execution time
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))

    # Print summary
    print_step "CI Checks Summary"
    echo "Total execution time: ${total_duration}s"
    echo ""
    echo "Passed: ${#PASSED_CHECKS[@]}"
    echo "Failed: ${#FAILED_CHECKS[@]}"
    echo "Skipped: ${#SKIPPED_CHECKS[@]}"
    echo ""

    # Print failed checks if any
    if [ ${#FAILED_CHECKS[@]} -gt 0 ]; then
        echo -e "${RED}Failed Checks:${NC}"
        for check in "${FAILED_CHECKS[@]}"; do
            echo " - $check"
        done
        echo ""
    fi

    # Final verdict
    if [ "$error_count" -eq 0 ]; then
        print_success "All CI checks passed! 🎉"
        return 0
    else
        print_error "$error_count check(s) failed"
        echo "Please fix these issues before pushing your code."
        return 1
    fi
}

# Execute all checks
run_ci_checks
exit $?
