-- +goose Up
-- +goose StatementBegin
-- Create cards table
CREATE TABLE cards (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    memo_id UUID NOT NULL,
    content JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Add foreign key constraints
    CONSTRAINT fk_cards_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_cards_memo
        FOREIGN KEY (memo_id)
        REFERENCES memos(id)
        ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX idx_cards_user_id ON cards(user_id);
CREATE INDEX idx_cards_memo_id ON cards(memo_id);
CREATE INDEX idx_cards_created_at ON cards(created_at DESC);

-- Create a GIN index on the JSONB content for efficient querying of card content
CREATE INDEX idx_cards_content ON cards USING GIN (content jsonb_path_ops);

-- Comment table and columns
COMMENT ON TABLE cards IS 'Flashcards generated from user memos';
COMMENT ON COLUMN cards.id IS 'Unique identifier (UUID) for the card';
COMMENT ON COLUMN cards.user_id IS 'Reference to the user who owns the card';
COMMENT ON COLUMN cards.memo_id IS 'Reference to the memo from which the card was generated';
COMMENT ON COLUMN cards.content IS 'Card content in JSONB format (front, back, hints, tags, etc.)';
COMMENT ON COLUMN cards.created_at IS 'Timestamp when the card was created';
COMMENT ON COLUMN cards.updated_at IS 'Timestamp when the card was last updated';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop cards table
DROP TABLE IF EXISTS cards;
-- +goose StatementEnd
