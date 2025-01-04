-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS users_stripe (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL UNIQUE,
    stripe_customer_id TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users_stripe;
-- +goose StatementEnd
