package main

import (
	"context"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"product-service/internal/auth"
	"product-service/internal/consul"
	"product-service/internal/stores/kafka"

	"fmt"
	"github.com/joho/godotenv"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"product-service/handlers"

	"product-service/internal/products"
	"product-service/internal/stores/postgres"
	"syscall"
	"time"
)

func main() {
	setupSlog()
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	err = startApp()
	if err != nil {
		panic(err)
	}
}

func startApp() error {

	/*
			//------------------------------------------------------//
		                Setting up DB & Migrating tables
			//------------------------------------------------------//
	*/

	slog.Info("Migrating tables for product-service if not already done")
	db, err := postgres.OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = postgres.RunMigration(db)
	if err != nil {
		return err
	}
	/*
		//------------------------------------------------------//
		//    Setting up product package config
		//------------------------------------------------------//
	*/
	p, err := products.NewConf(db)
	if err != nil {
		return err
	}

	/*
		//------------------------------------------------------//
		//  Setting up Auth layer
		//------------------------------------------------------//
	*/

	slog.Info("main : Started : Initializing authentication support")
	publicPEM, err := os.ReadFile("pubkey.pem")
	if err != nil {
		return fmt.Errorf("reading auth public key %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicPEM)
	if err != nil {
		return fmt.Errorf("parsing auth public key %w", err)
	}

	k, err := auth.NewKeys(publicKey)
	if err != nil {
		return fmt.Errorf("initializing auth %w", err)
	}

	/*
		/*
			//------------------------------------------------------//
			//   Consuming Kafka TOPICS [ORDER SERVICE EVENTS]
			//------------------------------------------------------//
	*/
	go func() {
		ch := make(chan kafka.ConsumeResult)
		go kafka.ConsumeMessage(context.Background(), kafka.TopicOrderPaid, kafka.ConsumerGroup, ch)
		for v := range ch {
			if v.Err != nil {
				fmt.Println(v.Err)
				continue
			}
			fmt.Printf("Consumed message: %s", string(v.Record.Value))
			var event kafka.OrderPaidEvent
			json.Unmarshal(v.Record.Value, &event)
			// create a method over internal/products to decrement the stock value by quantity
			err := p.DecrementStock(context.Background(), event.ProductId, event.Quantity)
			if err != nil {
				slog.Error("error decrementing the stock", slog.Any("error", err.Error()))
				continue
			}
			slog.Info("product sold successfully and decremented the stock")

		}
	}()

	/*
			//------------------------------------------------------//
		                Setting up http Server
			//------------------------------------------------------//
	*/
	// Initialize http service
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "80"
	}
	prefix := os.Getenv("SERVICE_ENDPOINT_PREFIX")
	if prefix == "" {
		return fmt.Errorf("SERVICE_ENDPOINT_PREFIX env variable is not set")
	}
	api := http.Server{
		Addr:         ":" + port,
		ReadTimeout:  8000 * time.Second,
		WriteTimeout: 800 * time.Second,
		IdleTimeout:  800 * time.Second,
		//handlers.API returns gin.Engine which implements Handler Interface
		Handler: handlers.API(p, prefix, k),
	}

	// channel to store any errors while setting up the service
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- api.ListenAndServe()
	}()

	/*
			//------------------------------------------------------//
		               Registering with Consul
			//------------------------------------------------------//
	*/

	consulClient, regId, err := consul.RegisterWithConsul()
	if err != nil {
		return err
	}

	defer consulClient.Agent().ServiceDeregister(regId)

	/*
			//------------------------------------------------------//
		               Listening for error signals
			//------------------------------------------------------//
	*/
	//shutdown channel intercepts ctrl+c signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, os.Kill)
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error %w", err)
	case <-shutdown:
		fmt.Println("de-registering from consul", err)
		fmt.Println("graceful shutdown")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		//Shutdown gracefully shuts down the server without interrupting any active connections.
		//Shutdown works by first closing all open listeners, then closing all idle connections,
		err := api.Shutdown(ctx)
		if err != nil {
			err := api.Close()
			if err != nil {
				return fmt.Errorf("could not stop server gracefully %w", err)
			}
		}
	}
	return nil

}

func setupSlog() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		//AddSource: true: This will cause the source file and line number of the log message to be included in the output
		AddSource: true,
	})

	logger := slog.New(logHandler)
	//SetDefault makes l the default Logger. in our case we would be doing structured logging
	slog.SetDefault(logger)
}
