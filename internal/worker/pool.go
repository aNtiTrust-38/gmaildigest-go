package worker

import (
	"context"
	"sync"
	"time"
)

// Task represents a unit of work to be executed by the worker pool
type Task interface {
	Execute(ctx context.Context) error
	OnSuccess()
	OnFailure(err error)
}

// WorkerPool manages a pool of workers for executing tasks
type WorkerPool struct {
	workers    int
	tasks     chan Task
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	metrics   *Metrics
	isStopped bool
	mu        sync.RWMutex
}

// Metrics tracks worker pool statistics
type Metrics struct {
	mu               sync.RWMutex
	activeWorkers    int
	completedTasks   int64
	failedTasks      int64
	queuedTasks      int64
	processingTime   time.Duration
	lastProcessed    time.Time
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers: workers,
		tasks:   make(chan Task, workers*2), // Buffer size = 2x number of workers
		ctx:     ctx,
		cancel:  cancel,
		metrics: &Metrics{},
	}
}

// Start initializes and starts the worker pool
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker processes tasks from the task queue
func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			if task == nil {
				continue
			}

			p.metrics.mu.Lock()
			p.metrics.activeWorkers++
			p.metrics.queuedTasks--
			p.metrics.mu.Unlock()

			start := time.Now()
			err := task.Execute(p.ctx)
			duration := time.Since(start)

			p.metrics.mu.Lock()
			p.metrics.activeWorkers--
			p.metrics.processingTime += duration
			p.metrics.lastProcessed = time.Now()
			if err != nil {
				p.metrics.failedTasks++
				task.OnFailure(err)
			} else {
				p.metrics.completedTasks++
				task.OnSuccess()
			}
			p.metrics.mu.Unlock()
		}
	}
}

// Submit adds a task to the worker pool queue
func (p *WorkerPool) Submit(task Task) bool {
	if task == nil {
		return false
	}

	p.mu.RLock()
	if p.isStopped {
		p.mu.RUnlock()
		return false
	}
	p.mu.RUnlock()

	select {
	case p.tasks <- task:
		p.metrics.mu.Lock()
		p.metrics.queuedTasks++
		p.metrics.mu.Unlock()
		return true
	default:
		// Queue is full
		return false
	}
}

// Stop gracefully shuts down the worker pool
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	p.isStopped = true
	p.mu.Unlock()

	p.cancel()
	close(p.tasks)
	p.wg.Wait()
}

// GetMetrics returns a copy of the current metrics
func (p *WorkerPool) GetMetrics() Metrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return Metrics{
		activeWorkers:    p.metrics.activeWorkers,
		completedTasks:   p.metrics.completedTasks,
		failedTasks:      p.metrics.failedTasks,
		queuedTasks:      p.metrics.queuedTasks,
		processingTime:   p.metrics.processingTime,
		lastProcessed:    p.metrics.lastProcessed,
	}
}

// ResetMetrics resets all metrics to their initial values
func (p *WorkerPool) ResetMetrics() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.completedTasks = 0
	p.metrics.failedTasks = 0
	p.metrics.queuedTasks = 0
	p.metrics.processingTime = 0
	p.metrics.lastProcessed = time.Time{}
} 