#!/bin/bash
set -e

# This script tests the database migrations against the provisioned database
# Usage: DATABASE_URL="postgres://user:pass@host:port/dbname" ./test-migrations.sh

if [ -z "$DATABASE_URL" ]; then
  echo "ERROR: DATABASE_URL environment variable must be set"
  exit 1
fi

echo "Testing migrations against: ${DATABASE_URL/\/\/[^:]*:[^@]*@/\/\/*****:*****@}"

# Navigate to the project root
cd "$(dirname "$0")/../.."

# Run migrations
echo "Running migrations up..."
go run cmd/server/main.go -migrate=up

# Check migration status
echo "Checking migration status..."
go run cmd/server/main.go -migrate=status

echo "Migration test completed successfully!"
