-- +goose Up
-- +goose StatementBegin
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Add index on status and updated_at for efficient querying
CREATE INDEX idx_tasks_status_updated_at ON tasks(status, updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasks_status_updated_at;
DROP TABLE IF EXISTS tasks;
-- +goose StatementEnd
