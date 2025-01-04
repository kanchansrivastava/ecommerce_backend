-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY, -- Unique identifier for the product
    name TEXT NOT NULL, -- Name of the product (up to 255 characters)
    description TEXT, -- Detailed description of the product
    price TEXT NOT NULL,
    category TEXT,
    stock INTEGER NOT NULL CHECK (stock >= 0), -- Stock level, must be non-negative
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS products;
-- +goose StatementEnd
