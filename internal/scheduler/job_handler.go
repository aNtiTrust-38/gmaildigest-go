package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobHandler is a function that handles a specific type of job
type JobHandler func(ctx context.Context, job *Job) error

// JobHandlerRegistry manages job type to handler mappings
type JobHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
}

// NewJobHandlerRegistry creates a new job handler registry
func NewJobHandlerRegistry() *JobHandlerRegistry {
	return &JobHandlerRegistry{
		handlers: make(map[string]JobHandler),
	}
}

// RegisterHandler registers a handler for a job type
func (r *JobHandlerRegistry) RegisterHandler(jobType string, handler JobHandler) {
	if jobType == "" || handler == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = handler
}

// GetHandler returns the handler for a job type
func (r *JobHandlerRegistry) GetHandler(jobType string) JobHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.handlers[jobType]
}

// UnregisterHandler removes a handler for a job type
func (r *JobHandlerRegistry) UnregisterHandler(jobType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, jobType)
}

// ListHandlerTypes returns a list of registered job types
func (r *JobHandlerRegistry) ListHandlerTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.handlers))
	for jobType := range r.handlers {
		types = append(types, jobType)
	}
	return types
}

// JobTask wraps a Job to implement the worker.Task interface
type JobTask struct {
	ctx       context.Context
	job       *Job
	registry  *JobHandlerRegistry
	scheduler *Scheduler
}

// NewJobTask creates a new JobTask
func NewJobTask(ctx context.Context, job *Job, registry *JobHandlerRegistry) *JobTask {
	return &JobTask{
		ctx:      ctx,
		job:      job,
		registry: registry,
	}
}

// Execute implements the worker.Task interface
func (t *JobTask) Execute(ctx context.Context) error {
	if t.job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	handler := t.registry.GetHandler(t.job.Type)
	if handler == nil {
		return fmt.Errorf("no handler registered for job type: %s", t.job.Type)
	}

	return handler(ctx, t.job)
}

// OnSuccess implements the worker.Task interface
func (t *JobTask) OnSuccess() {
	if t.scheduler == nil {
		return
	}

	t.scheduler.JobMu.Lock()
	defer t.scheduler.JobMu.Unlock()

	// Update job status
	t.job.Status = JobStatusCompleted
	t.job.LastError = ""
	t.job.RetryCount = 0

	// Calculate next run time based on schedule
	t.job.NextRun = t.scheduler.nextRunTime(t.job.Schedule)

	// Persist changes
	if err := t.scheduler.store.UpdateJob(t.ctx, t.job); err != nil {
		// Log error but continue
		fmt.Printf("Failed to update job status: %v\n", err)
	}

	// Update in-memory job
	t.scheduler.Jobs[t.job.ID] = t.job
	t.scheduler.signalCronWakeup()
}

// OnFailure implements the worker.Task interface
func (t *JobTask) OnFailure(err error) {
	if t.scheduler == nil {
		return
	}

	t.scheduler.JobMu.Lock()
	defer t.scheduler.JobMu.Unlock()

	// Update job status
	t.job.Status = JobStatusFailed
	t.job.LastError = err.Error()
	t.job.RetryCount++

	// Calculate retry delay using exponential backoff
	delay := time.Duration(t.job.RetryCount*t.job.RetryCount) * time.Minute
	if delay > 24*time.Hour {
		delay = 24 * time.Hour
	}
	t.job.NextRun = time.Now().Add(delay)

	// Check if max retries exceeded
	if t.job.RetryCount >= 5 { // Max 5 retries
		t.job.Status = JobStatusFailed
		t.job.NextRun = time.Time{} // Zero time indicates no more retries
	}

	// Persist changes
	if err := t.scheduler.store.UpdateJob(t.ctx, t.job); err != nil {
		// Log error but continue
		fmt.Printf("Failed to update job status: %v\n", err)
	}

	// Update in-memory job
	t.scheduler.Jobs[t.job.ID] = t.job
	t.scheduler.signalCronWakeup()
} 