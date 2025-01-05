package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (c *Conf) AddToCartDB(ctx context.Context, userID string, productID string, quantity int, stock int, stripePriceID string) error {
	return c.withTx(ctx, func(tx *sql.Tx) error {
		var cartID int
		// Query to find an active cart for the user
		queryActiveCart := `
			SELECT id
			FROM cart
			WHERE user_id = $1 AND status = 'active'
			FOR UPDATE
		`
		fmt.Printf("userID=%s, productID=%s, quantity=%d, stock=%d, stripePriceID=%s\n",
			userID, productID, quantity, stock, stripePriceID)
		err := tx.QueryRowContext(ctx, queryActiveCart, userID).Scan(&cartID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// No active cart exists; create a new cart
				queryCreateCart := `
					INSERT INTO cart (user_id, status, created_at, updated_at)
					VALUES ($1, 'active', NOW(), NOW())
					RETURNING id
				`
				err = tx.QueryRowContext(ctx, queryCreateCart, userID).Scan(&cartID)

				if err != nil {
					return fmt.Errorf("failed to create new cart: %w", err)
				}
			} else {
				return fmt.Errorf("failed to query active cart: %w", err)
			}
		}

		// Check if the product already exists in the cart
		queryCartItem := `
			SELECT id, quantity
			FROM cart_items
			WHERE cart_id = $1 AND product_id = $2
		`
		var cartItemID int
		var existingQuantity int

		err = tx.QueryRowContext(ctx, queryCartItem, cartID, productID).Scan(&cartItemID, &existingQuantity)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Product does not exist in the cart; check if we can add the quantity
				if quantity > stock {
					return fmt.Errorf("insufficient stock: requested %d, available %d", quantity, stock)
				}

				// Insert the new cart item
				queryAddCartItem := `
					INSERT INTO cart_items (cart_id, product_id, quantity, created_at, updated_at)
					VALUES ($1, $2, $3, NOW(), NOW())
				`
				_, err = tx.ExecContext(ctx, queryAddCartItem, cartID, productID, quantity)
				if err != nil {
					return fmt.Errorf("failed to add product to cart: %w", err)
				}
			} else {
				return fmt.Errorf("failed to query cart items: %w", err)
			}
		} else {
			// Product already exists in the cart; check if we can add to the existing quantity
			newQuantity := existingQuantity + quantity
			if newQuantity > stock {
				return fmt.Errorf("insufficient stock: requested %d, available %d", newQuantity, stock)
			}

			// Update the cart item quantity
			queryUpdateCartItem := `
				UPDATE cart_items
				SET quantity = $1, updated_at = NOW()
				WHERE id = $2
			`
			_, err = tx.ExecContext(ctx, queryUpdateCartItem, newQuantity, cartItemID)
			if err != nil {
				return fmt.Errorf("failed to update cart item quantity: %w", err)
			}
		}

		return nil
	})
}

func (c *Conf) GetActiveCartItems(ctx context.Context, userID string) (*CartResponse, error) {
	var cartID int
	var items []CartItem

	err := c.withTx(ctx, func(tx *sql.Tx) error {
		// Query to find an active cart for the user
		queryActiveCart := `
		SELECT c.id
		FROM cart c
		WHERE c.user_id = $1 AND c.status = 'active'
		LIMIT 1
		FOR UPDATE
	`
		err := tx.QueryRowContext(ctx, queryActiveCart, userID).Scan(&cartID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("no active cart found for user: %s", userID)
			}
			return fmt.Errorf("failed to query active cart: %w", err)
		}

		// Query for cart items
		queryItems := `
            SELECT ci.product_id,ci.quantity
            FROM cart_items ci
            WHERE ci.cart_id = $1
        `
		rows, err := tx.QueryContext(ctx, queryItems, cartID)
		if err != nil {
			return fmt.Errorf("failed to query cart items: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item CartItem
			if err := rows.Scan(&item.ProductID, &item.Quantity); err != nil {
				return fmt.Errorf("failed to scan cart item: %w", err)
			}
			items = append(items, item)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating cart items: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &CartResponse{
		Items: items,
	}, nil
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
