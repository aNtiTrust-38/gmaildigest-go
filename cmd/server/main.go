package main

import (
	"context"
	"database/sql"
	"gmaildigest-go/internal/scheduler"
	"gmaildigest-go/internal/storage"
	"gmaildigest-go/internal/worker"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Setup: Logger
	logger := log.New(os.Stdout, "gmaildigest: ", log.LstdFlags)

	// Setup: Database
	db, err := sql.Open("sqlite3", "./gmaildigest.db")
	if err != nil {
		logger.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := storage.NewSQLiteStorage(db).Migrate(context.Background()); err != nil {
		logger.Fatalf("Failed to run migrations: %v", err)
	}
	logger.Println("Database migrated successfully.")

	// Setup: WorkerPool
	pool := worker.NewWorkerPool(5) // 5 concurrent workers
	pool.Start()
	logger.Println("Worker pool started.")

	// Setup: Scheduler
	sched, err := scheduler.NewScheduler(context.Background(), db, pool)
	if err != nil {
		logger.Fatalf("Failed to create scheduler: %v", err)
	}
	sched.Start()
	logger.Println("Scheduler started.")

	// TODO: Setup TokenRefreshService and other job services here
	// For now, the scheduler is running but has no job handlers registered
	// other than the ones it might register itself (like token_refresh).

	// Setup and start the metrics server
	logger.Println("Starting metrics server on :8082")
	http.Handle("/metrics", promhttp.Handler())
	httpServer := &http.Server{
		Addr:    ":8082",
		Handler: nil, // Use DefaultServeMux
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Println("Shutting down gracefully...")

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Printf("HTTP server shutdown error: %v", err)
	}

	// Shutdown scheduler and worker pool
	sched.Stop()
	pool.Stop()

	logger.Println("Shutdown complete.")
} 