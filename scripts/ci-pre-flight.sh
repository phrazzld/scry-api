#!/bin/bash
set -euo pipefail

# CI Pre-flight Checks
# This script performs early validation of CI environment setup before main tests run.
# It checks critical environment variables, database connectivity, and project root detection.

echo "Starting CI pre-flight checks..."

# Function to display error messages and exit
error_exit() {
    echo "ERROR: $1" >&2
    exit 1
}

# Function to check if an environment variable is set
check_env_var() {
    local var_name="$1"
    local var_value="${!var_name:-}"

    if [ -z "$var_value" ]; then
        error_exit "Environment variable $var_name is not set"
    else
        echo "✅ $var_name is set"
    fi
}

# Check critical environment variables
echo "Checking critical environment variables..."

# Database configuration
check_env_var "DATABASE_URL"

# Check if we're running in CI
if [ -n "${CI:-}" ]; then
    echo "Running in CI environment"

    # Additional CI-specific environment variables
    check_env_var "CI"
    check_env_var "GITHUB_WORKFLOW"
    check_env_var "GITHUB_WORKSPACE"
else
    echo "Not running in CI environment - some checks will be skipped"
fi

# Verify project root detection
echo "Verifying project root detection..."
if [ ! -f "go.mod" ]; then
    error_exit "Cannot find go.mod file. Script must be run from the project root directory."
else
    PROJECT_ROOT=$(pwd)
    echo "✅ Project root detected at: $PROJECT_ROOT"
fi

# Verify database connectivity
echo "Verifying database connectivity..."

# Check if we can skip database checks when not in CI
if [ -z "${CI:-}" ] && [ -z "${FORCE_DB_CHECK:-}" ]; then
    echo "⚠️ Not in CI environment - skipping database connectivity check"
    echo "   Set FORCE_DB_CHECK=1 to force database checks"
else
    # Check if psql is installed
    if ! command -v psql &> /dev/null; then
        if [ -n "${CI:-}" ]; then
            error_exit "PostgreSQL client (psql) not installed in CI environment"
        else
            echo "⚠️ PostgreSQL client (psql) not installed - skipping database connectivity check"
        fi
    else
        # Use the existing script for database connectivity
        if ! ./scripts/wait-for-db.sh; then
            error_exit "Database connectivity check failed"
        else
            echo "✅ Database connectivity verified"
        fi
    fi
fi

# All checks passed
echo "✅ All pre-flight checks passed successfully"
exit 0
