package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

func (c *Conf) InsertUser(ctx context.Context, newUser NewUser) (User, error) {
	id := uuid.NewString()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	var user User
	err = c.withTx(ctx, func(tx *sql.Tx) error {
		// SQL query for inserting a new user
		query := `
		INSERT INTO users
		(id, name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, email, created_at, updated_at
	`
		// Execute the query
		err = tx.QueryRowContext(ctx, query, id, newUser.Name, newUser.Email, hashedPassword, createdAt, updatedAt).
			Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}

		// Successfully inserted the user, return the resulting User struct
		return nil
	})
	if err != nil {
		return User{}, fmt.Errorf("failed to insert user: %w", err)
	}
	return user, nil

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
