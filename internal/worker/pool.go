package worker

import (
	"context"
	"sync"
)

// Task represents a unit of work for the worker pool
// (Stub for now; will be expanded later)
type Task interface{}

// WorkerPool manages a pool of worker goroutines
// and a queue of tasks to process
//
type WorkerPool struct {
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	workers  int
	resizeMu sync.Mutex
	tasks    chan Task // buffered channel for tasks
	queueCap int      // capacity of the task queue
}

// NewWorkerPool creates a new WorkerPool with the given number of workers and queue capacity
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	queueCap := 10 // default queue size
	return &WorkerPool{
		ctx:      ctx,
		cancel:   cancel,
		workers:  workers,
		tasks:    make(chan Task, queueCap),
		queueCap: queueCap,
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
		case <-p.tasks:
			// In a full implementation, process the task here
		}
	}
}

// Workers returns the number of worker goroutines
func (p *WorkerPool) Workers() int {
	return p.workers
} 