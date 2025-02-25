package main

import (
	"cart-service/internal/auth"
	"cart-service/internal/consul"
	"context"
	"net"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"cart-service/handlers"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	pb "cart-service/gen/proto"

	"cart-service/internal/cart"
	"cart-service/internal/stores/postgres"
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
	slog.Info("Migrating tables for cart-service if not already done")
	db, err := postgres.OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = postgres.RunMigration(db)
	if err != nil {
		return err
	}
	/*******  Setting up cart package config  *******/
	cConf, err := cart.NewConf(db)
	if err != nil {
		return err
	}

	/*******   Setting up Auth layer  *******/

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

	/**************** Setting Up the Kakfa *************/

	/***** Registering with Consul ******/

	consulClient, regId, err := consul.RegisterWithConsul()
	if err != nil {
		return err
	}

	defer consulClient.Agent().ServiceDeregister(regId)

	/****** Setting up http Server *******/

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
		Handler: handlers.API(prefix, a, consulClient, cConf),
	}

	// channel to store any errors while setting up the service
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- api.ListenAndServe()
	}()

	/***** startup GRPC server ****/
	grpcErrors := make(chan error)

	go func() {
		listener, err := net.Listen("tcp", ":5001")

		//send error to channel

		if err != nil {
			grpcErrors <- err // Send error to the channel
			return
		}

		//NewServer creates a gRPC server which has no service registered
		// creating an instance of the server
		s := grpc.NewServer()

		pb.RegisterCartItemServiceServer(s, handlers.NewCartItemServiceHandler(cConf))

		//exposing gRPC service to be tested by postman
		reflection.Register(s)

		// Start serving requests
		if err := s.Serve(listener); err != nil {
			grpcErrors <- err // Send error to the channel
		}
	}()

	/*****  Listening for error signals  ******/

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
	case err := <-grpcErrors: // handling error
		return fmt.Errorf("GRPC error %w", err)
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
