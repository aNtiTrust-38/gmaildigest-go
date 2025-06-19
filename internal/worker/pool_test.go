package worker_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"context"
	"gmaildigest-go/internal/worker"
	"sync"
	"time"
	"errors"
)

var (
	mu    sync.Mutex
	count int
)

type testTask struct{}

func (testTask) Process() error {
	mu.Lock()
	count++
	mu.Unlock()
	return nil
}

type retryTask struct {
	failCount int
	called    int
}

func (r *retryTask) Process() error {
	r.called++
	if r.called <= r.failCount {
		return errors.New("fail")
	}
	return nil
}

// Test: Worker pool processes jobs concurrently
func TestWorkerPool_ProcessJobs(t *testing.T) {
	count = 0 // reset before test
	pool := worker.NewWorkerPool(4)
	pool.Start()
	defer pool.Stop()

	numTasks := 10
	for i := 0; i < numTasks; i++ {
		ok := pool.Submit(testTask{})
		assert.True(t, ok, "Task should be accepted")
	}

	// Wait for all tasks to be processed
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	finalCount := count
	mu.Unlock()
	assert.Equal(t, numTasks, finalCount, "All tasks should be processed")
}

// Test: Worker pool retry logic
func TestWorkerPool_JobRetry(t *testing.T) {
	task := &retryTask{failCount: 3}
	pool := worker.NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	ok := pool.Submit(task)
	assert.True(t, ok, "Task should be accepted")

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 4, task.called, "Task should be retried 3 times then succeed")
}

// Test: Worker pool dead letter queue
func TestWorkerPool_DeadLetterQueue(t *testing.T) {
	task := &retryTask{failCount: 20}
	pool := worker.NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	ok := pool.Submit(task)
	assert.True(t, ok, "Task should be accepted")

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, pool.DeadLetterCount(), 1, "Task should be in dead letter queue after 10 retries")
}

// Test: Worker pool start, stop, and resize
func TestWorkerPool_Lifecycle(t *testing.T) {
	pool := worker.NewWorkerPool(2)
	assert.NotNil(t, pool)
	assert.Equal(t, 2, pool.Workers())
	// Start and stop should not panic
	pool.Start()
	pool.Stop()
}

// Test: Worker pool job submission and backpressure
func TestWorkerPool_JobSubmissionBackpressure(t *testing.T) {
	pool := worker.NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	successCount := 0
	for i := 0; i < 10; i++ {
		ok := pool.Submit(testTask{})
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 10, successCount, "Should accept up to queue capacity")
	ok := pool.Submit(testTask{})
	assert.False(t, ok, "Should not accept more than queue capacity (backpressure)")
}

// Test: Worker pool monitoring and stats
func TestWorkerPool_MonitoringStats(t *testing.T) {
	t.Skip("not implemented")
	_ = context.Background()
	assert.True(t, true)
}
