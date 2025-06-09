# Database Migrations

This directory contains database migration files for the Scry API.

## Migration Commands

The application supports the following migration commands:

```
make migrate-<command> [NAME=<migration_name>]
```

Or using the underlying Go command:
```
go run ./cmd/server -migrate=<command> [-name=<migration_name>]
```

### Available Commands

- `up`: Run all pending migrations
- `down`: Rollback the last applied migration
- `status`: Show the status of all migrations
- `version`: Show the current migration version
- `create`: Create a new migration file (requires `-name` flag)

### Examples

```sh
# Run all pending migrations
make migrate-up

# Rollback the last migration
make migrate-down

# Show migration status
make migrate-status

# Show current version
make migrate-version

# Create a new migration
make migrate-create NAME=create_users_table
```

## Migration File Format

Migration files follow this naming convention:
```
YYYYMMDDHHMMSS_name.sql
```

Each migration file contains both an "up" and "down" section:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
```

The "up" section contains SQL to apply the migration, while the "down" section contains SQL to roll it back.
