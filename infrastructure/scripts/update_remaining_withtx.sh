#!/bin/bash

# Update cmd/server/main_integration_test.go
file_path="/Users/phaedrus/Development/scry/scry-api/cmd/server/main_integration_test.go"
echo "Updating $file_path"
sed -i '' 's/func(tx store\.DBTX)/func(t \*testing.T, tx \*sql.Tx)/g' "$file_path"

# Update cmd/server/main_task_test.go
file_path="/Users/phaedrus/Development/scry/scry-api/cmd/server/main_task_test.go"
echo "Updating $file_path"
sed -i '' 's/func(tx store\.DBTX)/func(t \*testing.T, tx \*sql.Tx)/g' "$file_path"

# Update card_store_crud_test.go
file_path="/Users/phaedrus/Development/scry/scry-api/internal/platform/postgres/card_store_crud_test.go"
echo "Updating $file_path"
sed -i '' 's/func(tx \*sql\.Tx)/func(t \*testing.T, tx \*sql.Tx)/g' "$file_path"

# Ensure imports are correct
if ! grep -q "database/sql" "$file_path"; then
  sed -i '' '/^import (/,/)/ s/)/\t"database\/sql"\n)/' "$file_path"
fi

echo "Done updating files"
