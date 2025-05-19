CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    login varchar(64) NOT NULL UNIQUE,
    password varchar(64) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    balance real DEFAULT 0
);

CREATE TABLE orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL references users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    num varchar(256) NOT NULL UNIQUE,
    accrual real,
    status order_status NOT NULL DEFAULT 'NEW'
);

CREATE TABLE withdrawals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL references users(id),
    order_num varchar(256) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    amount real NOT NULL
);