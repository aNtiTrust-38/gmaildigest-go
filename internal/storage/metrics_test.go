package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_GetMetrics(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Create test data
	users := []struct {
		telegramID  int64
		gmailUserID string
		tokenValid  bool
	}{
		{1, "user1@example.com", true},
		{2, "user2@example.com", false},
		{3, "user3@example.com", true},
	}

	for _, u := range users {
		err = storage.CreateUser(ctx, u.telegramID, u.gmailUserID, time.Hour)
		require.NoError(t, err)

		if u.tokenValid {
			err = storage.StoreToken(ctx, u.gmailUserID, []byte("token"), []byte("nonce"))
			require.NoError(t, err)
		}
	}

	// Mark some emails as processed
	for i := 0; i < 5; i++ {
		err = storage.MarkEmailProcessed(ctx, "msg1", users[0].gmailUserID)
		require.NoError(t, err)
		err = storage.MarkEmailProcessed(ctx, "msg2", users[2].gmailUserID)
		require.NoError(t, err)
	}

	// Get metrics
	metrics, err := storage.GetMetrics(ctx)
	require.NoError(t, err)

	// Verify metrics
	assert.Equal(t, int64(3), metrics.TotalUsers)
	assert.Equal(t, int64(2), metrics.ActiveUsers)
	assert.Equal(t, int64(10), metrics.ProcessedEmails)
	assert.Equal(t, int64(2), metrics.ValidTokens)
}

func TestSQLiteStorage_GetUserMetrics(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	telegramID := int64(1)
	gmailUserID := "test@example.com"

	// Create user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, time.Hour)
	require.NoError(t, err)

	// Store token
	err = storage.StoreToken(ctx, gmailUserID, []byte("token"), []byte("nonce"))
	require.NoError(t, err)

	// Mark some emails as processed
	for i := 0; i < 3; i++ {
		err = storage.MarkEmailProcessed(ctx, "msg1", gmailUserID)
		require.NoError(t, err)
	}

	// Get user metrics
	metrics, err := storage.GetUserMetrics(ctx, telegramID)
	require.NoError(t, err)

	// Verify metrics
	assert.Equal(t, telegramID, metrics.TelegramID)
	assert.Equal(t, gmailUserID, metrics.GmailUserID)
	assert.Equal(t, int64(3), metrics.ProcessedEmails)
	assert.True(t, metrics.HasValidToken)
	assert.NotZero(t, metrics.LastActive)
}

func TestSQLiteStorage_GetUserMetrics_NonExistentUser(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Try to get metrics for non-existent user
	_, err = storage.GetUserMetrics(context.Background(), 1)
	assert.Error(t, err)
}

func TestSQLiteStorage_GetMetricsWithinTimeRange(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	telegramID := int64(1)
	gmailUserID := "test@example.com"

	// Create user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, time.Hour)
	require.NoError(t, err)

	// Mark some emails as processed
	for i := 0; i < 5; i++ {
		err = storage.MarkEmailProcessed(ctx, "msg1", gmailUserID)
		require.NoError(t, err)
	}

	// Update some timestamps to be older
	_, err = db.Exec(`
		UPDATE processed_emails 
		SET processed_at = datetime('now', '-2 days')
		WHERE message_id = 'msg1'
		LIMIT 2
	`)
	require.NoError(t, err)

	// Get metrics for last 24 hours
	metrics, err := storage.GetMetricsWithinTimeRange(ctx, time.Now().Add(-24*time.Hour), time.Now())
	require.NoError(t, err)

	// Should only count emails processed in the last 24 hours
	assert.Equal(t, int64(3), metrics.ProcessedEmails)
}

func TestSQLiteStorage_GetMetricsInTransaction(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	telegramID := int64(1)
	gmailUserID := "test@example.com"

	// Create user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, time.Hour)
	require.NoError(t, err)

	// Start transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Mark email as processed within transaction
	err = tx.MarkEmailProcessed("msg1", gmailUserID)
	require.NoError(t, err)

	// Get metrics within transaction
	metrics, err := tx.GetMetrics()
	require.NoError(t, err)

	// Should see the changes within the transaction
	assert.Equal(t, int64(1), metrics.ProcessedEmails)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Get metrics outside transaction
	metrics, err = storage.GetMetrics(ctx)
	require.NoError(t, err)

	// Should see the committed changes
	assert.Equal(t, int64(1), metrics.ProcessedEmails)
} 