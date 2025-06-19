package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gmaildigest-go/internal/scheduler"
	"gmaildigest-go/internal/storage"
	"gmaildigest-go/internal/worker"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Run migrations
	store := storage.NewSQLiteStorage(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestSchedulerIntegration(t *testing.T) {
	// Setup: Database
	db, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Setup: WorkerPool
	pool := worker.NewWorkerPool(1)
	defer pool.Stop()

	// Setup: Scheduler
	sched, err := scheduler.NewScheduler(context.Background(), db, pool)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer sched.Stop()

	// Setup: Mock Job Handler
	handlerExecuted := make(chan bool, 1)
	mockHandler := func(ctx context.Context, job *scheduler.Job) error {
		handlerExecuted <- true
		return nil
	}
	sched.RegisterHandler("mock_job", mockHandler)

	// Start everything
	pool.Start()
	sched.Start()

	// Action: Schedule a job
	job, err := sched.ScheduleJob("test-user", "mock_job", "0 0 1 1 *", nil) // Far future schedule
	if err != nil {
		t.Fatalf("Failed to schedule job: %v", err)
	}

	// Manually set the job's next run time to the past to force immediate execution
	sched.JobMu.Lock()
	job.NextRun = time.Now().Add(-1 * time.Minute)
	sched.Jobs[job.ID] = job
	sched.JobMu.Unlock()

	// Force the scheduler to check for jobs now
	sched.ForceCheck()

	// Verification
	select {
	case <-handlerExecuted:
		// Success!
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out: handler was not executed")
	}

	// Give a moment for the job status to be updated in the database
	time.Sleep(100 * time.Millisecond)

	// Check job status in the database
	jobs, err := sched.ListJobs(context.Background(), &scheduler.ListJobsOptions{Type: "mock_job"})
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, found %d", len(jobs))
	}

	dbJob := jobs[0]
	if dbJob.Status != scheduler.JobStatusCompleted {
		t.Errorf("Expected job status to be '%s', got '%s'", scheduler.JobStatusCompleted, dbJob.Status)
	}
	// The next run should now be in the far future as per the original schedule
	if dbJob.NextRun.Before(time.Now().Add(24 * time.Hour)) {
		t.Errorf("Expected next run to be in the far future, but it was %v", dbJob.NextRun)
	}
}

func TestTokenRefreshService_HandleTokenRefresh_Integration(t *testing.T) {
	// Setup: Database
	db, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Setup: OAuth2 Mock Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"access_token": "new-test-access-token", "refresh_token": "new-test-refresh-token", "token_type": "Bearer", "expiry": "2099-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	// Setup: Config
	oauthConfig := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://localhost/auth",
			TokenURL: server.URL,
		},
	}

	// Setup: Storage
	sqliteStorage := storage.NewSQLiteStorage(db)
	tokenStore := storage.NewTokenStore(sqliteStorage)
	userID := "test-user-123"
	initialToken := &oauth2.Token{
		AccessToken:  "initial-access-token",
		RefreshToken: "initial-refresh-token",
		Expiry:       time.Now().Add(-1 * time.Hour), // Expired
	}
	if err := tokenStore.StoreToken(context.Background(), userID, initialToken); err != nil {
		t.Fatalf("Failed to store initial token: %v", err)
	}

	// Setup: Service (without a real scheduler)
	tokenRefreshService := &scheduler.TokenRefreshService{
		Storage: tokenStore,
		Config:  oauthConfig,
	}
	tokenRefreshService.SetClient(server.Client())

	// Action: Create and handle the job directly
	jobPayload, _ := json.Marshal(scheduler.TokenRefreshPayload{UserID: userID})
	job := &scheduler.Job{
		ID:      "test-job-1",
		Type:    "token_refresh",
		UserID:  userID,
		Payload: json.RawMessage(jobPayload),
	}

	err := tokenRefreshService.HandleTokenRefresh(context.Background(), job)
	if err != nil {
		t.Fatalf("HandleTokenRefresh failed: %v", err)
	}

	// Verification
	refreshedToken, err := tokenStore.GetToken(context.Background(), userID)
	if err != nil {
		t.Fatalf("Failed to get token after refresh: %v", err)
	}

	if refreshedToken.AccessToken == initialToken.AccessToken {
		t.Error("Expected access token to be refreshed, but it was not.")
	}
	if refreshedToken.AccessToken != "new-test-access-token" {
		t.Errorf("Expected new access token to be 'new-test-access-token', got '%s'", refreshedToken.AccessToken)
	}
} 