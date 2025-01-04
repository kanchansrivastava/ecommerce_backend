package users

import "time"

// User struct represents the users table in the stores
type User struct {
	ID               string    `json:"id"` // UUID
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	PasswordHash     string    `json:"-"`          // Password hash (not exposed in JSON)
	StripeCustomerID string    `json:"-"`          // Not part of json output
	CreatedAt        time.Time `json:"created_at"` // Timestamp of creation
	UpdatedAt        time.Time `json:"updated_at"` // Timestamp of last update
}

// NewUser struct represents the data required when creating a new user
type NewUser struct {
	Name     string `json:"name" validate:"required,min=2,max=100"` // User name must be between 2-100 chars
	Email    string `json:"email" validate:"required,email"`        // Valid email required
	Password string `json:"password" validate:"required,min=5"`     // Password must be at least 5 characters long
}
