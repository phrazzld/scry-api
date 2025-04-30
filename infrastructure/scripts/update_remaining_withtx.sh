#!/bin/bash

# Get project root directory
PROJECT_ROOT=$(git rev-parse --show-toplevel)
if [ $? -ne 0 ]; then
  echo "ERROR: Failed to determine project root directory. Make sure you're in a git repository."
  exit 1
fi

# Update cmd/server/main_integration_test.go
file_path="$PROJECT_ROOT/cmd/server/main_integration_test.go"
if [ -f "$file_path" ]; then
  echo "Updating $file_path"
  sed -i '' 's/func(tx store\.DBTX)/func(t \*testing.T, tx \*sql.Tx)/g' "$file_path"
else
  echo "WARNING: File $file_path not found. Skipping."
fi

# Update cmd/server/main_task_test.go
file_path="$PROJECT_ROOT/cmd/server/main_task_test.go"
if [ -f "$file_path" ]; then
  echo "Updating $file_path"
  sed -i '' 's/func(tx store\.DBTX)/func(t \*testing.T, tx \*sql.Tx)/g' "$file_path"
else
  echo "WARNING: File $file_path not found. Skipping."
fi

# Update card_store_crud_test.go
file_path="$PROJECT_ROOT/internal/platform/postgres/card_store_crud_test.go"
if [ -f "$file_path" ]; then
  echo "Updating $file_path"
  sed -i '' 's/func(tx \*sql\.Tx)/func(t \*testing.T, tx \*sql.Tx)/g' "$file_path"

  # Ensure imports are correct
  if ! grep -q "database/sql" "$file_path"; then
    sed -i '' '/^import (/,/)/ s/)/\t"database\/sql"\n)/' "$file_path"
  fi
else
  echo "WARNING: File $file_path not found. Skipping."
fi

echo "Done updating files"
