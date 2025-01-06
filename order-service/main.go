package main

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"log/slog"
	"net/http"
	"order-service/handlers"
	"order-service/internal/auth"
	"order-service/internal/consul"
	"order-service/internal/orders"
	"order-service/internal/stores/kafka"
	postgres "order-service/internal/stores/postgres/migrations"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
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
	setupSlog()
	/*
			//------------------------------------------------------//
		                Setting up DB & Migrating tables
			//------------------------------------------------------//
	*/

	db, err := postgres.OpenDB()
	if err != nil {
		return err
	}
	err = postgres.RunMigration(db)
	if err != nil {
		return err
	}

	/*
		//------------------------------------------------------//
		//    Setting up orders package config
		//------------------------------------------------------//
	*/
	o, err := orders.NewConf(db)
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

	a, err := auth.NewKeys(publicKey)
	if err != nil {
		return fmt.Errorf("initializing auth %w", err)
	}

	/*
			//------------------------------------------------------//
		                Setting up Kafka & Creating topics
			//------------------------------------------------------//
	*/

	kafkaConf, err := kafka.NewConf(kafka.TopicOrderPaid, kafka.ConsumerGroup)
	if err != nil {
		return err
	}

	fmt.Println("kafka conf", kafkaConf)
	fmt.Println("connected to kafka")
	//------------------------------------------------------//

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

		Handler: handlers.API(prefix, a, consulClient, &o, kafkaConf),
	}
	serverErrors := make(chan error)
	go func() {
		serverErrors <- api.ListenAndServe()
	}()

	/*
			//------------------------------------------------------//
		               Listening for error signals
			//------------------------------------------------------//
	*/

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return err
	case <-shutdown:

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
