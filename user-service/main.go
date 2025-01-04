package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user-service/handlers"
	"user-service/internal/auth"
	"user-service/internal/consul"
	"user-service/internal/stores/kafka"
	"user-service/internal/stores/postgres"
	"user-service/internal/users"
)

func main() {

	setupSlog()
	err := godotenv.Load(".env")
	if err != nil {
		slog.Error("error in loading env file")
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

	slog.Info("Migrating tables for user-service if not already done")
	db, err := postgres.OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = postgres.RunMigrations(db)
	if err != nil {
		return err
	}
	//------------------------------------------------------//
	/*

			//------------------------------------------------------//
		                Setting up Auth layer
			//------------------------------------------------------//
	*/
	slog.Info("main : Started : Initializing authentication support")
	privatePEM, err := os.ReadFile("private.pem")
	if err != nil {
		return fmt.Errorf("reading auth private key %w", err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return fmt.Errorf("parsing auth private key %w", err)
	}

	publicPEM, err := os.ReadFile("pubkey.pem")
	if err != nil {
		return fmt.Errorf("reading auth public key %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicPEM)
	if err != nil {
		return fmt.Errorf("parsing auth public key %w", err)
	}

	a, err := auth.NewKeys(privateKey, publicKey)
	if err != nil {
		return fmt.Errorf("constructing auth %w", err)
	}

	/*
		//------------------------------------------------------//
		//    Setting up users package config
		//------------------------------------------------------//
	*/
	u, err := users.NewConf(db)
	if err != nil {
		return err
	}
	//------------------------------------------------------//

	/*
			//------------------------------------------------------//
		                Setting up Kafka & Creating topics
			//------------------------------------------------------//
	*/

	kafkaConf, err := kafka.NewConf(kafka.TopicAccountCreated, kafka.ConsumerGroup)
	if err != nil {
		return err
	}

	fmt.Println("kafka conf", kafkaConf)
	fmt.Println("connected to kafka")
	//------------------------------------------------------//

	/*
			//------------------------------------------------------//
		                Consuming Kafka topics
			//------------------------------------------------------//
	*/

	// Start a goroutine to handle Kafka message consumption
	go func() {
		// Create a channel to receive messages of type `kafka.ConsumeResult`
		ch := make(chan kafka.ConsumeResult)

		// Start a goroutine to consume messages from Kafka
		// This function `ConsumeMessage` does the work of fetching messages from a Kafka topic
		// and pushing them into the `ch` channel.
		go kafkaConf.ConsumeMessage(context.Background(), ch)

		// Iterate over the channel `ch` to process messages as they are received
		// The loop continues until the application stops
		for v := range ch {
			//  message that has been consumed from Kafka
			fmt.Printf("Consumed message: %s\n", string(v.Record.Value))

			// Declare a variable of type `kafka.MSGUserServiceAccountCreated` to unmarshal the message body
			var event kafka.MSGUserServiceAccountCreated

			// Unmarshal the JSON message into the `event` struct.
			// This converts the Kafka JSON message (v.Record.Value) into a Go struct to make it easier to work with.
			json.Unmarshal(v.Record.Value, &event)

			// Log/Print the event data after successfully unmarshaling
			fmt.Printf("Successfully received the event : %+v\n", event)
			// The below method would create the customer over stripe and add it to database
			err := u.CreateCustomerStripe(context.Background(), event.ID, event.Name, event.Email)
			if err != nil {
				slog.Error("error creating customer", slog.Any("error", err))
				continue
			}
			slog.Info("customer created successfully on stripe")

		}
	}()

	/*

			//------------------------------------------------------//
		                Setting up http Server
			//------------------------------------------------------//
	*/
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	api := http.Server{
		Addr:         ":" + port,
		ReadTimeout:  8000 * time.Second,
		WriteTimeout: 800 * time.Second,
		IdleTimeout:  800 * time.Second,
		//handlers.API returns gin.Engine which implements Handler Interface
		Handler: handlers.API(u, a, kafkaConf),
	}
	serverErrors := make(chan error)
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

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, os.Kill)
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error %w", err)
	case <-shutdown:

		fmt.Println("Shutting down server gracefully")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		//Shutdown gracefully shuts down the server without interrupting any active connections.
		//Shutdown works by first closing all open listeners, then closing all idle connections,
		err = api.Shutdown(ctx)
		if err != nil {

			//forceful closure
			err := api.Close()
			if err != nil {
				// returning error to main if everything fails, the main would panic
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
