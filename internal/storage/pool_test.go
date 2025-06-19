package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_ConnectionPool(t *testing.T) {
	// Open database with connection pool settings
	db, err := sql.Open("sqlite3", ":memory:?_busy_timeout=5000")
	require.NoError(t, err)
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Test concurrent operations
	ctx := context.Background()
	done := make(chan bool)
	errs := make(chan error, 10)

	// Start 10 goroutines to test concurrent access
	for i := 0; i < 10; i++ {
		go func(id int64) {
			// Create user
			err := storage.CreateUser(ctx, id, "test@example.com", time.Hour)
			if err != nil {
				errs <- err
				return
			}

			// Get user
			user, err := storage.GetUser(ctx, id)
			if err != nil {
				errs <- err
				return
			}

			// Update user
			err = storage.UpdateUser(ctx, user.TelegramID, time.Hour*2)
			if err != nil {
				errs <- err
				return
			}

			done <- true
		}(int64(i))
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case err := <-errs:
			t.Errorf("Error in goroutine: %v", err)
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for goroutine")
		}
	}

	// Verify all users were created
	for i := 0; i < 10; i++ {
		user, err := storage.GetUser(ctx, int64(i))
		require.NoError(t, err)
		assert.Equal(t, time.Hour*2, user.DigestInterval)
	}
}

func TestSQLiteStorage_ConnectionTimeout(t *testing.T) {
	// Open database with short busy timeout
	db, err := sql.Open("sqlite3", ":memory:?_busy_timeout=100")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Start a long-running transaction
	tx1, err := storage.BeginTx(context.Background())
	require.NoError(t, err)

	// Create user in first transaction
	err = tx1.CreateUser(1, "test@example.com", time.Hour)
	require.NoError(t, err)

	// Try to access the same data from another connection
	// This should timeout due to the lock
	go func() {
		time.Sleep(50 * time.Millisecond) // Give the first transaction time to acquire the lock
		err := storage.CreateUser(context.Background(), 1, "test@example.com", time.Hour)
		assert.Error(t, err) // Should fail due to timeout or unique constraint
	}()

	time.Sleep(200 * time.Millisecond)
	err = tx1.Commit()
	require.NoError(t, err)
}

func TestSQLiteStorage_ConnectionPoolExhaustion(t *testing.T) {
	// Open database with limited connections
	db, err := sql.Open("sqlite3", ":memory:?_busy_timeout=1000")
	require.NoError(t, err)
	defer db.Close()

	// Set very low connection limits
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Start multiple transactions to exhaust the pool
	tx1, err := storage.BeginTx(context.Background())
	require.NoError(t, err)
	defer tx1.Rollback()

	tx2, err := storage.BeginTx(context.Background())
	require.NoError(t, err)
	defer tx2.Rollback()

	// This should block or fail due to no available connections
	done := make(chan bool)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := storage.BeginTx(ctx)
		assert.Error(t, err) // Should fail due to context timeout
		done <- true
	}()

	select {
	case <-done:
		// Success - operation failed as expected
	case <-time.After(time.Second):
		t.Error("Operation did not timeout as expected")
	}
} 