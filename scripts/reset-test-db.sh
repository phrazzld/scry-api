#!/bin/bash
#
# reset-test-db.sh - Reset the test database for CI
#
# This script drops all tables in the test database to ensure a clean state
# before running migrations. It's designed to be used in CI environments to
# prevent "relation already exists" errors during migration.
#
# Usage:
#   ./scripts/reset-test-db.sh [DATABASE_URL]
#
# If DATABASE_URL is not provided, it will use the environment variable
# DATABASE_URL, SCRY_TEST_DB_URL, or SCRY_DATABASE_URL in that order.

set -e

# Default values
# This should match the MigrationTableName constant in internal/testdb/migration_helpers.go
SCHEMA_MIGRATIONS_TABLE="schema_migrations"
POSTGRES_SCHEMA="public"

# Get database URL from arguments or environment
if [ -n "$1" ]; then
  DB_URL="$1"
else
  # Try environment variables in priority order
  if [ -n "$DATABASE_URL" ]; then
    DB_URL="$DATABASE_URL"
  elif [ -n "$SCRY_TEST_DB_URL" ]; then
    DB_URL="$SCRY_TEST_DB_URL"
  elif [ -n "$SCRY_DATABASE_URL" ]; then
    DB_URL="$SCRY_DATABASE_URL"
  else
    echo "ERROR: No database URL provided"
    echo "Please provide a database URL as an argument or set DATABASE_URL, SCRY_TEST_DB_URL, or SCRY_DATABASE_URL"
    exit 1
  fi
fi

# Extract database name from connection string
DB_NAME=$(echo "$DB_URL" | sed -E 's/.*\/([^?]*).*/\1/')

echo "Resetting database: $DB_NAME (tables in schema $POSTGRES_SCHEMA)"

# Build PSQL command with connection parameters from URL
# Convert URL format to psql arguments
if [[ "$DB_URL" =~ postgres://([^:]+):([^@]+)@([^:]+):([^/]+)/([^?]+) ]]; then
  PGUSER="${BASH_REMATCH[1]}"
  PGPASSWORD="${BASH_REMATCH[2]}"
  PGHOST="${BASH_REMATCH[3]}"
  PGPORT="${BASH_REMATCH[4]}"
  PGDATABASE="${BASH_REMATCH[5]}"
  export PGUSER PGPASSWORD PGHOST PGPORT PGDATABASE
else
  echo "ERROR: Invalid database URL format"
  echo "Expected format: postgres://username:password@hostname:port/database"
  exit 1
fi

echo "Connecting to database server: $PGHOST:$PGPORT as $PGUSER"

# Check connection
if ! psql -c "SELECT 1" >/dev/null 2>&1; then
  echo "ERROR: Failed to connect to database server"
  exit 1
fi

echo "Connected successfully to database: $PGDATABASE"

# First, drop all custom types (ENUMs) in the schema
echo "Dropping all custom types in schema $POSTGRES_SCHEMA..."
psql -c "DO \$\$
DECLARE
    type_name text;
BEGIN
    FOR type_name IN
        SELECT t.typname
        FROM pg_type t
        JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
        WHERE n.nspname = 'public'
        AND t.typtype = 'e'  -- 'e' for enum types
    LOOP
        EXECUTE 'DROP TYPE IF EXISTS ' || type_name || ' CASCADE';
        RAISE NOTICE 'Dropped type: %', type_name;
    END LOOP;
END \$\$;"

# Now drop all tables including the migration table without destroying the database
echo "Dropping all tables in schema $POSTGRES_SCHEMA..."

# Get a list of all tables to drop
TABLES=$(psql -t -c "SELECT tablename FROM pg_tables WHERE schemaname = '$POSTGRES_SCHEMA' AND tablename != 'spatial_ref_sys';")

# Check if any tables exist
if [ -z "$TABLES" ]; then
  echo "No tables found in schema $POSTGRES_SCHEMA. Database is already clean."
else
  # Generate drop statements for all tables
  DROP_STATEMENTS=""
  for TABLE in $TABLES; do
    TABLE=$(echo "$TABLE" | tr -d '[:space:]')
    DROP_STATEMENTS+="DROP TABLE IF EXISTS \"$TABLE\" CASCADE; "
  done

  # Execute the drop statements in a single transaction
  echo "Executing: $DROP_STATEMENTS"
  if psql -c "BEGIN; $DROP_STATEMENTS COMMIT;"; then
    echo "All tables dropped successfully"
  else
    echo "ERROR: Failed to drop tables"
    exit 1
  fi
fi

echo "Database reset completed successfully"
exit 0
