package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockTask implements the Task interface for testing
type mockTask struct {
	mu            sync.Mutex
	executed      bool
	shouldFail    bool
	successCalled bool
	failureCalled bool
	delay         time.Duration
}

func (t *mockTask) Execute(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.executed = true
	if t.delay > 0 {
		time.Sleep(t.delay)
	}
	if t.shouldFail {
		return errors.New("task failed")
	}
	return nil
}

func (t *mockTask) OnSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.successCalled = true
}

func (t *mockTask) OnFailure(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failureCalled = true
}

func TestWorkerPool_Basic(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	// Test successful task
	task1 := &mockTask{}
	if !pool.Submit(task1) {
		t.Error("Failed to submit task1")
	}

	time.Sleep(100 * time.Millisecond) // Give time for execution

	task1.mu.Lock()
	if !task1.executed {
		t.Error("Task1 was not executed")
	}
	if !task1.successCalled {
		t.Error("OnSuccess was not called for task1")
	}
	if task1.failureCalled {
		t.Error("OnFailure was incorrectly called for task1")
	}
	task1.mu.Unlock()

	// Test failing task
	task2 := &mockTask{shouldFail: true}
	if !pool.Submit(task2) {
		t.Error("Failed to submit task2")
	}

	time.Sleep(100 * time.Millisecond) // Give time for execution

	task2.mu.Lock()
	if !task2.executed {
		t.Error("Task2 was not executed")
	}
	if task2.successCalled {
		t.Error("OnSuccess was incorrectly called for task2")
	}
	if !task2.failureCalled {
		t.Error("OnFailure was not called for task2")
	}
	task2.mu.Unlock()
}

func TestWorkerPool_Metrics(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Start()
	defer pool.Stop()

	// Submit successful task
	task1 := &mockTask{}
	pool.Submit(task1)

	time.Sleep(100 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.completedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", metrics.completedTasks)
	}
	if metrics.failedTasks != 0 {
		t.Errorf("Expected 0 failed tasks, got %d", metrics.failedTasks)
	}

	// Submit failing task
	task2 := &mockTask{shouldFail: true}
	pool.Submit(task2)

	time.Sleep(100 * time.Millisecond)

	metrics = pool.GetMetrics()
	if metrics.completedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", metrics.completedTasks)
	}
	if metrics.failedTasks != 1 {
		t.Errorf("Expected 1 failed task, got %d", metrics.failedTasks)
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	pool := NewWorkerPool(1) // 1 worker, queue size = 2
	pool.Start()
	defer pool.Stop()

	// Fill the queue
	task1 := &mockTask{delay: 100 * time.Millisecond}
	task2 := &mockTask{delay: 100 * time.Millisecond}
	task3 := &mockTask{delay: 100 * time.Millisecond}

	if !pool.Submit(task1) {
		t.Error("Failed to submit task1")
	}
	if !pool.Submit(task2) {
		t.Error("Failed to submit task2")
	}

	// This should fail as queue is full
	if pool.Submit(task3) {
		t.Error("Task3 should not have been accepted")
	}
}

func TestWorkerPool_Shutdown(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()

	// Submit some tasks
	task1 := &mockTask{delay: 50 * time.Millisecond}
	task2 := &mockTask{delay: 50 * time.Millisecond}

	if !pool.Submit(task1) {
		t.Error("Failed to submit task1")
	}
	if !pool.Submit(task2) {
		t.Error("Failed to submit task2")
	}

	// Wait for tasks to start executing
	time.Sleep(25 * time.Millisecond)

	// Stop the pool and wait for tasks to complete
	pool.Stop()

	// Verify tasks were completed
	task1.mu.Lock()
	if !task1.executed {
		t.Error("Task1 was not executed before shutdown")
	}
	task1.mu.Unlock()

	task2.mu.Lock()
	if !task2.executed {
		t.Error("Task2 was not executed before shutdown")
	}
	task2.mu.Unlock()

	// Try to submit after shutdown
	task3 := &mockTask{}
	if pool.Submit(task3) {
		t.Error("Should not accept tasks after shutdown")
	}
} 