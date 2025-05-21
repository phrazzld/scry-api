#!/bin/bash

# This script updates all occurrences of testutils.WithTx to use the new function signature

# Get project root directory
PROJECT_ROOT=$(git rev-parse --show-toplevel)
if [ $? -ne 0 ]; then
  echo "ERROR: Failed to determine project root directory. Make sure you're in a git repository."
  exit 1
fi

# Find all files that use testutils.WithTx
files=$(grep -l "testutils.WithTx(t, db, func(tx store.DBTX)" --include="*.go" -r "$PROJECT_ROOT/internal")

for file in $files; do
  echo "Updating $file"
  # Ensure database/sql is imported
  if ! grep -q "database/sql" "$file"; then
    sed -i '' '/^import (/,/)/ s/)/\t"database\/sql"\n)/' "$file"
  fi

  # Update the function signature
  sed -i '' 's/testutils\.WithTx(t, db, func(tx store\.DBTX)/testutils.WithTx(t, db, func(t \*testing.T, tx \*sql.Tx)/g' "$file"
done

echo "Done updating files"
