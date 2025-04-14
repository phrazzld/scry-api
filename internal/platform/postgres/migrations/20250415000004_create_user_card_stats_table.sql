-- +goose Up
-- +goose StatementBegin
-- Create user_card_stats table
CREATE TABLE user_card_stats (
    user_id UUID NOT NULL,
    card_id UUID NOT NULL,
    interval INTEGER NOT NULL DEFAULT 0,
    ease_factor DECIMAL(4,2) NOT NULL DEFAULT 2.5,
    consecutive_correct INTEGER NOT NULL DEFAULT 0,
    last_reviewed_at TIMESTAMP WITH TIME ZONE,
    next_review_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    review_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Define composite primary key
    PRIMARY KEY (user_id, card_id),

    -- Add foreign key constraints
    CONSTRAINT fk_stats_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_stats_card
        FOREIGN KEY (card_id)
        REFERENCES cards(id)
        ON DELETE CASCADE,

    -- Add constraints for validity
    CONSTRAINT check_interval_positive
        CHECK (interval >= 0),

    CONSTRAINT check_ease_factor_range
        CHECK (ease_factor > 1.0 AND ease_factor <= 2.5)
);

-- Create indexes
-- The most critical index for performance is on next_review_at,
-- as it will be used to find cards due for review
CREATE INDEX idx_stats_next_review_at ON user_card_stats(next_review_at);
CREATE INDEX idx_stats_user_next_review_at ON user_card_stats(user_id, next_review_at);

-- Comment table and columns
COMMENT ON TABLE user_card_stats IS 'SRS algorithm data for user-card pairs';
COMMENT ON COLUMN user_card_stats.user_id IS 'User ID - part of composite primary key';
COMMENT ON COLUMN user_card_stats.card_id IS 'Card ID - part of composite primary key';
COMMENT ON COLUMN user_card_stats.interval IS 'Current interval in days for the SRS algorithm';
COMMENT ON COLUMN user_card_stats.ease_factor IS 'Ease factor (1.3-2.5) for the SRS algorithm';
COMMENT ON COLUMN user_card_stats.consecutive_correct IS 'Count of consecutive correct answers';
COMMENT ON COLUMN user_card_stats.last_reviewed_at IS 'When the card was last reviewed';
COMMENT ON COLUMN user_card_stats.next_review_at IS 'When the card should be reviewed next';
COMMENT ON COLUMN user_card_stats.review_count IS 'Total number of times the card has been reviewed';
COMMENT ON COLUMN user_card_stats.created_at IS 'Timestamp when the stats record was created';
COMMENT ON COLUMN user_card_stats.updated_at IS 'Timestamp when the stats record was last updated';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop user_card_stats table
DROP TABLE IF EXISTS user_card_stats;
-- +goose StatementEnd
