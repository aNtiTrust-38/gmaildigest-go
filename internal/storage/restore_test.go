package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_Restore(t *testing.T) {
	// Create temporary directory for test databases
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sourceDBPath := filepath.Join(tmpDir, "source.db")
	backupPath := filepath.Join(tmpDir, "backup.db")
	restoreDBPath := filepath.Join(tmpDir, "restore.db")

	// Create and populate source database
	sourceDB, err := sql.Open("sqlite3", sourceDBPath)
	require.NoError(t, err)
	defer sourceDB.Close()

	sourceStorage := NewSQLiteStorage(sourceDB)
	err = sourceStorage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	telegramID := int64(1)
	gmailUserID := "test@example.com"

	// Create test data
	err = sourceStorage.CreateUser(ctx, telegramID, gmailUserID, time.Hour)
	require.NoError(t, err)

	err = sourceStorage.StoreToken(ctx, gmailUserID, []byte("token"), []byte("nonce"))
	require.NoError(t, err)

	err = sourceStorage.MarkEmailProcessed(ctx, "msg1", gmailUserID)
	require.NoError(t, err)

	// Create backup
	err = sourceStorage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Create restore database
	restoreDB, err := sql.Open("sqlite3", restoreDBPath)
	require.NoError(t, err)
	defer restoreDB.Close()

	restoreStorage := NewSQLiteStorage(restoreDB)
	err = restoreStorage.Migrate(context.Background())
	require.NoError(t, err)

	// Restore from backup
	err = restoreStorage.Restore(ctx, backupPath)
	require.NoError(t, err)

	// Verify restored data
	user, err := restoreStorage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, gmailUserID, user.GmailUserID)

	token, nonce, err := restoreStorage.GetToken(ctx, gmailUserID)
	require.NoError(t, err)
	assert.Equal(t, []byte("token"), token)
	assert.Equal(t, []byte("nonce"), nonce)

	processed, err := restoreStorage.IsEmailProcessed(ctx, "msg1", gmailUserID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestSQLiteStorage_RestoreWithExistingData(t *testing.T) {
	// Create temporary directory for test databases
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sourceDBPath := filepath.Join(tmpDir, "source.db")
	backupPath := filepath.Join(tmpDir, "backup.db")
	restoreDBPath := filepath.Join(tmpDir, "restore.db")

	// Create and populate source database
	sourceDB, err := sql.Open("sqlite3", sourceDBPath)
	require.NoError(t, err)
	defer sourceDB.Close()

	sourceStorage := NewSQLiteStorage(sourceDB)
	err = sourceStorage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Create test data in source
	err = sourceStorage.CreateUser(ctx, 1, "user1@example.com", time.Hour)
	require.NoError(t, err)

	// Create backup
	err = sourceStorage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Create restore database with existing data
	restoreDB, err := sql.Open("sqlite3", restoreDBPath)
	require.NoError(t, err)
	defer restoreDB.Close()

	restoreStorage := NewSQLiteStorage(restoreDB)
	err = restoreStorage.Migrate(context.Background())
	require.NoError(t, err)

	// Add some data to restore database
	err = restoreStorage.CreateUser(ctx, 2, "user2@example.com", time.Hour)
	require.NoError(t, err)

	// Restore from backup (should replace existing data)
	err = restoreStorage.Restore(ctx, backupPath)
	require.NoError(t, err)

	// Verify only backup data exists
	_, err = restoreStorage.GetUser(ctx, 1)
	assert.NoError(t, err)

	_, err = restoreStorage.GetUser(ctx, 2)
	assert.Error(t, err)
}

func TestSQLiteStorage_RestoreFailure(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Try to restore from non-existent backup
	err = storage.Restore(context.Background(), "/nonexistent/path/backup.db")
	assert.Error(t, err)
}

func TestSQLiteStorage_RestoreWithConcurrentOperations(t *testing.T) {
	// Create temporary directory for test databases
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sourceDBPath := filepath.Join(tmpDir, "source.db")
	backupPath := filepath.Join(tmpDir, "backup.db")
	restoreDBPath := filepath.Join(tmpDir, "restore.db")

	// Create and populate source database
	sourceDB, err := sql.Open("sqlite3", sourceDBPath)
	require.NoError(t, err)
	defer sourceDB.Close()

	sourceStorage := NewSQLiteStorage(sourceDB)
	err = sourceStorage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Create test data in source
	for i := 0; i < 10; i++ {
		err = sourceStorage.CreateUser(ctx, int64(i), "test@example.com", time.Hour)
		require.NoError(t, err)
	}

	// Create backup
	err = sourceStorage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Create restore database
	restoreDB, err := sql.Open("sqlite3", restoreDBPath)
	require.NoError(t, err)
	defer restoreDB.Close()

	restoreStorage := NewSQLiteStorage(restoreDB)
	err = restoreStorage.Migrate(context.Background())
	require.NoError(t, err)

	// Start goroutines to perform concurrent operations
	done := make(chan bool)
	errs := make(chan error, 10)

	for i := 10; i < 20; i++ {
		go func(id int64) {
			err := restoreStorage.CreateUser(ctx, id, "test@example.com", time.Hour)
			if err != nil {
				errs <- err
				return
			}
			done <- true
		}(int64(i))
	}

	// Restore from backup while operations are running
	time.Sleep(10 * time.Millisecond) // Give some time for operations to start
	err = restoreStorage.Restore(ctx, backupPath)
	require.NoError(t, err)

	// Wait for all operations to complete or fail
	for i := 0; i < 10; i++ {
		select {
		case err := <-errs:
			// Operations should fail due to restore
			assert.Error(t, err)
		case <-done:
			// Operations should not succeed
			t.Error("Concurrent operation succeeded during restore")
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for goroutine")
		}
	}

	// Verify only backup data exists
	var count int64
	err = restoreDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Verify all users from backup exist
	for i := 0; i < 10; i++ {
		user, err := restoreStorage.GetUser(ctx, int64(i))
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.GmailUserID)
	}

	// Verify no additional users exist
	for i := 10; i < 20; i++ {
		_, err := restoreStorage.GetUser(ctx, int64(i))
		assert.Error(t, err)
	}
} 