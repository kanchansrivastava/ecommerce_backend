package products

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/price"
	"github.com/stripe/stripe-go/v81/product"
	"os"
	"time"
)

func (c *Conf) CreateProductStripe(ctx context.Context, productId, name, description string, productPrice uint) error {
	sKey := os.Getenv("STRIPE_TEST_KEY")
	if sKey == "" {
		return fmt.Errorf("STRIPE_TEST_KEY not set")
	}

	stripe.Key = sKey

	err := c.withTx(ctx, func(tx *sql.Tx) error {
		sqlQuery := `SELECT stripe_product_id FROM product_pricing_stripe WHERE product_id = $1`
		var stripeProductId string

		err := tx.QueryRowContext(ctx, sqlQuery, productId).Scan(&stripeProductId)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to fetch Stripe product ID: %w", err)
			}

			params := &stripe.ProductParams{
				Name:        stripe.String(name),
				Description: stripe.String(description),
				Active:      stripe.Bool(true),
			}
			productResult, err := product.New(params)
			if err != nil {
				return fmt.Errorf("failed to create Stripe product: %w", err)
			}
			stripeProductID := productResult.ID
			priceParams := &stripe.PriceParams{
				Currency:   stripe.String(string(stripe.CurrencyINR)),
				UnitAmount: stripe.Int64(int64(productPrice)),
				Product:    stripe.String(stripeProductID),
			}

			priceResult, err := price.New(priceParams)

			if err != nil {
				return fmt.Errorf("failed to create Stripe price: %w", err)
			}
			insertQuery := `
				INSERT INTO product_pricing_stripe (
					product_id, 
					stripe_product_id, 
					price_id, 
					price, 
					created_at, 
					updated_at
				)
				VALUES ($1, $2, $3, $4, $5, $6)
			`
			createdAt := time.Now().UTC()
			updatedAt := createdAt
			_, err = tx.ExecContext(ctx, insertQuery, productId, priceResult.Product.ID, priceResult.ID, productPrice, createdAt, updatedAt)
			if err != nil {
				return fmt.Errorf("failed to insert product into database: %w", err)
			}

			return nil
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
