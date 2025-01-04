package products

import (
	"time"
)

// Product represents the products table.
type Product struct {
	ID          string    `json:"id"`                    // Product ID
	Name        string    `json:"name"`                  // Product name
	Description string    `json:"description,omitempty"` // Product description (optional)
	Price       string    `json:"price"`                 // Product price (non-negative)
	Category    string    `json:"category,omitempty"`    // Product category (optional)
	Stock       int       `json:"stock"`                 // Stock level (non-negative)
	CreatedAt   time.Time `json:"-"`                     // Timestamp when created
	UpdatedAt   time.Time `json:"-"`                     // Timestamp when last updated

}

type NewProduct struct {
	Name        string `json:"name" validate:"required,max=255"`               // Name of the product (required, max 255 characters)
	Description string `json:"description,omitempty" validate:"max=500"`       // Description (optional, max 500 characters)
	Price       string `json:"price" validate:"required"`                      // Price (required, non-negative)
	Category    string `json:"category,omitempty" validate:"required,max=100"` // Category (optional, max 100 characters)
	Stock       int    `json:"stock" validate:"required,min=0"`                // Stock level (required, non-negative)
}

// ProductPricing represents the product_pricing table.
type ProductPricing struct {
	ID              int       `json:"id"`                // Unique ID for pricing
	ProductID       string    `json:"product_id" `       // Foreign key for products
	StripeProductID string    `json:"stripe_product_id"` // Stripe product ID
	PriceID         string    `json:"price_id" `         // Stripe price ID
	Price           int64     `json:"price"`             // Price (must be non-negative)
	CreatedAt       time.Time `json:"-"`                 // Timestamp when created
	UpdatedAt       time.Time `json:"-"`                 // Timestamp when last updated
}
