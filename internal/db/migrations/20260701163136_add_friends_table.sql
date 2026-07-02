-- +goose Up
CREATE TABLE IF NOT EXISTS friends (
    user_id   UUID NOT NULL REFERENCES users(id),
    friend_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, friend_id)
);
-- +goose Down
DROP TABLE IF EXISTS friends;
