package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/arijanluiken/mercantile/internal/supervisor"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
		log.Printf("Continuing with system environment variables...")
	} else {
		log.Println("Successfully loaded .env file")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the supervisor actor system
	supervisorActor := supervisor.New()
	if err := supervisorActor.Start(ctx); err != nil {
		log.Fatalf("Failed to start supervisor: %v", err)
	}

	// Wait for interrupt signal to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down trading bot...")
	cancel()
}