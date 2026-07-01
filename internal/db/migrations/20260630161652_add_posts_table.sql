-- +goose Up
CREATE TABLE IF NOT EXISTS posts (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    "text"     TEXT        NOT NULL,
    creator_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Feed queries: posts by a user sorted by newest first
CREATE INDEX idx_posts_creator_created ON posts (creator_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS posts;