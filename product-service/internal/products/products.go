package products

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/price"
	"os"
	"strconv"
	"strings"
	"time"
)

type Conf struct {
	db *sql.DB
}

var (
	ErrInvalidPrice = errors.New("invalid price")
)

func NewConf(db *sql.DB) (Conf, error) {
	if db == nil {
		return Conf{}, fmt.Errorf("db is nil")
	}
	return Conf{db: db}, nil
}

func (c *Conf) InsertProduct(ctx context.Context, newProduct NewProduct) (Product, error) {
	// Generate a new UUID for the product ID
	id := uuid.New().String()

	// Timestamps for creation and update
	createdAt := time.Now()
	updatedAt := time.Now()

	_, err := RupeesToPaisa(newProduct.Price)
	if err != nil {
		return Product{}, fmt.Errorf("%w %w", err, ErrInvalidPrice)
	}
	// Define a Product struct to capture the returned data
	var product Product

	// Insert the product into the database within a transaction
	err = c.withTx(ctx, func(tx *sql.Tx) error {
		// SQL query for inserting a new product
		query := `
		INSERT INTO products
		(id, name, description, price, category, stock, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, description, price, category, stock, created_at, updated_at
		`

		// Execute the query
		err := tx.QueryRowContext(ctx, query, id, newProduct.Name, newProduct.Description, newProduct.Price,
			newProduct.Category, newProduct.Stock, createdAt, updatedAt).
			Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.Category, &product.Stock, &product.CreatedAt, &product.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert product: %w", err)
		}

		// Successfully inserted the product, return the resulting Product struct
		return nil
	})

	if err != nil {
		return Product{}, fmt.Errorf("failed to insert product: %w", err)
	}

	// Return the inserted product
	return product, nil
}

// RupeesToPaisa converts a price from rupees (e.g., "99.99") to paise (e.g., 9999).
func RupeesToPaisa(price string) (uint64, error) {

	price = strings.TrimSpace(price)
	// Split the price into integer and fractional parts using the dot (.)
	parts := strings.Split(price, ".")
	if len(parts) > 2 {
		return 0, errors.New("price cannot have more than one decimal point")
	}

	// Convert integer part to paise (e.g., "99" -> 9900)
	integerPart, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, errors.New("invalid integer part in price")
	}

	if integerPart > 10000000 { // greater than ten million, then we are not selling that product,
		return 0, errors.New("price cannot be greater than 10 million")
	}

	// Handle fractional part if it exists
	var fractionalPart uint64 = 0
	if len(parts) > 1 {
		// Ensure no more than two decimal places
		if len(parts[1]) > 2 {
			return 0, errors.New("price cannot have more than two decimal places")
		}
		// Add trailing zero if fractional part has only one digit
		for len(parts[1]) < 2 {
			parts[1] += "0"
		}
		// Convert fractional part to paise
		fractionalPart, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0, errors.New("invalid fractional part in price")
		}
	}

	return integerPart*100 + fractionalPart, nil
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

/*
	CreateCustomerStripe

1. Start a transaction using `withTx()` to group all related database operations.
  - Ensures atomicity (all succeed or all rollback).

2. Define a SQL query to check if a Stripe customer ID already exists for the given user ID in the `users_stripe` table.

3. Execute the query and handle the result:
  - If a `stripe_customer_id` exists, do nothing (no need to create a new customer).
  - If `sql.ErrNoRows` occurs, it means the user does not have a Stripe customer ID. Proceed to create one.

4. Check the environment for the `STRIPE_TEST_KEY`:
  - If not set, return an error. This is required to interact with the Stripe API.

5. Set the Stripe API key and create a new customer using Stripe's API (`customer.New()`):
  - Pass the `name` and `email` into the `stripe.CustomerParams`.
  - On success, retrieve the new Stripe customer ID.

6. Prepare a SQL query to insert the user record into the `users_stripe` table:
  - Insert the `userId`, `email`, `stripe_customer_id`, along with the current timestamp for `created_at` and `updated_at`.

7. Execute the `INSERT` query to save the user and Stripe customer data to the database:
  - Check how many rows were affected by the query.
  - If no rows are affected, return an error indicating the operation failed.

8. Handle any errors at each step, providing clear error messages for debugging.

9. End the transaction:
  - If no errors occurred, the transaction commits the changes to the database.
  - If any error occurred, the transaction rolls back all changes.

10. Return `nil` to indicate success or the error encountered.
*/

// Function to add a new product with pricing to product_pricing_stripe table
func (c *Conf) CreateProductPriceStripe(product Product) error {
	// Check and set Stripe key
	sKey := os.Getenv("STRIPE_TEST_KEY")
	if sKey == "" {
		return errors.New("STRIPE_TEST_KEY is not set")
	}

	stripe.Key = sKey

	err := c.withTx(context.Background(), func(tx *sql.Tx) error {
		// Query to check if product already exists
		checkQuery := `
			SELECT stripe_product_id 
			FROM product_pricing_stripe
			WHERE product_id = $1
		`

		var stripeProductID string
		err := tx.QueryRow(checkQuery, product.ID).Scan(&stripeProductID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to check if product exists: %w", err)
			}
			paisa, err := RupeesToPaisa(product.Price)

			if err != nil {
				return fmt.Errorf("%w %w", err, ErrInvalidPrice)
			}

			// Stripe: Create product and price since product is not in database
			params := &stripe.PriceParams{
				Currency:   stripe.String(string(stripe.CurrencyINR)), // Set currency
				UnitAmount: stripe.Int64(int64(paisa)),                // Set price in smallest currency units (e.g., cents for USD)
				ProductData: &stripe.PriceProductDataParams{
					Name: stripe.String(product.Name), // Name of the product

					//UnitLabel: stripe.String("1"),
				},
			}

			result, stripeErr := price.New(params) // Create the price on Stripe
			if stripeErr != nil {
				fmt.Println(stripeErr)
				return fmt.Errorf("failed to create product and price on Stripe: %w", stripeErr)
			}

			// Insert into database
			insertQuery := `
				INSERT INTO product_pricing_stripe 
					(product_id, stripe_product_id, price_id, price, created_at, updated_at)
				VALUES 
					($1, $2, $3, $4, $5, $6)
			`

			createdAt := time.Now().UTC()
			updatedAt := time.Now().UTC()
			_, dbErr := tx.Exec(insertQuery,
				product.ID,        // product_id
				result.Product.ID, // stripe_product_id
				result.ID,         // price_id
				paisa,             // price (in smallest currency units)
				createdAt,         // created_at
				updatedAt,         // updated_at
			)
			if dbErr != nil {
				return fmt.Errorf("failed to save product pricing info into database: %w", dbErr)
			}

			// Success case
			return nil
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (c *Conf) GetProductStockAndStripePriceId(ctx context.Context, productID string) (int, string, error) {
	// Variables to store stock and price_id
	var stock int
	var priceID string

	// Execute the query within a transaction
	err := c.withTx(ctx, func(tx *sql.Tx) error {
		// SQL query to get the stock and price for the given product ID
		query := `
		SELECT 
			p.stock, 
			pp.price_id
		FROM 
			products p
		JOIN 
			product_pricing_stripe pp
		ON 
			p.id = pp.product_id
		WHERE 
			p.id = $1
		`

		// Execute the query inside the transaction
		row := tx.QueryRowContext(ctx, query, productID)
		err := row.Scan(&stock, &priceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("no product found with id: %s", productID)
			}
			return fmt.Errorf("failed to query database: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch product stock and price: %w", err)
	}

	// Return the stock and priceID
	return stock, priceID, nil
}

func (c *Conf) DecrementStock(ctx context.Context, productID string, quantity int) error {
	// Decrement stock directly in the database
	err := c.withTx(ctx, func(tx *sql.Tx) error {
		// SQL query to decrement stock atomically
		query := `
		UPDATE products
		SET stock = stock - $1, updated_at = $2
		WHERE id = $3 AND stock > 0
		`

		updatedAt := time.Now().UTC() // Current timestamp

		// Execute the query
		result, err := tx.ExecContext(ctx, query, quantity, updatedAt, productID)
		if err != nil {
			return fmt.Errorf("failed to decrement stock: %w", err)
		}

		// Check if any rows were affected
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("failed to decrement stock")
		}

		// Successfully decremented stock
		return nil
	})

	// Handle errors
	if err != nil {
		return err
	}

	// Successfully completed
	return nil

}
