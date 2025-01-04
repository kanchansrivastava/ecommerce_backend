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
	driverName    = "pgx" // Database driver
	migrationsDir = "internal/stores/postgres/migrations"
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

	db, err := sql.Open(driverName, psqlInfo)
	if err != nil {
		return nil, err
	}

	var backoff time.Duration = 2
	for i := 0; i < 8; i++ {
		err = db.Ping()
		if err != nil {
			fmt.Println("postgres not ready yet")
			time.Sleep(backoff * time.Second)
			backoff += 2
			continue
		}
		break
	}

	if err != nil {
		return nil, err
	}

	return db, nil

}

func RunMigrations(db *sql.DB) error {

	// Set the dialect for Goose (PostgreSQL in this case)
	err := goose.SetDialect(driverName)
	if err != nil {
		return err
	}

	// Apply all pending migrations in the directory
	err = goose.Up(db, migrationsDir)
	if err != nil {
		return err
	}
	return nil
}
