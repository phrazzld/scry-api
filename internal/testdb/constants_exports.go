//go:build exported_core_functions

package testdb

// MigrationTableName is the name of the table used by goose to track migrations.
// This constant is exported for use in both test and production code.
const MigrationTableName = "schema_migrations"
