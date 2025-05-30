-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ADD CONSTRAINT users_balance_nonnegative CHECK (balance >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
DROP CONSTRAINT users_balance_nonnegative;
-- +goose StatementEnd
