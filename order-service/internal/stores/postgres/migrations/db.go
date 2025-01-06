package postgres

import (
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"os"
	"time"
)

const (
	driverName = "pgx" // Database driver
	// Connection string
	// write a complete relative path from the root of the project
	migrationsDir = `internal/stores/postgres/migrations` // Directory where the migration files are stored
)

func OpenDB() (*sql.DB, error) {
	var (
		host     = os.Getenv("POSTGRES_HOST")
		port     = os.Getenv("POSTGRES_PORT")
		user     = os.Getenv("POSTGRES_USER")
		password = os.Getenv("POSTGRES_PASSWORD")
		dbname   = os.Getenv("POSTGRES_DATABASE")
	)

	//sql.Open(psqlInfo)
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	var (
		db  *sql.DB
		err error
	)
	db, err = sql.Open(driverName, psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stores: %w", err)
	}
	var backoff time.Duration = 2
	for i := 0; i < 8; i++ {
		// almost 8 min minimum wait before stopping service
		// Open a connection to the stores
		err = db.Ping()
		if err != nil {
			fmt.Println("postgres not ready yet")
			time.Sleep(backoff * time.Second)
			backoff *= 2
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stores: %w", err)
	}
	return db, nil
}

func RunMigration(db *sql.DB) error {
	// Set the dialect for Goose (PostgreSQL in this case)
	err := goose.SetDialect(driverName)
	if err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Apply all pending migrations in the directory
	err = goose.Up(db, migrationsDir)
	//err = goose.UpTo(stores, migrationsDir, 1)

	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil

}
