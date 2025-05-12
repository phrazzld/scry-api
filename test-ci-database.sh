#!/bin/bash
# test-ci-database.sh - Script to test database configuration and migration in a CI-like environment
# This script simulates the CI environment to verify our fixes work correctly

set -e

# Set CI environment variables to simulate CI environment
export CI=true
export GITHUB_ACTIONS=true
export GITHUB_WORKSPACE=$(pwd)

# Define color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}==== Scry API CI Database Test ====${NC}"
echo -e "${BLUE}This script tests database configuration and migration in a CI-like environment${NC}"
echo -e "${BLUE}Current directory: $(pwd)${NC}"

# Step 1: Test database URL handling
echo -e "\n${YELLOW}Step 1: Testing database URL handling${NC}"

# Test with different usernames to ensure auto-correction works
URL_VARIATIONS=(
  "postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
  "postgres://root:postgres@localhost:5432/scry_test?sslmode=disable"
  "postgres://scryapiuser:postgres@localhost:5432/scry_test?sslmode=disable"
)

echo -e "${YELLOW}Testing multiple DATABASE_URL variations${NC}"
for url in "${URL_VARIATIONS[@]}"; do
  echo -e "\n${YELLOW}Testing with DATABASE_URL=$url${NC}"

  # Clear any previous environment variables
  unset DATABASE_URL SCRY_TEST_DB_URL SCRY_DATABASE_URL

  # Set the test URL
  export DATABASE_URL=$url

  # Run the wait-for-db.sh script
  echo -e "${YELLOW}Running wait-for-db.sh${NC}"
  if ! ./scripts/wait-for-db.sh --attempts 3 --timeout 10; then
    echo -e "${RED}Error: wait-for-db.sh failed${NC}"
    exit 1
  fi

  # Check that DATABASE_URL was properly standardized
  if [[ "$DATABASE_URL" != *"postgres:postgres"* ]]; then
    echo -e "${RED}Error: DATABASE_URL was not standardized correctly: $DATABASE_URL${NC}"
    exit 1
  else
    echo -e "${GREEN}DATABASE_URL standardized successfully: $DATABASE_URL${NC}"
  fi
done

# Step 2: Test with inconsistent environment variables
echo -e "\n${YELLOW}Step 2: Testing with inconsistent environment variables${NC}"
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
export SCRY_TEST_DB_URL="postgres://root:postgres@localhost:5432/scry_test?sslmode=disable"
export SCRY_DATABASE_URL="postgres://scryapiuser:postgres@localhost:5432/scry_test?sslmode=disable"

echo -e "${YELLOW}Initial environment:${NC}"
echo "DATABASE_URL=$DATABASE_URL"
echo "SCRY_TEST_DB_URL=$SCRY_TEST_DB_URL"
echo "SCRY_DATABASE_URL=$SCRY_DATABASE_URL"

# Run the reset-test-db.sh script
echo -e "${YELLOW}Running reset-test-db.sh${NC}"
if ! ./scripts/reset-test-db.sh; then
  echo -e "${RED}Error: reset-test-db.sh failed${NC}"
  exit 1
fi

# Check that all URLs were standardized
echo -e "${YELLOW}Final environment:${NC}"
echo "DATABASE_URL=$DATABASE_URL"
echo "SCRY_TEST_DB_URL=$SCRY_TEST_DB_URL"
echo "SCRY_DATABASE_URL=$SCRY_DATABASE_URL"

if [[ "$DATABASE_URL" != "$SCRY_TEST_DB_URL" || "$DATABASE_URL" != "$SCRY_DATABASE_URL" ]]; then
  echo -e "${RED}Error: Environment variables were not standardized correctly${NC}"
  exit 1
else
  echo -e "${GREEN}All database URLs standardized successfully${NC}"
fi

# Step 3: Test migration verification
echo -e "\n${YELLOW}Step 3: Testing migration verification${NC}"
echo -e "${YELLOW}Running migration verification${NC}"
if ! go run ./cmd/server/main.go -migrate=up -verbose; then
  echo -e "${RED}Error: Migration verification failed${NC}"
  exit 1
fi

echo -e "\n${GREEN}All tests passed successfully!${NC}"
echo -e "${GREEN}Database configuration standardization and migration validation are working correctly.${NC}"
