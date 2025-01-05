-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS cart(
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    status VARCHAR(50) DEFAULT 'active', -- Can be 'active', 'abandoned', 'ordered'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cart;
-- +goose StatementEnd
