package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, JobStore) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	store := NewSQLiteJobStore(db)
	err = store.Initialize(context.Background())
	require.NoError(t, err)

	return db, store
}

func createTestJob(userID, jobType string) *Job {
	now := time.Now().UTC()
	return &Job{
		ID:       uuid.New().String(),
		UserID:   userID,
		Type:     jobType,
		Schedule: "*/5 * * * *",
		Payload:  json.RawMessage(`{"key":"value"}`),
		Status:   JobStatusPending,
		NextRun:  now.Add(5 * time.Minute),
	}
}

func TestSQLiteJobStore_Initialize(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// Test that we can initialize multiple times without error
	err := store.Initialize(context.Background())
	assert.NoError(t, err)
}

func TestSQLiteJobStore_CreateJob(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name    string
		job     *Job
		wantErr bool
	}{
		{
			name: "valid job",
			job:  createTestJob("user1", "test"),
		},
		{
			name: "duplicate job",
			job:  createTestJob("user1", "test"),
			wantErr: true,
		},
		{
			name: "different user same type",
			job:  createTestJob("user2", "test"),
		},
		{
			name: "same user different type",
			job:  createTestJob("user1", "test2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateJob(context.Background(), tt.job)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify job was created
			saved, err := store.GetJob(context.Background(), tt.job.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.job.UserID, saved.UserID)
			assert.Equal(t, tt.job.Type, saved.Type)
			assert.Equal(t, tt.job.Schedule, saved.Schedule)
			assert.Equal(t, tt.job.Status, saved.Status)
			assert.JSONEq(t, string(tt.job.Payload), string(saved.Payload))
		})
	}
}

func TestSQLiteJobStore_UpdateJob(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	job := createTestJob("user1", "test")
	err := store.CreateJob(context.Background(), job)
	require.NoError(t, err)

	// Update job status and retry count
	job.Status = JobStatusFailed
	job.RetryCount++
	job.LastError = "test error"
	err = store.UpdateJob(context.Background(), job)
	require.NoError(t, err)

	// Verify updates
	saved, err := store.GetJob(context.Background(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusFailed, saved.Status)
	assert.Equal(t, 1, saved.RetryCount)
	assert.Equal(t, "test error", saved.LastError)

	// Test updating non-existent job
	nonExistentJob := createTestJob("user2", "test2")
	err = store.UpdateJob(context.Background(), nonExistentJob)
	assert.Error(t, err)
}

func TestSQLiteJobStore_ListJobs(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// Create test jobs
	jobs := []*Job{
		createTestJob("user1", "type1"),
		createTestJob("user1", "type2"),
		createTestJob("user2", "type1"),
	}
	for _, job := range jobs {
		err := store.CreateJob(context.Background(), job)
		require.NoError(t, err)
	}

	// Test various filters
	tests := []struct {
		name       string
		filter     JobFilter
		wantCount  int
		checkFirst func(*testing.T, *Job)
	}{
		{
			name:      "no filter",
			filter:    JobFilter{},
			wantCount: 3,
		},
		{
			name: "filter by user",
			filter: JobFilter{
				UserID: "user1",
			},
			wantCount: 2,
			checkFirst: func(t *testing.T, job *Job) {
				assert.Equal(t, "user1", job.UserID)
			},
		},
		{
			name: "filter by type",
			filter: JobFilter{
				Type: "type1",
			},
			wantCount: 2,
			checkFirst: func(t *testing.T, job *Job) {
				assert.Equal(t, "type1", job.Type)
			},
		},
		{
			name: "filter by status",
			filter: JobFilter{
				Status: JobStatusPending,
			},
			wantCount: 3,
		},
		{
			name: "filter by next run",
			filter: JobFilter{
				NextRun: time.Now().Add(10 * time.Minute),
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.ListJobs(context.Background(), tt.filter)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount)
			if tt.checkFirst != nil && len(got) > 0 {
				tt.checkFirst(t, got[0])
			}
		})
	}
}

func TestSQLiteJobStore_DeleteJob(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// Create a test job
	job := createTestJob("user1", "test")
	err := store.CreateJob(context.Background(), job)
	require.NoError(t, err)

	// Delete the job
	err = store.DeleteJob(context.Background(), job.ID)
	require.NoError(t, err)

	// Verify job was deleted
	_, err = store.GetJob(context.Background(), job.ID)
	assert.Error(t, err)

	// Test deleting non-existent job
	err = store.DeleteJob(context.Background(), "non-existent")
	assert.Error(t, err)
}

func TestSQLiteJobStore_DeadLetterHandling(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// Create a test job
	job := createTestJob("user1", "test")
	err := store.CreateJob(context.Background(), job)
	require.NoError(t, err)

	// Simulate multiple retries until dead
	for i := 0; i < 10; i++ {
		job.Status = JobStatusFailed
		job.RetryCount++
		job.LastError = "test error"
		err = store.UpdateJob(context.Background(), job)
		require.NoError(t, err)
	}

	// Mark as dead after max retries
	job.Status = JobStatusDead
	err = store.UpdateJob(context.Background(), job)
	require.NoError(t, err)

	// Verify job is in dead letter queue
	saved, err := store.GetJob(context.Background(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusDead, saved.Status)
	assert.Equal(t, 10, saved.RetryCount)

	// List dead jobs
	deadJobs, err := store.ListJobs(context.Background(), JobFilter{
		Status: JobStatusDead,
	})
	require.NoError(t, err)
	assert.Len(t, deadJobs, 1)
	assert.Equal(t, job.ID, deadJobs[0].ID)
}

// Test: Job persistence - saving jobs to database
func TestPersistence_SaveJobs(t *testing.T) {
	// TODO: Test that jobs are saved to the database correctly
}

// Test: Job persistence - loading jobs from database
func TestPersistence_LoadJobs(t *testing.T) {
	// TODO: Test that jobs are loaded from the database correctly
}

// Test: Job recovery after restart
func TestPersistence_JobRecovery(t *testing.T) {
	// TODO: Test that jobs are recovered and rescheduled after application restart
}

// Test: Persistence of deduplicated jobs
func TestPersistence_DeduplicatedJobs(t *testing.T) {
	// TODO: Test that deduplicated jobs are persisted and not duplicated in the database
}

// Test: Persistence of retry counters
func TestPersistence_JobRetryCounter(t *testing.T) {
	// TODO: Test that job retry counters are persisted and restored after restart
}

// Test: Persistence of generic payloads
func TestPersistence_GenericPayloads(t *testing.T) {
	// TODO: Test that generic job payloads are persisted and restored correctly
}

func TestPersistencePlaceholder(t *testing.T) {
	// TODO: Implement persistence tests
	t.Skip("Persistence tests not implemented yet")
}
