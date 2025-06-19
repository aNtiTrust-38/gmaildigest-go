package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"gmaildigest-go/internal/worker"
)

// Scheduler manages job scheduling, deduplication, and persistence
type Scheduler struct {
	store      JobStore
	Jobs       map[string]*Job // jobID -> Job (exported for testing)
	JobMu      sync.Mutex      // exported for testing
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	cronWakeup chan struct{}
	pool       *worker.WorkerPool
	registry   *JobHandlerRegistry
}

// NewScheduler creates a new Scheduler and loads jobs from the database
func NewScheduler(ctx context.Context, db *sql.DB, pool *worker.WorkerPool) (*Scheduler, error) {
	cctx, cancel := context.WithCancel(ctx)
	store := NewSQLiteJobStore(db)
	if err := store.Initialize(cctx); err != nil {
		cancel()
		return nil, err
	}

	s := &Scheduler{
		store:      store,
		Jobs:       make(map[string]*Job),
		ctx:        cctx,
		cancel:     cancel,
		cronWakeup: make(chan struct{}, 1),
		pool:       pool,
		registry:   NewJobHandlerRegistry(),
	}
	if err := s.loadJobsFromDB(); err != nil {
		cancel()
		return nil, err
	}
	return s, nil
}

// loadJobsFromDB loads persisted jobs into memory
func (s *Scheduler) loadJobsFromDB() error {
	jobs, err := s.store.ListJobs(s.ctx, JobFilter{})
	if err != nil {
		return err
	}
	for _, job := range jobs {
		s.Jobs[job.ID] = job
	}
	return nil
}

// ScheduleJob schedules a new job or deduplicates if one exists for user/type/schedule
func (s *Scheduler) ScheduleJob(userID, jobType, schedule string, payload interface{}) (*Job, error) {
	s.JobMu.Lock()
	defer s.JobMu.Unlock()

	// Convert payload to JSON
	var payloadJSON json.RawMessage
	if p, ok := payload.(json.RawMessage); ok {
		payloadJSON = p
	} else {
		var err error
		payloadJSON, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	// Deduplication: check for existing job
	for _, job := range s.Jobs {
		if job.UserID == userID && job.Type == jobType && job.Schedule == schedule {
			// Update payload and reset status
			job.Payload = payloadJSON
			job.Status = JobStatusPending
			job.RetryCount = 0
			job.NextRun = s.nextRunTime(schedule)
			if err := s.store.UpdateJob(s.ctx, job); err != nil {
				return nil, err
			}
			s.signalCronWakeup()
			return job, nil
		}
	}

	// New job
	nextRun := s.nextRunTime(schedule)
	job := &Job{
		UserID:   userID,
		Type:     jobType,
		Schedule: schedule,
		Payload:  payloadJSON,
		Status:   JobStatusPending,
		NextRun:  nextRun,
	}

	if err := s.store.CreateJob(s.ctx, job); err != nil {
		return nil, err
	}

	s.Jobs[job.ID] = job
	s.signalCronWakeup()
	return job, nil
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

// ForceCheck manually triggers the scheduler to re-evaluate jobs.
// This is primarily useful for testing.
func (s *Scheduler) ForceCheck() {
	s.signalCronWakeup()
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
	for id, job := range s.Jobs {
		if job.Status == JobStatusPending && !job.NextRun.After(now) {
			jt := NewJobTask(s.ctx, job, s.registry)
			jt.scheduler = s // Set the scheduler
			ok := s.pool.Submit(jt)
			if ok {
				job.Status = JobStatusRunning
				job.LastRun = &now
				if err := s.store.UpdateJob(s.ctx, job); err != nil {
					// Log error but continue with other jobs
					continue
				}
				s.Jobs[id] = job // Update job in memory
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
		if job.Status == JobStatusPending && job.NextRun.Before(next) {
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

// RegisterTokenRefreshHandler registers the token refresh handler with the scheduler
func (s *Scheduler) RegisterTokenRefreshHandler(handler JobHandler) {
	s.registry.RegisterHandler("token_refresh", handler)
}

// RegisterHandler registers a handler function for a job type
func (s *Scheduler) RegisterHandler(jobType string, handler JobHandler) {
	s.registry.RegisterHandler(jobType, handler)
}

// ListJobs returns a list of jobs matching the given options
func (s *Scheduler) ListJobs(ctx context.Context, opts *ListJobsOptions) ([]*Job, error) {
	if opts == nil {
		opts = &ListJobsOptions{}
	}

	filter := JobFilter{
		UserID: opts.UserID,
		Type:   opts.Type,
		Status: opts.Status,
	}

	if !opts.Before.IsZero() {
		filter.NextRun = opts.Before
	}

	jobs, err := s.store.ListJobs(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply additional filtering
	var filtered []*Job
	for _, job := range jobs {
		if !opts.After.IsZero() && job.NextRun.Before(opts.After) {
			continue
		}
		filtered = append(filtered, job)
	}

	// Apply limit
	if opts.Limit > 0 && len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
	}

	return filtered, nil
} 