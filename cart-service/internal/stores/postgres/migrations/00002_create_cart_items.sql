-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS cart_items (
    id SERIAL PRIMARY KEY, -- Unique identifier for the cart item
    cart_id INT NOT NULL REFERENCES cart(id) ON DELETE CASCADE, -- Foreign key referencing the carts table
    product_id UUID NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- Timestamp when the cart item was created
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Timestamp when the cart item was last updated
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cart_items;
-- +goose StatementEnd
