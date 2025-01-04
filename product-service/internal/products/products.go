package products

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type Conf struct {
	db *sql.DB
}

func NewConf(db *sql.DB) (*Conf, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &Conf{db: db}, nil
}

func (c *Conf) InsertProduct(ctx context.Context, newProduct NewProduct) (Product, error) {
	id := uuid.NewString()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	var product Product

	err := c.withTx(ctx, func(tx *sql.Tx) error {
		query := `
      INSERT INTO products
      (id, name, description, price, category, stock, created_at, updated_at)
      VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
      RETURNING id, name, description, price, category, stock, created_at, updated_at
      `
		err := tx.QueryRowContext(ctx, query, id, newProduct.Name, newProduct.Description, newProduct.Price, newProduct.Category, newProduct.Stock, createdAt, updatedAt).
			Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.Category, &product.Stock, &product.CreatedAt, &product.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert product: %w", err)
		}

		return nil
	})

	if err != nil {
		return Product{}, fmt.Errorf("failed to insert product: %w", err)
	}

	return product, nil
}

func (c *Conf) GetPriceAndStock(ctx context.Context, productID string) (string, int, error) {
	var priceID string
	var stock int

	err := c.withTx(ctx, func(tx *sql.Tx) error {
		query := `
			SELECT 
				pps.price_id, 
				p.stock
			FROM 
				product_pricing_stripe pps
			INNER JOIN 
				products p
			ON 
				pps.product_id = p.id
			WHERE 
				p.id = $1;
		`

		err := tx.QueryRowContext(ctx, query, productID).Scan(&priceID, &stock)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("no product found for productID: %s", productID)
			}
			return fmt.Errorf("failed to fetch priceID and stock: %w", err)
		}
		return nil
	})

	if err != nil {
		return "", 0, err
	}

	return priceID, stock, nil
}

func (c *Conf) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	// Start a new transaction using the context.
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err) // Return an error if the transaction cannot be started.
	}

	// Execute the provided function (`fn`) within the transaction.
	if err := fn(tx); err != nil {
		// If the function returns an error, attempt to roll back the transaction.
		er := tx.Rollback()
		if er != nil && !errors.Is(err, sql.ErrTxDone) {
			// If rollback also fails (and it's not because the transaction is already done),
			// return an error indicating the failure to roll back.
			return fmt.Errorf("failed to rollback withTx: %w", err)
		}
		// Return the original error from the function execution.
		return fmt.Errorf("failed to execute withTx: %w", err)
	}

	// If no errors occur, commit the transaction to apply the changes.
	err = tx.Commit()
	if err != nil {
		// Return an error if the transaction commit fails.
		return fmt.Errorf("failed to commit withTx: %w", err)
	}

	// Return nil if the function executes successfully and the transaction is committed.
	return nil
}
