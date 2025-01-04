package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"product-service/handlers"
	"product-service/internal/consul"
	"product-service/internal/products"
	"product-service/internal/stores/postgres"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(".env"); err != nil {
		panic("Error loading .env file")
	}
	setupSlog()

	db, err := setUpDB()
	if err != nil {
		slog.Error("Failed to set up database", slog.Any("error", err))
		panic(err)
	}
	defer db.Close()

	p, err := products.NewConf(db)
	if err != nil {
		slog.Error("Failed to create products configuration", slog.Any("error", err))
		panic(err)
	}
	consulClient, regId, err := consul.RegisterWithConsul()
	if err != nil {
		slog.Error("Failed to register with Consul", slog.Any("error", err))
		panic(err)
	}
	defer func() {
		if err := consulClient.Agent().ServiceDeregister(regId); err != nil {
			slog.Error("Failed to deregister from Consul", slog.Any("error", err))
		}
	}()
	setUpServer(p)
}

func setUpDB() (*sql.DB, error) {
	db, err := postgres.OpenDB()
	if err != nil {
		return nil, err
	}
	if err := postgres.RunMigrations(db); err != nil {
		return nil, err
	}
	return db, nil
}

func setUpServer(p *products.Conf) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	api := http.Server{
		Addr:         ":" + port,
		ReadTimeout:  8000 * time.Second,
		WriteTimeout: 800 * time.Second,
		IdleTimeout:  800 * time.Second,
		Handler:      handlers.API(p),
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- api.ListenAndServe()
	}()

	// Graceful shutdown setup
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, os.Kill)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-shutdown:
		fmt.Println("Shutting down server gracefully")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := api.Shutdown(ctx); err != nil {
			fmt.Println("Graceful shutdown failed, forcing shutdown")

			// Attempt forceful shutdown
			if closeErr := api.Close(); closeErr != nil {
				return fmt.Errorf("could not stop server gracefully or forcefully: %w", closeErr)
			}

			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}
	return nil
}

func setupSlog() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
}
