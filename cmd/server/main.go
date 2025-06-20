package main

import (
	"context"
	"gmaildigest-go/internal/app"
	"gmaildigest-go/internal/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetPrefix("gmaildigest: ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.LoadFromFile("./configs/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create a new application instance
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		<-sigchan
		log.Println("Shutdown signal received, initiating graceful shutdown...")
		if err := application.Stop(ctx); err != nil {
			log.Printf("Error during graceful shutdown: %v", err)
		}
		cancel()
	}()

	// Start the application
	if err := application.Start(ctx); err != nil {
		log.Fatalf("Application failed to start: %v", err)
	}

	log.Println("Application has stopped.")
} 