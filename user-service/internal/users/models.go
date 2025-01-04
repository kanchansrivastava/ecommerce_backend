package users

import (
	"github.com/lib/pq"
	"time"
)

// User struct represents the users table in the stores
type User struct {
	ID               string         `json:"id"` // UUID
	Name             string         `json:"name"`
	Email            string         `json:"email"`
	PasswordHash     string         `json:"-"` // Password hash (not exposed in JSON)
	StripeCustomerID string         `json:"-"` // Not part of json output
	Roles            pq.StringArray `json:"roles"`
	CreatedAt        time.Time      `json:"created_at"` // Timestamp of creation
	UpdatedAt        time.Time      `json:"updated_at"` // Timestamp of last update
}

// NewUser struct represents the data required when creating a new user
type NewUser struct {
	Name     string   `json:"name" validate:"required,min=2,max=100"` // User name must be between 2-100 chars
	Email    string   `json:"email" validate:"required,email"`        // Valid email required
	Password string   `json:"password" validate:"required,min=5"`     // Password must be at least 5 characters long
	Roles    []string `json:"roles" validate:"required,unique,dive,oneof=user admin"`
}

//required: The roles field is mandatory.
//unique: Ensures there are no duplicate roles in the array (requires the validator's unique tag to be supported).
//dive: Applies validation rules to each individual element of the slice.
//oneof=user admin: Restricts each role value to either user or admin.
