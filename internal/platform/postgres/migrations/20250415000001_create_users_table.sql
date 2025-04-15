-- +goose Up
-- +goose StatementBegin
-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    hashed_password TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on email for faster lookups
CREATE INDEX idx_users_email ON users(email);

-- Comment table and columns
COMMENT ON TABLE users IS 'User accounts for authentication and identification';
COMMENT ON COLUMN users.id IS 'Unique identifier (UUID) for the user';
COMMENT ON COLUMN users.email IS 'User email address, used for authentication';
COMMENT ON COLUMN users.hashed_password IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.created_at IS 'Timestamp when the user was created';
COMMENT ON COLUMN users.updated_at IS 'Timestamp when the user was last updated';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop users table (will cascade to other tables with foreign keys)
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
