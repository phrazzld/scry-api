#!/bin/bash
# Script to update Go build tags for integration tests

# 1. Add integration tag to postgres test files
for file in $(find /Users/phaedrus/Development/scry/scry-api/internal/platform/postgres -name "*_test.go"); do
  package_line=$(head -n 1 "$file")
  if [[ "$package_line" == "package postgres" ]]; then
    sed -i '' '1s/^/\/\/go:build integration\n\n/' "$file"
    echo "Added integration tag to $file (package postgres)"
  elif [[ "$package_line" == "package postgres_test" ]]; then
    sed -i '' '1s/^/\/\/go:build integration\n\n/' "$file"
    echo "Added integration tag to $file (package postgres_test)"
  else
    echo "WARNING: Unexpected package declaration in $file: $package_line"
  fi
done

# 2. Add integration tag to service tx_test.go files
for file in $(find /Users/phaedrus/Development/scry/scry-api/internal/service -name "*_tx_test.go"); do
  sed -i '' '1s/^/\/\/go:build integration\n\n/' "$file"
  echo "Added integration tag to $file"
done

# 3. Add integration tag to cmd/server integration_test.go files
for file in $(find /Users/phaedrus/Development/scry/scry-api/cmd/server -name "*_integration_test.go"); do
  # Check if file already has a build tag
  if grep -q '//go:build' "$file"; then
    echo "WARNING: File $file already has a build tag. Skipping."
  else
    sed -i '' '1s/^/\/\/go:build integration\n\n/' "$file"
    echo "Added integration tag to $file"
  fi
done

# 4. Replace test_without_external_deps with integration in card test files
for file in "/Users/phaedrus/Development/scry/scry-api/cmd/server/card_review_api_test.go" "/Users/phaedrus/Development/scry/scry-api/cmd/server/card_management_api_test.go"; do
  sed -i '' 's/\/\/go:build test_without_external_deps/\/\/go:build integration/g' "$file"
  sed -i '' '/\/\/ +build test_without_external_deps/d' "$file"
  echo "Replaced test_without_external_deps with integration in $file"
done

echo "Build tag updates completed."
