package worker

import (
	"context"
	"sync"
)

// Task represents a unit of work for the worker pool
// Now returns an error from Process() for retry logic
//
type Task interface {
	Process() error
}

// WorkerPool manages a pool of worker goroutines
// and a queue of tasks to process
//
type WorkerPool struct {
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	workers     int
	resizeMu    sync.Mutex
	tasks       chan Task // buffered channel for tasks
	queueCap    int      // capacity of the task queue
	deadLetter  []Task
	deadLetterMu sync.Mutex
	maxRetries  int
}

// PoolStats holds monitoring information about the worker pool
//
type PoolStats struct {
	ActiveWorkers int
	QueueLength   int
	DeadLetters   int
}

// NewWorkerPool creates a new WorkerPool with the given number of workers and queue capacity
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	queueCap := 10 // default queue size
	return &WorkerPool{
		ctx:        ctx,
		cancel:     cancel,
		workers:    workers,
		tasks:      make(chan Task, queueCap),
		queueCap:   queueCap,
		maxRetries: 10,
		deadLetter: make([]Task, 0),
	}
}

// Start launches the worker goroutines
func (p *WorkerPool) Start() {
	p.resizeMu.Lock()
	defer p.resizeMu.Unlock()
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.workerLoop()
	}
}

// Stop signals all workers to exit and waits for them to finish
func (p *WorkerPool) Stop() {
	p.cancel()
	close(p.tasks)
	p.wg.Wait()
}

// Resize changes the number of worker goroutines (not implemented yet)
func (p *WorkerPool) Resize(newSize int) {
	// TODO: Implement dynamic resizing
}

// Submit adds a task to the queue, returns false if the queue is full
func (p *WorkerPool) Submit(task Task) bool {
	select {
	case p.tasks <- task:
		return true
	default:
		return false // backpressure: queue is full
	}
}

// workerLoop is the main loop for each worker goroutine
func (p *WorkerPool) workerLoop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			p.processWithRetry(task, 0)
		}
	}
}

// processWithRetry processes a task, retrying up to maxRetries, then moves to dead letter
func (p *WorkerPool) processWithRetry(task Task, attempt int) {
	for attempt < p.maxRetries {
		select {
		case <-p.ctx.Done():
			return
		default:
			if err := task.Process(); err != nil {
				attempt++
				continue
			}
			return
		}
	}
	p.deadLetterMu.Lock()
	p.deadLetter = append(p.deadLetter, task)
	p.deadLetterMu.Unlock()
}

// DeadLetterCount returns the number of tasks in the dead letter queue
func (p *WorkerPool) DeadLetterCount() int {
	p.deadLetterMu.Lock()
	defer p.deadLetterMu.Unlock()
	return len(p.deadLetter)
}

// Workers returns the number of worker goroutines
func (p *WorkerPool) Workers() int {
	return p.workers
}

// Stats returns current statistics about the worker pool
func (p *WorkerPool) Stats() PoolStats {
	return PoolStats{
		ActiveWorkers: p.workers, // static for now; dynamic if resizing is implemented
		QueueLength:   len(p.tasks),
		DeadLetters:   p.DeadLetterCount(),
	}
} 