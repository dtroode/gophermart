-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserForUpdate :one
SELECT * FROM users
WHERE id = $1 LIMIT 1
FOR UPDATE;

-- name: GetUserByLogin :one
SELECT * FROM users
WHERE login = $1 LIMIT 1;

-- name: SaveUser :one
INSERT INTO users (login, password)
VALUES ($1, $2)
RETURNING id, login, password, created_at, balance;

-- name: SetUserBalance :one
UPDATE users
SET balance = $1
WHERE id = $2
RETURNING id, login, created_at, balance;

-- name: IncrementUserBalance :one
UPDATE users
SET balance = balance + $1
WHERE id = $2
RETURNING id, login, created_at, balance;

-- name: GetOrderByNumber :one
SELECT * FROM orders
WHERE num = $1 LIMIT 1;

-- name: SaveOrder :one
INSERT INTO orders (user_id, num, accrual, status)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, created_at, num, accrual, status;

-- name: SetOrderAccrual :one
UPDATE orders
SET accrual = $1
WHERE id = $2
RETURNING id, user_id, created_at, num, accrual, status;

-- name: SetOrderStatus :one
UPDATE orders
SET status = $1
WHERE id = $2
RETURNING id, user_id, created_at, num, accrual, status;

-- name: GetUserWithdrawalSum :one
SELECT COALESCE(SUM(amount), 0)::real FROM withdrawals
WHERE user_id = $1;

-- name: GetUserWithdrawals :many
SELECT id, user_id, order_num, created_at, amount FROM withdrawals
WHERE user_id = $1;

-- name: GetUserOrdersNewestFirst :many
SELECT * FROM orders
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: CreateWithdrawal :one
INSERT INTO withdrawals (user_id, order_num, amount)
VALUES ($1, $2, $3)
RETURNING id, user_id, order_num, created_at, amount;
