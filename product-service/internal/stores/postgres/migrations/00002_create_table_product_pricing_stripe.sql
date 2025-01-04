-- +goose Up
-- +goose StatementBegin
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
DROP TABLE IF EXISTS product_pricing_stripe;
-- +goose StatementEnd
