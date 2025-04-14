# Database Migrations

This directory contains database migration files for the Scry API.

## Migration Commands

The application supports the following migration commands:

```
go run cmd/server/main.go -migrate=<command> [-name=<migration_name>]
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
go run cmd/server/main.go -migrate=up

# Rollback the last migration
go run cmd/server/main.go -migrate=down

# Show migration status
go run cmd/server/main.go -migrate=status

# Show current version
go run cmd/server/main.go -migrate=version

# Create a new migration
go run cmd/server/main.go -migrate=create -name=create_users_table
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
