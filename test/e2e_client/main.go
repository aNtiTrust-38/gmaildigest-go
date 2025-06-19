package main

import (
	"context"
	"database/sql"
	"gmaildigest-go/internal/scheduler"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logger := log.New(os.Stdout, "e2e-client: ", log.LstdFlags)

	db, err := sql.Open("sqlite3", "./gmaildigest.db")
	if err != nil {
		logger.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// We don't need a real worker pool for this client, so we pass nil.
	sched, err := scheduler.NewScheduler(context.Background(), db, nil)
	if err != nil {
		logger.Fatalf("Failed to create scheduler: %v", err)
	}

	// The scheduler needs a handler for the job type we are about to schedule.
	// Even though the *server* is what will execute it, the client-side
	// scheduler instance needs to know about it to schedule it.
	sched.RegisterHandler("e2e_test_job", func(ctx context.Context, job *scheduler.Job) error {
		// This handler will not be executed by the client.
		return nil
	})

	logger.Println("Scheduling E2E test job...")
	_, err = sched.ScheduleJob("e2e-user", "e2e_test_job", "* * * * *", `{"test": "data"}`)
	if err != nil {
		logger.Fatalf("Failed to schedule job: %v", err)
	}

	logger.Println("Successfully scheduled E2E test job.")
} 