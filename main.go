package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/arijanluiken/mercantile/pkg/config"
)

func main() {
	fmt.Println("Mercantile Trading Bot starting...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Database path: %s\n", cfg.Database.Path)
	fmt.Printf("API port: %d\n", cfg.API.Port)
	fmt.Printf("UI port: %d\n", cfg.UI.Port)
	fmt.Printf("Log level: %s\n", cfg.Logging.Level)

	// TODO: Start actor system here when Hollywood API is figured out
	fmt.Println("Trading bot would start here...")
	fmt.Println("Press Ctrl+C to exit")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down trading bot...")
}