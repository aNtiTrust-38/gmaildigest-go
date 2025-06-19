package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "gmail_digest.db", cfg.Path)
	assert.Equal(t, 10, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, time.Hour, cfg.ConnMaxLifetime)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxIdleTime)
	assert.Equal(t, 5*time.Second, cfg.BusyTimeout)
}

func TestOpenDatabase(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Test with default configuration
	cfg := DefaultConfig()
	cfg.Path = dbPath

	storage, err := OpenDatabase(cfg)
	require.NoError(t, err)
	defer storage.Close()

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Test database operations
	err = storage.CreateUser(context.Background(), 1, "test@example.com", time.Hour)
	require.NoError(t, err)

	user, err := storage.GetUser(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.GmailUserID)
}

func TestOpenDatabase_InvalidPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Path = "/nonexistent/directory/test.db"

	_, err := OpenDatabase(cfg)
	assert.Error(t, err)
}

func TestOpenDatabase_CustomConfig(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Test with custom configuration
	cfg := Config{
		Path:            dbPath,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: 30 * time.Second,
		BusyTimeout:     time.Second,
	}

	storage, err := OpenDatabase(cfg)
	require.NoError(t, err)
	defer storage.Close()

	// Test concurrent access with limited connections
	done := make(chan bool)
	errs := make(chan error, 3)

	// Start 3 goroutines to test connection pool limits
	for i := 0; i < 3; i++ {
		go func(id int64) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := storage.CreateUser(ctx, id, "test@example.com", time.Hour)
			if err != nil {
				errs <- err
				return
			}
			done <- true
		}(int64(i))
	}

	// Wait for goroutines with timeout
	timeout := time.After(5 * time.Second)
	completed := 0

	for completed < 3 {
		select {
		case err := <-errs:
			// Some errors are expected due to connection limits
			t.Logf("Got error: %v", err)
		case <-done:
			completed++
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}
}

func TestOpenDatabase_Reconnect(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := DefaultConfig()
	cfg.Path = dbPath

	// Open first connection
	storage1, err := OpenDatabase(cfg)
	require.NoError(t, err)

	// Create some data
	err = storage1.CreateUser(context.Background(), 1, "test@example.com", time.Hour)
	require.NoError(t, err)

	// Close first connection
	err = storage1.Close()
	require.NoError(t, err)

	// Open second connection to same database
	storage2, err := OpenDatabase(cfg)
	require.NoError(t, err)
	defer storage2.Close()

	// Verify data persisted
	user, err := storage2.GetUser(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.GmailUserID)
} 