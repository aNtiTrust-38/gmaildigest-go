package scheduler

import (
	"context"
	"database/sql"
	"sync"
	"time"
	"github.com/google/uuid"
	"gmaildigest-go/internal/worker"
)

// Job represents a scheduled job (simplified for now)
type Job struct {
	ID         string
	UserID     string
	Type       string
	Schedule   string
	Payload    string
	Status     string
	RetryCount int
	NextRun    time.Time
	LastRun    time.Time
}

// JobTask wraps a Job and implements worker.Task
//
type JobTask struct {
	Job *Job
	Exec func(*Job) error // allows test injection
}

func (jt *JobTask) Process() error {
	if jt.Exec != nil {
		return jt.Exec(jt.Job)
	}
	// Default: mark job as executed (stub)
	return nil
}

// Scheduler manages job scheduling, deduplication, and persistence
type Scheduler struct {
	db         *sql.DB
	Jobs       map[string]*Job // jobID -> Job (exported for testing)
	JobMu      sync.Mutex      // exported for testing
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	cronWakeup chan struct{}
	pool       *worker.WorkerPool
}

// NewScheduler creates a new Scheduler and loads jobs from the database
func NewScheduler(ctx context.Context, db *sql.DB, pool *worker.WorkerPool) (*Scheduler, error) {
	cctx, cancel := context.WithCancel(ctx)
	s := &Scheduler{
		db:         db,
		Jobs:       make(map[string]*Job),
		ctx:        cctx,
		cancel:     cancel,
		cronWakeup: make(chan struct{}, 1),
		pool:       pool,
	}
	if err := s.loadJobsFromDB(); err != nil {
		return nil, err
	}
	return s, nil
}

// loadJobsFromDB loads persisted jobs into memory
func (s *Scheduler) loadJobsFromDB() error {
	rows, err := s.db.QueryContext(s.ctx, `SELECT id, user_id, type, schedule, payload, status, retry_count, next_run, last_run FROM jobs`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var j Job
		var nextRun, lastRun sql.NullTime
		if err := rows.Scan(&j.ID, &j.UserID, &j.Type, &j.Schedule, &j.Payload, &j.Status, &j.RetryCount, &nextRun, &lastRun); err != nil {
			return err
		}
		if nextRun.Valid {
			j.NextRun = nextRun.Time
		}
		if lastRun.Valid {
			j.LastRun = lastRun.Time
		}
		s.Jobs[j.ID] = &j
	}
	return nil
}

// ScheduleJob schedules a new job or deduplicates if one exists for user/type/schedule
func (s *Scheduler) ScheduleJob(userID, jobType, schedule, payload string) (*Job, error) {
	s.JobMu.Lock()
	defer s.JobMu.Unlock()
	// Deduplication: check for existing job
	for _, job := range s.Jobs {
		if job.UserID == userID && job.Type == jobType && job.Schedule == schedule {
			// Update payload and reset status
			job.Payload = payload
			job.Status = "scheduled"
			job.RetryCount = 0
			job.NextRun = s.nextRunTime(schedule)
			if err := s.persistJob(job); err != nil {
				return nil, err
			}
			s.signalCronWakeup()
			return job, nil
		}
	}
	// New job
	id := uuid.NewString()
	nextRun := s.nextRunTime(schedule)
	job := &Job{
		ID:       id,
		UserID:   userID,
		Type:     jobType,
		Schedule: schedule,
		Payload:  payload,
		Status:   "scheduled",
		NextRun:  nextRun,
	}
	s.Jobs[id] = job
	if err := s.persistJob(job); err != nil {
		return nil, err
	}
	s.signalCronWakeup()
	return job, nil
}

// persistJob inserts or updates a job in the database
func (s *Scheduler) persistJob(job *Job) error {
	_, err := s.db.ExecContext(s.ctx, `INSERT INTO jobs (id, user_id, type, schedule, payload, status, retry_count, next_run, last_run, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET payload=excluded.payload, status=excluded.status, retry_count=excluded.retry_count, next_run=excluded.next_run, last_run=excluded.last_run, updated_at=CURRENT_TIMESTAMP`,
		job.ID, job.UserID, job.Type, job.Schedule, job.Payload, job.Status, job.RetryCount, job.NextRun, job.LastRun)
	return err
}

// nextRunTime computes the next run time for a cron schedule
func (s *Scheduler) nextRunTime(schedule string) time.Time {
	cron, err := ParseCron(schedule)
	if err != nil {
		return time.Now().Add(time.Hour) // fallback: 1 hour later
	}
	return cron.Next(time.Now())
}

// signalCronWakeup notifies the scheduling loop to re-evaluate jobs
func (s *Scheduler) signalCronWakeup() {
	select {
	case s.cronWakeup <- struct{}{}:
	default:
	}
}

// Start begins the scheduling loop (does not execute jobs yet)
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.schedulingLoop()
}

// schedulingLoop waits for the next job and triggers execution
func (s *Scheduler) schedulingLoop() {
	defer s.wg.Done()
	for {
		next := s.findNextJobTime()
		timer := time.NewTimer(time.Until(next))
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Dispatch jobs due at 'next' to the WorkerPool
			s.dispatchDueJobs(next)
		case <-s.cronWakeup:
			timer.Stop()
			continue
		}
	}
}

// dispatchDueJobs submits all jobs due at or before 'now' to the WorkerPool
func (s *Scheduler) dispatchDueJobs(now time.Time) {
	s.JobMu.Lock()
	defer s.JobMu.Unlock()
	for _, job := range s.Jobs {
		if job.Status == "scheduled" && !job.NextRun.After(now) {
			jt := &JobTask{Job: job}
			ok := s.pool.Submit(jt)
			if ok {
				job.Status = "running"
				job.LastRun = now
				// Optionally: update job.NextRun for recurring jobs
				_ = s.persistJob(job)
			} else {
				// Backpressure: could not submit, reschedule or log
			}
		}
	}
}

// findNextJobTime finds the soonest NextRun among scheduled jobs
func (s *Scheduler) findNextJobTime() time.Time {
	s.JobMu.Lock()
	defer s.JobMu.Unlock()
	next := time.Now().Add(24 * time.Hour)
	for _, job := range s.Jobs {
		if job.Status == "scheduled" && job.NextRun.Before(next) {
			next = job.NextRun
		}
	}
	return next
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
} 