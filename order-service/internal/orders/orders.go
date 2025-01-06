package orders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Conf struct {
	db *sql.DB
}

func NewConf(db *sql.DB) (Conf, error) {
	if db == nil {
		return Conf{}, fmt.Errorf("db is nil")
	}
	return Conf{db: db}, nil
}

const (
	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusCanceled = "canceled"
)

func (c *Conf) CreateOrder(ctx context.Context, orderId, userId string, items []DetailedCartItem, totalPrice int64) error {
	// Define the status and timestamps
	status := StatusPending
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	// Use a transaction to execute the inserts
	err := c.withTx(ctx, func(tx *sql.Tx) error {
		// SQL query for inserting a new order
		orderQuery := `
        INSERT INTO orders
        (id, user_id, status, total_price, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        `
		// Execute the query to insert the order
		_, err := tx.ExecContext(ctx, orderQuery, orderId, userId, status, totalPrice, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert order: %w", err)
		}

		// SQL query for inserting order items
		itemQuery := `
        INSERT INTO order_items
        (id, order_id, product_id, quantity, price_per_unit, total_price, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        `

		// Insert each item in the order
		for _, item := range items {
			// Calculate total price for the item
			itemTotalPrice := int64(item.Quantity) * int64(item.Price)

			// Generate a unique ID for the order item
			itemId := uuid.NewString()

			// Execute the query to insert the order item
			_, err := tx.ExecContext(ctx, itemQuery, itemId, orderId, item.ProductID, item.Quantity, int64(item.Price), itemTotalPrice, createdAt)
			if err != nil {
				return fmt.Errorf("failed to insert order item: %w", err)
			}
		}

		// Successfully inserted the order and items
		return nil
	})

	if err != nil {
		// Return an error if the transaction fails
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Return nil if successful
	return nil
}

func (c *Conf) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	if err := fn(tx); err != nil {
		er := tx.Rollback()
		if er != nil && !errors.Is(err, sql.ErrTxDone) {
			return fmt.Errorf("failed to rollback withTx: %w", err)
		}
		return fmt.Errorf("failed to execute withTx: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit withTx: %w", err)
	}
	return nil

}

func (c *Conf) UpdateOrder(ctx context.Context, orderId string, status string, stripeTransactionId string) error {
	updatedAt := time.Now().UTC() // Current timestamp

	// Use a transaction to ensure consistency
	err := c.withTx(ctx, func(tx *sql.Tx) error {

		// Step 3: Perform the update since the `updated_at` condition is met
		queryUpdate := `
		UPDATE orders
		SET status = $1, stripe_transaction_id = $2, updated_at = $3
		WHERE id = $4
		`

		res, err := tx.ExecContext(ctx, queryUpdate, status, stripeTransactionId, updatedAt, orderId)
		if err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		num, err := res.RowsAffected()
		if num == 0 || err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		// Successfully updated the order
		return nil
	})

	if err != nil {
		// Return the error, if any
		return err
	}

	// Return nil if the update is successful or skipped gracefully
	return nil
}
