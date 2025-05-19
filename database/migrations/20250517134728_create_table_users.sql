-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    login varchar(64) NOT NULL UNIQUE,
    password varchar(64) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    balance real DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
