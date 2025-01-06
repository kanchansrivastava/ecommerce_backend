-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS orders (
    id UUID NOT NULL PRIMARY KEY,                     -- Unique ID for the order
    user_id UUID NOT NULL,                            -- UUID of the user placing the order
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'paid', 'canceled')), -- Status of the order
    stripe_transaction_id TEXT,                      -- Stripe unique transaction ID
    total_price BIGINT NOT NULL,                     -- Total price in cents (sum of all items)
    created_at TIMESTAMP DEFAULT NOW(),              -- Timestamp of order creation
    updated_at TIMESTAMP DEFAULT NOW()               -- Timestamp of last update
);


CREATE TABLE IF NOT EXISTS order_items (
    id UUID NOT NULL PRIMARY KEY,                    -- Unique ID for the order item
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE, -- Foreign key to orders
    product_id UUID NOT NULL,                        -- UUID of the product
    quantity INT NOT NULL CHECK (quantity > 0),      -- Quantity of the product
    price_per_unit BIGINT NOT NULL,                  -- Price per unit in cents
    total_price  BIGINT NOT NULL, -- Total price for this item
    created_at TIMESTAMP DEFAULT NOW()               -- Timestamp of item addition
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
-- +goose StatementEnd
