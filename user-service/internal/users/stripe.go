package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"log/slog"
	"os"
	"time"
	"user-service/pkg/logkey"
)

func (c *Conf) CreateCustomerStripe(ctx context.Context, userId, name, email string) error {
	// Step 1: Retrieve the Stripe secret key from the environment variables
	sKey := os.Getenv("STRIPE_TEST_KEY")
	if sKey == "" {
		// If the key is not set, return an error
		return fmt.Errorf("STRIPE_TEST_KEY not set")
	}

	// Step 2: Assign the Stripe API key to the Stripe library's internal configuration
	stripe.Key = sKey

	// Step 3: Begin a database transaction using the `withTx` method (assumed to be defined elsewhere)
	err := c.withTx(ctx, func(tx *sql.Tx) error {
		// Step 4: Define a SQL query to check if the user already has a Stripe customer ID in the database
		sqlQuery := `
				SELECT stripe_customer_id 
				FROM users_stripe
				WHERE user_id = $1
				`

		// Step 5: Declare a variable to hold the Stripe customer ID we fetch from the database
		var stripeCustomerId string

		// Step 6: Execute the query to get the Stripe customer ID for the given user ID
		err := tx.QueryRowContext(ctx, sqlQuery, userId).Scan(&stripeCustomerId)
		if err != nil {
			// Step 7: Handle the case where no rows are found (i.e., the user doesn't have a Stripe customer ID yet)
			if !errors.Is(err, sql.ErrNoRows) {
				// If the error is not `sql.ErrNoRows`, that means something went wrong with the query execution; return an error
				return fmt.Errorf("failed to fetch Stripe customer ID: %w", err)
			}

			// Step 8: If the user doesn't have a Stripe customer ID, create a new Stripe customer
			params := &stripe.CustomerParams{
				Name:  stripe.String(name),  // Set the customer's name
				Email: stripe.String(email), // Set the customer's email
			}

			// Step 9: Call the Stripe API to create a new customer using the parameters
			customerResult, err := customer.New(params)
			if err != nil {
				// Log the error and return it if the creation of the Stripe customer fails
				slog.Error("failed to create Stripe customer", slog.Any(logkey.ERROR, err))
				return fmt.Errorf("failed to create Stripe customer: %w", err)
			}

			// Step 10: Define the SQL query to insert the new Stripe customer into the `users_stripe` table
			query := `
		INSERT INTO users_stripe (user_id, email, stripe_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
			// Step 11: Get the current timestamp for record creation and updates
			createdAt := time.Now().UTC()
			updatedAt := createdAt

			// Step 12: Execute the query to insert the new Stripe customer into the database
			res, err := tx.ExecContext(ctx, query, userId, email, customerResult.ID, createdAt, updatedAt)
			if err != nil {
				// Log the error and return it if the database insertion fails
				slog.Error("failed to insert Stripe customer ID", slog.Any(logkey.ERROR, err))
				return fmt.Errorf("failed to insert Stripe customer ID: %w", err)
			}

			// Step 13: Check if the insertion affected any rows (it should affect exactly one row if successful)
			if num, err := res.RowsAffected(); num == 0 || err != nil {
				// If no rows were affected or another error occurred, return an error
				return fmt.Errorf("failed to insert Stripe customer ID: %w", err)
			}

			// Step 14: Return `nil` if the Stripe customer is successfully added to the database
			return nil
		}

		// Step 15: CustomerId already exist on stripe, no need to add the customer
		return nil
	})

	// Step 16: Handle any errors from the transaction function
	if err != nil {
		return err
	}

	// Step 17: If everything succeeds, return `nil`
	return nil
}
