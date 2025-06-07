-- +goose Up
-- +goose StatementBegin
-- Create memo status enum type with defensive check
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'memo_status') THEN
        CREATE TYPE memo_status AS ENUM (
            'pending',
            'processing',
            'completed',
            'completed_with_errors',
            'failed'
        );
    END IF;
END $$;

-- Create memos table
CREATE TABLE memos (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    text TEXT NOT NULL,
    status memo_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Add foreign key constraint
    CONSTRAINT fk_memos_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX idx_memos_user_id ON memos(user_id);
CREATE INDEX idx_memos_status ON memos(status);
CREATE INDEX idx_memos_created_at ON memos(created_at DESC);

-- Comment table and columns
COMMENT ON TABLE memos IS 'User-submitted text entries for generating flashcards';
COMMENT ON COLUMN memos.id IS 'Unique identifier (UUID) for the memo';
COMMENT ON COLUMN memos.user_id IS 'Reference to the user who created the memo';
COMMENT ON COLUMN memos.text IS 'The text content of the memo';
COMMENT ON COLUMN memos.status IS 'Processing status of the memo (pending, processing, completed, completed_with_errors, failed)';
COMMENT ON COLUMN memos.created_at IS 'Timestamp when the memo was created';
COMMENT ON COLUMN memos.updated_at IS 'Timestamp when the memo was last updated';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop memos table
DROP TABLE IF EXISTS memos;

-- Drop memo_status enum type
DROP TYPE IF EXISTS memo_status;
-- +goose StatementEnd
