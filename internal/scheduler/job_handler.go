package scheduler

import (
	"context"
	"fmt"
)

// JobHandler is a function that processes a job
type JobHandler func(ctx context.Context, job *Job) error

// JobHandlerRegistry maintains a map of job type to handler functions
type JobHandlerRegistry struct {
	handlers map[string]JobHandler
}

// NewJobHandlerRegistry creates a new job handler registry
func NewJobHandlerRegistry() *JobHandlerRegistry {
	return &JobHandlerRegistry{
		handlers: make(map[string]JobHandler),
	}
}

// RegisterHandler registers a handler function for a job type
func (r *JobHandlerRegistry) RegisterHandler(jobType string, handler JobHandler) {
	r.handlers[jobType] = handler
}

// GetHandler returns the handler function for a job type
func (r *JobHandlerRegistry) GetHandler(jobType string) (JobHandler, error) {
	handler, ok := r.handlers[jobType]
	if !ok {
		return nil, fmt.Errorf("no handler registered for job type: %s", jobType)
	}
	return handler, nil
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

// Process implements the worker.Task interface
func (t *JobTask) Process() error {
	handler, err := t.registry.GetHandler(t.job.Type)
	if err != nil {
		t.job.Status = JobStatusFailed
		t.job.LastError = fmt.Sprintf("failed to get handler: %v", err)
		t.job.RetryCount++
		if t.job.RetryCount >= 10 {
			t.job.Status = JobStatusDead
		}
		return fmt.Errorf("failed to get handler for job %s: %w", t.job.Type, err)
	}

	err = handler(t.ctx, t.job)
	if err != nil {
		t.job.Status = JobStatusFailed
		t.job.LastError = err.Error()
		t.job.RetryCount++
		if t.job.RetryCount >= 10 {
			t.job.Status = JobStatusDead
		}
	} else {
		t.job.Status = JobStatusCompleted
		t.job.LastError = ""
		t.job.RetryCount = 0
		t.job.NextRun = t.scheduler.nextRunTime(t.job.Schedule)
	}

	// Update job in store and memory
	if t.scheduler != nil {
		t.scheduler.JobMu.Lock()
		defer t.scheduler.JobMu.Unlock()

		if err := t.scheduler.store.UpdateJob(t.ctx, t.job); err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
		t.scheduler.Jobs[t.job.ID] = t.job
	}

	return err
} 