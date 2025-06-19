package scheduler

import (
	"testing"
	"context"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"gmaildigest-go/internal/worker"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

// Test: Scheduler initialization
func TestScheduler_NewScheduler(t *testing.T) {
	// Setup in-memory SQLite DB
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	pool := worker.NewWorkerPool(1)
	scheduler, err := NewScheduler(ctx, db, pool)
	require.NoError(t, err)
	require.NotNil(t, scheduler)
}

// Test: Job scheduling and execution
func TestScheduler_ScheduleJob(t *testing.T) {
	// Setup in-memory SQLite DB
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	pool := worker.NewWorkerPool(1)
	scheduler, err := NewScheduler(ctx, db, pool)
	require.NoError(t, err)

	// Schedule a job
	payload := map[string]string{"test": "value"}
	job, err := scheduler.ScheduleJob("user1", "test", "* * * * *", payload)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "user1", job.UserID)
	assert.Equal(t, "test", job.Type)
	assert.Equal(t, "* * * * *", job.Schedule)
}

// Test: Recurring job handling
func TestScheduler_RecurringJobs(t *testing.T) {
	// TODO: Test that recurring jobs are executed at the correct intervals
}

// Test: Graceful shutdown
func TestScheduler_GracefulShutdown(t *testing.T) {
	// TODO: Test that the scheduler can shut down gracefully, finishing in-flight jobs
}

// Test: Job persistence and recovery
func TestScheduler_JobPersistence(t *testing.T) {
	// TODO: Test that scheduled jobs are persisted and recovered after restart
}

// Test: Error handling and retries
func TestScheduler_ErrorHandling(t *testing.T) {
	// TODO: Test that job errors are handled and retried according to policy
}

// Test: Token refresh background service
func TestTokenRefreshService_BackgroundRefresh(t *testing.T) {
	// TODO: Test that token refresh jobs run in the background and refresh tokens as needed
}

// Test: Job deduplication by type and user
func TestScheduler_JobDeduplication(t *testing.T) {
	// TODO: Test that scheduling a job with the same type and user deduplicates (updates/reschedules) the job
}

// Test: Generic payload support
func TestScheduler_GenericPayloadSupport(t *testing.T) {
	// TODO: Test that jobs can accept and persist generic (e.g., JSON-encoded) payloads
}

// Test: Retry and dead letter handling
func TestScheduler_DeadLetterHandling(t *testing.T) {
	// TODO: Test that jobs are retried up to 10 times and then moved to a dead letter queue/failure routine
}

// Test: Scheduler dispatches jobs to WorkerPool
func TestScheduler_DispatchesJobsToWorkerPool(t *testing.T) {
	// Setup in-memory SQLite DB
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	pool := worker.NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	executed := make(chan struct{}, 1)
	scheduler, err := NewScheduler(ctx, db, pool)
	require.NoError(t, err)

	// Register a test handler
	scheduler.RegisterHandler("test", func(ctx context.Context, job *Job) error {
		executed <- struct{}{}
		return nil
	})

	// Start the scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Wait for the scheduler to start
	time.Sleep(100 * time.Millisecond)

	// Schedule a job due now
	payload := map[string]string{"test": "value"}
	job, err := scheduler.ScheduleJob("user1", "test", "* * * * *", payload)
	require.NoError(t, err)

	// Set the job's next run time to now
	job.NextRun = time.Now()
	err = scheduler.store.UpdateJob(ctx, job)
	require.NoError(t, err)
	scheduler.signalCronWakeup()

	// Wait for execution
	select {
	case <-executed:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("job was not executed by worker pool")
	}
}

func TestScheduler_RegisterTokenRefreshHandler(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	pool := worker.NewWorkerPool(1)
	pool.Start()
	defer pool.Stop()

	scheduler, err := NewScheduler(ctx, db, pool)
	require.NoError(t, err)

	// Register a token refresh handler
	handlerCalled := false
	scheduler.RegisterTokenRefreshHandler(func(ctx context.Context, job *Job) error {
		handlerCalled = true
		return nil
	})

	// Start the scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Wait for the scheduler to start
	time.Sleep(100 * time.Millisecond)

	// Schedule a token refresh job
	payload := TokenRefreshPayload{UserID: "user1"}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	job, err := scheduler.ScheduleJob("user1", "token_refresh", "*/5 * * * *", json.RawMessage(payloadBytes))
	require.NoError(t, err)

	// Set the job's next run time to now
	job.NextRun = time.Now()
	err = scheduler.store.UpdateJob(ctx, job)
	require.NoError(t, err)
	scheduler.signalCronWakeup()

	// Wait for the job to be executed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, handlerCalled)
}
