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

func TestSQLiteStorage_Backup(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	backupPath := filepath.Join(tmpDir, "backup.db")

	// Create and populate source database
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	telegramID := int64(1)
	gmailUserID := "test@example.com"

	// Create test data
	err = storage.CreateUser(ctx, telegramID, gmailUserID, time.Hour)
	require.NoError(t, err)

	err = storage.StoreToken(ctx, gmailUserID, []byte("token"), []byte("nonce"))
	require.NoError(t, err)

	err = storage.MarkEmailProcessed(ctx, "msg1", gmailUserID)
	require.NoError(t, err)

	// Create backup
	err = storage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Verify backup file exists
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)

	// Open backup database
	backupDB, err := sql.Open("sqlite3", backupPath)
	require.NoError(t, err)
	defer backupDB.Close()

	backupStorage := NewSQLiteStorage(backupDB)

	// Verify data in backup
	user, err := backupStorage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, gmailUserID, user.GmailUserID)

	token, nonce, err := backupStorage.GetToken(ctx, gmailUserID)
	require.NoError(t, err)
	assert.Equal(t, []byte("token"), token)
	assert.Equal(t, []byte("nonce"), nonce)

	processed, err := backupStorage.IsEmailProcessed(ctx, "msg1", gmailUserID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestSQLiteStorage_BackupWithTransaction(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	backupPath := filepath.Join(tmpDir, "backup.db")

	// Create and populate source database
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	telegramID := int64(1)
	gmailUserID := "test@example.com"

	// Start transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Create test data in transaction
	err = tx.CreateUser(telegramID, gmailUserID, time.Hour)
	require.NoError(t, err)

	// Create backup before commit
	err = storage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Open backup database
	backupDB, err := sql.Open("sqlite3", backupPath)
	require.NoError(t, err)
	defer backupDB.Close()

	backupStorage := NewSQLiteStorage(backupDB)

	// Verify data not in backup (transaction not committed)
	_, err = backupStorage.GetUser(ctx, telegramID)
	assert.Error(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Create new backup after commit
	backupPath2 := filepath.Join(tmpDir, "backup2.db")
	err = storage.Backup(ctx, backupPath2)
	require.NoError(t, err)

	// Open second backup database
	backupDB2, err := sql.Open("sqlite3", backupPath2)
	require.NoError(t, err)
	defer backupDB2.Close()

	backupStorage2 := NewSQLiteStorage(backupDB2)

	// Verify data in second backup (transaction committed)
	user, err := backupStorage2.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, gmailUserID, user.GmailUserID)
}

func TestSQLiteStorage_BackupFailure(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create source database
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Try to backup to invalid path
	err = storage.Backup(context.Background(), "/nonexistent/path/backup.db")
	assert.Error(t, err)
}

func TestSQLiteStorage_BackupWithConcurrentOperations(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gmail_digest_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	backupPath := filepath.Join(tmpDir, "backup.db")

	// Create source database
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	done := make(chan bool)
	errs := make(chan error, 10)

	// Start goroutines to perform concurrent operations
	for i := 0; i < 10; i++ {
		go func(id int64) {
			err := storage.CreateUser(ctx, id, "test@example.com", time.Hour)
			if err != nil {
				errs <- err
				return
			}
			done <- true
		}(int64(i))
	}

	// Create backup while operations are running
	time.Sleep(10 * time.Millisecond) // Give some time for operations to start
	err = storage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Wait for all operations to complete
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

	// Open backup database
	backupDB, err := sql.Open("sqlite3", backupPath)
	require.NoError(t, err)
	defer backupDB.Close()

	backupStorage := NewSQLiteStorage(backupDB)

	// Verify backup contains consistent data
	var count int64
	err = backupDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	require.NoError(t, err)
	assert.True(t, count > 0)

	// Verify all successful operations are reflected in backup
	for i := 0; i < 10; i++ {
		user, err := backupStorage.GetUser(ctx, int64(i))
		if err == nil {
			assert.Equal(t, "test@example.com", user.GmailUserID)
		}
	}
} 