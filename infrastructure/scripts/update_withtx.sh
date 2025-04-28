#!/bin/bash

# This script updates all occurrences of testutils.WithTx to use the new function signature

# Find all files that use testutils.WithTx
files=$(grep -l "testutils.WithTx(t, db, func(tx store.DBTX)" --include="*.go" -r /Users/phaedrus/Development/scry/scry-api/internal)

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
