-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN balance TYPE INTEGER USING (balance * 100)::INTEGER;
ALTER TABLE users ALTER COLUMN balance SET DEFAULT 0;

ALTER TABLE withdrawals ALTER COLUMN amount TYPE INTEGER USING (amount * 100)::INTEGER;

ALTER TABLE orders ALTER COLUMN accrual TYPE INTEGER USING (accrual * 100)::INTEGER;
ALTER TABLE orders ALTER COLUMN accrual SET DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users ALTER COLUMN balance TYPE REAL USING balance / 100.0;
ALTER TABLE users ALTER COLUMN balance SET DEFAULT 0.0;

ALTER TABLE withdrawals ALTER COLUMN amount TYPE REAL USING amount / 100.0;

ALTER TABLE orders ALTER COLUMN accrual TYPE REAL USING accrual / 100.0;
ALTER TABLE orders ALTER COLUMN accrual SET DEFAULT 0.0;
-- +goose StatementEnd
