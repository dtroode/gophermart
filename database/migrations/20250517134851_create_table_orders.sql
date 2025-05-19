-- +goose Up
-- +goose StatementBegin
CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE IF NOT EXISTS orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL references users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    num varchar(256) NOT NULL UNIQUE,
    accrual real,
    status order_status NOT NULL DEFAULT 'NEW'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS orders;

DROP TYPE order_status;
-- +goose StatementEnd
