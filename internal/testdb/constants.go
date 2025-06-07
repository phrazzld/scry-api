//go:build (integration || test_without_external_deps) && !exported_core_functions

package testdb

// MigrationTableName is the name of the table used by goose to track migrations.
// This constant is defined here to ensure consistency across the codebase.
const MigrationTableName = "schema_migrations"
