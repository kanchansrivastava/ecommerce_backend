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

CREATE TABLE IF NOT EXISTS product_pricing_stripe  (
    id SERIAL PRIMARY KEY, -- Unique identifier for the record
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE, -- Foreign key referencing products table
    stripe_product_id TEXT NOT NULL UNIQUE , -- Stripe product ID
    price_id TEXT NOT NULL UNIQUE, -- Stripe price ID
    price BIGINT NOT NULL CHECK (price >= 0), -- must be non-negative
    created_at TIMESTAMP, -- Timestamp when the record was created
    updated_at TIMESTAMP -- Timestamp when the record was last updated
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS product_pricing_stripe;

-- +goose StatementEnd
