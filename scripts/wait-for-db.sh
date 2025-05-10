#!/bin/bash
# wait-for-db.sh - Script to verify database connectivity before running tests
# Especially useful in CI environments to ensure the database is ready

set -e

# Default values
MAX_ATTEMPTS=30
SLEEP_TIME=2
DB_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable}"
TIMEOUT=60

# Help message
function show_help {
  echo "Usage: $0 [options]"
  echo "Wait for PostgreSQL database to be ready"
  echo ""
  echo "Options:"
  echo "  -u, --url URL       Database URL (default: \$DATABASE_URL or postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable)"
  echo "  -a, --attempts N    Maximum number of connection attempts (default: 30)"
  echo "  -s, --sleep N       Seconds to sleep between attempts (default: 2)"
  echo "  -t, --timeout N     Total timeout in seconds (default: 60)"
  echo "  -h, --help          Show this help message"
  echo ""
  echo "Environment variables:"
  echo "  DATABASE_URL        Database URL (used if --url not specified)"
  echo "  SCRY_TEST_DB_URL    Alternate database URL (used if DATABASE_URL not set)"
  echo "  SCRY_DATABASE_URL   Alternate database URL (used if others not set)"
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -u|--url)
      DB_URL="$2"
      shift 2
      ;;
    -a|--attempts)
      MAX_ATTEMPTS="$2"
      shift 2
      ;;
    -s|--sleep)
      SLEEP_TIME="$2"
      shift 2
      ;;
    -t|--timeout)
      TIMEOUT="$2"
      shift 2
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      show_help
      exit 1
      ;;
  esac
done

# Check if psql is available
if ! command -v psql &> /dev/null; then
  echo "Error: psql command not found. Please install PostgreSQL client."
  exit 1
fi

# If DB_URL not set, try alternate environment variables
if [ -z "$DB_URL" ]; then
  if [ -n "$SCRY_TEST_DB_URL" ]; then
    DB_URL="$SCRY_TEST_DB_URL"
  elif [ -n "$SCRY_DATABASE_URL" ]; then
    DB_URL="$SCRY_DATABASE_URL"
  else
    echo "Error: No database URL specified. Please set DATABASE_URL environment variable or use --url option."
    exit 1
  fi
fi

# Extract connection parameters from URL
# Format: postgres://username:password@hostname:port/database?parameters
if [[ "$DB_URL" =~ postgres://([^:]+):([^@]+)@([^:]+):([0-9]+)/([^?]+) ]]; then
  DB_USER="${BASH_REMATCH[1]}"
  DB_PASS="${BASH_REMATCH[2]}"
  DB_HOST="${BASH_REMATCH[3]}"
  DB_PORT="${BASH_REMATCH[4]}"
  DB_NAME="${BASH_REMATCH[5]}"
else
  echo "Error: Invalid database URL format. Expected postgres://user:password@host:port/dbname"
  exit 1
fi

# Set PGPASSWORD environment variable for psql
export PGPASSWORD="$DB_PASS"

echo "Waiting for PostgreSQL to be ready at $DB_HOST:$DB_PORT..."
echo "Database: $DB_NAME, User: $DB_USER"
echo "Will try up to $MAX_ATTEMPTS times with ${SLEEP_TIME}s interval (timeout: ${TIMEOUT}s)"

START_TIME=$(date +%s)
for i in $(seq 1 $MAX_ATTEMPTS); do
  # Check if we've exceeded the timeout
  CURRENT_TIME=$(date +%s)
  ELAPSED=$((CURRENT_TIME - START_TIME))
  if [ $ELAPSED -gt $TIMEOUT ]; then
    echo "Error: Timeout of ${TIMEOUT}s exceeded while waiting for database."
    exit 1
  fi

  # First test using pg_isready
  if pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t 1 &> /dev/null; then
    # Then test an actual query
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" &> /dev/null; then
      echo "Database is ready! (Attempt $i/$MAX_ATTEMPTS, elapsed time: ${ELAPSED}s)"

      # Additional diagnostic info in CI
      if [ -n "$CI" ]; then
        echo "--- PostgreSQL Version ---"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();"

        echo "--- Connection Info ---"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\conninfo"
      fi

      exit 0
    fi
  fi

  echo "Waiting for PostgreSQL to be ready... (Attempt $i/$MAX_ATTEMPTS, elapsed time: ${ELAPSED}s)"
  sleep $SLEEP_TIME
done

echo "Error: Failed to connect to PostgreSQL after $MAX_ATTEMPTS attempts."
echo "Please check the database connection details and ensure PostgreSQL is running."

# Additional diagnostics in CI
if [ -n "$CI" ]; then
  echo "--- Environment Variables ---"
  echo "DATABASE_URL: ${DATABASE_URL:-not set}"
  echo "SCRY_TEST_DB_URL: ${SCRY_TEST_DB_URL:-not set}"
  echo "SCRY_DATABASE_URL: ${SCRY_DATABASE_URL:-not set}"

  echo "--- Network Diagnostics ---"
  echo "Hostname resolution for $DB_HOST:"
  getent hosts "$DB_HOST" || echo "Failed to resolve $DB_HOST"

  echo "Port check for $DB_HOST:$DB_PORT:"
  nc -zv "$DB_HOST" "$DB_PORT" || echo "Failed to connect to $DB_HOST:$DB_PORT"
fi

exit 1
