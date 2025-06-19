package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_CleanupProcessedEmails(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	userID := "test@example.com"

	// Create test data
	messages := []string{"msg1", "msg2", "msg3"}
	for _, msgID := range messages {
		err = storage.MarkEmailProcessed(ctx, msgID, userID)
		require.NoError(t, err)
	}

	// Manually update processed_at timestamps
	_, err = db.Exec(`
		UPDATE processed_emails 
		SET processed_at = datetime('now', '-8 days')
		WHERE message_id IN ('msg1', 'msg2')
	`)
	require.NoError(t, err)

	// Run cleanup
	deleted, err := storage.CleanupProcessedEmails(ctx, 7*24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify old records were deleted
	for _, msgID := range messages[:2] {
		processed, err := storage.IsEmailProcessed(ctx, msgID, userID)
		require.NoError(t, err)
		assert.False(t, processed)
	}

	// Verify recent record remains
	processed, err := storage.IsEmailProcessed(ctx, messages[2], userID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestSQLiteStorage_CleanupInvalidTokens(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Create test users with tokens
	users := []struct {
		telegramID  int64
		gmailUserID string
		tokenValid  bool
	}{
		{1, "user1@example.com", false},
		{2, "user2@example.com", true},
		{3, "user3@example.com", false},
	}

	for _, u := range users {
		err = storage.CreateUser(ctx, u.telegramID, u.gmailUserID, time.Hour)
		require.NoError(t, err)

		err = storage.StoreToken(ctx, u.gmailUserID, []byte("token"), []byte("nonce"))
		require.NoError(t, err)

		if !u.tokenValid {
			_, err = db.Exec("UPDATE users SET google_token_valid = ? WHERE telegram_id = ?", false, u.telegramID)
			require.NoError(t, err)
		}
	}

	// Run cleanup
	deleted, err := storage.CleanupInvalidTokens(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify invalid tokens were deleted
	for _, u := range users {
		_, _, err := storage.GetToken(ctx, u.gmailUserID)
		if u.tokenValid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestSQLiteStorage_CleanupInactiveUsers(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Create test users
	users := []struct {
		telegramID  int64
		gmailUserID string
		lastActive  string
	}{
		{1, "user1@example.com", "now"},
		{2, "user2@example.com", "-31 days"},
		{3, "user3@example.com", "-32 days"},
	}

	for _, u := range users {
		err = storage.CreateUser(ctx, u.telegramID, u.gmailUserID, time.Hour)
		require.NoError(t, err)

		if u.lastActive != "now" {
			_, err = db.Exec(`
				UPDATE users 
				SET updated_at = datetime('now', ?)
				WHERE telegram_id = ?`,
				u.lastActive, u.telegramID)
			require.NoError(t, err)
		}
	}

	// Run cleanup
	deleted, err := storage.CleanupInactiveUsers(ctx, 30*24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify inactive users were deleted
	for _, u := range users {
		_, err := storage.GetUser(ctx, u.telegramID)
		if u.lastActive == "now" {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestSQLiteStorage_CleanupWithTransaction(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	userID := "test@example.com"

	// Create test data
	err = storage.MarkEmailProcessed(ctx, "msg1", userID)
	require.NoError(t, err)

	// Update timestamp
	_, err = db.Exec(`
		UPDATE processed_emails 
		SET processed_at = datetime('now', '-8 days')
	`)
	require.NoError(t, err)

	// Start transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Run cleanup within transaction
	deleted, err := tx.CleanupProcessedEmails(7 * 24 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify record still exists before commit
	processed, err := storage.IsEmailProcessed(ctx, "msg1", userID)
	require.NoError(t, err)
	assert.True(t, processed)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify record was deleted after commit
	processed, err = storage.IsEmailProcessed(ctx, "msg1", userID)
	require.NoError(t, err)
	assert.False(t, processed)
}

func TestSQLiteStorage_CleanupWithRollback(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	userID := "test@example.com"

	// Create test data
	err = storage.MarkEmailProcessed(ctx, "msg1", userID)
	require.NoError(t, err)

	// Update timestamp
	_, err = db.Exec(`
		UPDATE processed_emails 
		SET processed_at = datetime('now', '-8 days')
	`)
	require.NoError(t, err)

	// Start transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Run cleanup within transaction
	deleted, err := tx.CleanupProcessedEmails(7 * 24 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify record still exists after rollback
	processed, err := storage.IsEmailProcessed(ctx, "msg1", userID)
	require.NoError(t, err)
	assert.True(t, processed)
} 