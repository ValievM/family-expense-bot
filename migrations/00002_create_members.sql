-- +goose Up
CREATE TABLE members (
    user_id BIGINT PRIMARY KEY CHECK (user_id > 0),
    display_name TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE members;
