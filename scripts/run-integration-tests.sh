#!/bin/bash

# Run integration tests with a local PostgreSQL instance
# This script starts a PostgreSQL container using docker-compose,
# runs migrations, executes the tests, then cleans up

set -e

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "Error: docker-compose is not installed"
    exit 1
fi

# Load .env if it exists
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Set DATABASE_URL if not already set
if [ -z "$DATABASE_URL" ]; then
    export DATABASE_URL="postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
    echo "Using default DATABASE_URL: $DATABASE_URL"
fi

# Set SCRY_TEST_DB_URL to the same value for compatibility
export SCRY_TEST_DB_URL="$DATABASE_URL"

# Start PostgreSQL using docker-compose
echo "Starting PostgreSQL container..."
docker-compose up -d postgres

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
until docker-compose exec -T postgres pg_isready -U postgres; do
    echo "PostgreSQL is not ready yet..."
    sleep 1
done

echo "PostgreSQL is ready!"

# Run migrations
echo "Running database migrations..."
go run cmd/server/main.go -migrate=up

# Run tests with integration tag
echo "Running integration tests..."
go test -v -race -tags=integration ./...

# Get the exit code of the tests
TEST_EXIT_CODE=$?

# Cleanup (optional - comment out to leave container running)
echo "Cleaning up containers..."
docker-compose down

# Exit with the test exit code
exit $TEST_EXIT_CODE
