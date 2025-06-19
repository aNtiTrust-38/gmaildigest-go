package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusDead      JobStatus = "dead"
)

// Job represents a scheduled task in the system
type Job struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	Type       string          `json:"type"`
	Schedule   string          `json:"schedule"`
	Payload    json.RawMessage `json:"payload"`
	Status     JobStatus       `json:"status"`
	RetryCount int            `json:"retry_count"`
	LastError  string         `json:"last_error,omitempty"`
	NextRun    time.Time      `json:"next_run"`
	LastRun    *time.Time     `json:"last_run,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// JobStore defines the interface for job persistence operations
type JobStore interface {
	// Initialize sets up the database schema
	Initialize(ctx context.Context) error

	// CreateJob creates a new job
	CreateJob(ctx context.Context, job *Job) error

	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, id string) (*Job, error)

	// UpdateJob updates an existing job
	UpdateJob(ctx context.Context, job *Job) error

	// ListJobs returns all jobs matching the given criteria
	ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error)

	// DeleteJob deletes a job by ID
	DeleteJob(ctx context.Context, id string) error
}

// JobFilter defines criteria for listing jobs
type JobFilter struct {
	UserID   string    `json:"user_id,omitempty"`
	Type     string    `json:"type,omitempty"`
	Status   JobStatus `json:"status,omitempty"`
	NextRun  time.Time `json:"next_run,omitempty"`
	Statuses []JobStatus `json:"statuses,omitempty"`
}

// ListJobsOptions represents the options for listing jobs
type ListJobsOptions struct {
	Type   string    `json:"type,omitempty"`
	UserID string    `json:"user_id,omitempty"`
	Status JobStatus `json:"status,omitempty"`
	Before time.Time `json:"before,omitempty"`
	After  time.Time `json:"after,omitempty"`
	Limit  int       `json:"limit,omitempty"`
}

// SQLiteJobStore implements JobStore using SQLite
type SQLiteJobStore struct {
	db *sql.DB
}

// NewSQLiteJobStore creates a new SQLite-backed job store
func NewSQLiteJobStore(db *sql.DB) *SQLiteJobStore {
	return &SQLiteJobStore{db: db}
}

// Initialize implements JobStore
func (s *SQLiteJobStore) Initialize(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		schedule TEXT NOT NULL,
		payload TEXT NOT NULL,
		status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'dead')),
		retry_count INTEGER NOT NULL DEFAULT 0,
		last_error TEXT,
		next_run DATETIME NOT NULL,
		last_run DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, type, schedule)
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_next_run ON jobs(next_run) WHERE status = 'pending';
	CREATE INDEX IF NOT EXISTS idx_jobs_user ON jobs(user_id);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// CreateJob implements JobStore
func (s *SQLiteJobStore) CreateJob(ctx context.Context, job *Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.Status == "" {
		job.Status = JobStatusPending
	}
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now

	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	query := `
	INSERT INTO jobs (
		id, user_id, type, schedule, payload, status,
		retry_count, last_error, next_run, last_run,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		job.ID, job.UserID, job.Type, job.Schedule, string(payload),
		job.Status, job.RetryCount, job.LastError, job.NextRun, job.LastRun,
		job.CreatedAt, job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

// GetJob implements JobStore
func (s *SQLiteJobStore) GetJob(ctx context.Context, id string) (*Job, error) {
	query := `SELECT * FROM jobs WHERE id = ?`
	return s.queryJob(ctx, query, id)
}

// UpdateJob implements JobStore
func (s *SQLiteJobStore) UpdateJob(ctx context.Context, job *Job) error {
	job.UpdatedAt = time.Now().UTC()
	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	query := `
	UPDATE jobs SET
		user_id = ?, type = ?, schedule = ?, payload = ?,
		status = ?, retry_count = ?, last_error = ?,
		next_run = ?, last_run = ?, updated_at = ?
	WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		job.UserID, job.Type, job.Schedule, string(payload),
		job.Status, job.RetryCount, job.LastError,
		job.NextRun, job.LastRun, job.UpdatedAt,
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", job.ID)
	}
	return nil
}

// ListJobs implements JobStore
func (s *SQLiteJobStore) ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error) {
	var conditions []string
	var args []interface{}

	if filter.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, filter.UserID)
	}
	if filter.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, filter.Type)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			placeholders[i] = "?"
			args = append(args, status)
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", 
			strings.Join(placeholders, ",")))
	}
	if !filter.NextRun.IsZero() {
		conditions = append(conditions, "next_run >= ?")
		args = append(args, filter.NextRun)
	}

	query := "SELECT * FROM jobs"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY next_run ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := s.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return jobs, nil
}

// DeleteJob implements JobStore
func (s *SQLiteJobStore) DeleteJob(ctx context.Context, id string) error {
	query := `DELETE FROM jobs WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", id)
	}
	return nil
}

// scanJob scans a row into a Job struct
func (s *SQLiteJobStore) scanJob(rows *sql.Rows) (*Job, error) {
	var job Job
	var payloadStr string
	err := rows.Scan(
		&job.ID, &job.UserID, &job.Type, &job.Schedule,
		&payloadStr, &job.Status, &job.RetryCount, &job.LastError,
		&job.NextRun, &job.LastRun, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan job: %w", err)
	}

	if err := json.Unmarshal([]byte(payloadStr), &job.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	return &job, nil
}

// queryJob executes a query that returns a single job
func (s *SQLiteJobStore) queryJob(ctx context.Context, query string, args ...interface{}) (*Job, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query job: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("job not found")
	}

	job, err := s.scanJob(rows)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return job, nil
} 