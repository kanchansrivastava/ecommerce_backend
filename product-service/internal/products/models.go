package products

import (
	"time"
)

type Product struct {
	ID          string    `json:"id"` // UUID
	Name        string    `db:"name" json:"name" binding:"required"`
	Description string    `db:"description" json:"description"`
	Price       string    `db:"price" json:"price" binding:"required"`
	Category    string    `db:"category" json:"category"`
	Stock       int       `db:"stock" json:"stock" binding:"required"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"` // Creation timestamp
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"` // Last updated timestamp
}

type NewProduct struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"omitempty"`
	Price       string `json:"price" binding:"required"`
	Category    string `json:"category" binding:"omitempty"`
	Stock       int    `json:"stock" binding:"required,gte=0"`
}

type StripeProductPricing struct {
	ID              int       `json:"id"`                // Unique identifier
	ProductID       string    `json:"product_id"`        // Foreign key referencing products table
	StripeProductID string    `json:"stripe_product_id"` // Stripe product ID
	PriceID         string    `json:"price_id"`          // Stripe price ID
	Price           uint64    `json:"price"`
	CreatedAt       time.Time `json:"created_at"` // Timestamp when the record was created
	UpdatedAt       time.Time `json:"updated_at"` // Timestamp when the record was last updated
}
