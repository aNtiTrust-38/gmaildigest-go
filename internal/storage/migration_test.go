package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_MigrationVersioning(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)

	// First migration
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Check migration version
	var version int64
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	require.NoError(t, err)
	assert.Greater(t, version, int64(0))

	// Running migrations again should be idempotent
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	var newVersion int64
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&newVersion)
	require.NoError(t, err)
	assert.Equal(t, version, newVersion)
}

func TestSQLiteStorage_TableCreation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Check if all required tables exist
	tables := []string{
		"schema_migrations",
		"tokens",
		"users",
		"processed_emails",
	}

	for _, table := range tables {
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name=?
			)`,
			table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Table %s should exist", table)
	}
}

func TestSQLiteStorage_TableSchema(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Test users table schema
	ctx := context.Background()
	telegramID := int64(123456)
	gmailUserID := "test@example.com"
	digestInterval := time.Hour * 2

	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	user, err := storage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)
	assert.Equal(t, gmailUserID, user.GmailUserID)
	assert.Equal(t, digestInterval, user.DigestInterval)
	assert.False(t, user.TokenValid)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)

	// Test tokens table schema
	token := []byte("test_token")
	nonce := []byte("test_nonce")
	err = storage.StoreToken(ctx, gmailUserID, token, nonce)
	require.NoError(t, err)

	storedToken, storedNonce, err := storage.GetToken(ctx, gmailUserID)
	require.NoError(t, err)
	assert.Equal(t, token, storedToken)
	assert.Equal(t, nonce, storedNonce)

	// Test processed_emails table schema
	messageID := "test_message_id"
	err = storage.MarkEmailProcessed(ctx, messageID, gmailUserID)
	require.NoError(t, err)

	processed, err := storage.IsEmailProcessed(ctx, messageID, gmailUserID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestSQLiteStorage_MigrationFailure(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create a table that will conflict with our migrations
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY
		)
	`)
	require.NoError(t, err)

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	assert.Error(t, err)
}

func TestSQLiteStorage_MigrationConcurrency(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)

	// Run migrations concurrently
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- storage.Migrate(context.Background())
		}()
	}

	// All migrations should complete successfully
	for i := 0; i < 3; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	// Check final migration version
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestSQLiteStorage_MigrationTimeout(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Migration should fail due to timeout
	err = storage.Migrate(ctx)
	assert.Error(t, err)
} 