package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_DuplicateUser(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	telegramID := int64(123456)
	gmailUserID := "test@example.com"
	digestInterval := time.Hour * 2

	// Create first user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Try to create duplicate user with same Telegram ID
	err = storage.CreateUser(ctx, telegramID, "other@example.com", digestInterval)
	assert.Error(t, err)

	// Try to create user with same Gmail ID
	err = storage.CreateUser(ctx, telegramID+1, gmailUserID, digestInterval)
	assert.Error(t, err)
}

func TestSQLiteStorage_NonExistentUser(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Try to get non-existent user
	_, err = storage.GetUser(ctx, 1)
	assert.Error(t, err)

	// Try to update non-existent user
	err = storage.UpdateUser(ctx, 1, time.Hour)
	assert.Error(t, err)
}

func TestSQLiteStorage_InvalidInput(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Try to create user with zero Telegram ID
	err = storage.CreateUser(ctx, 0, "test@example.com", time.Hour)
	assert.Error(t, err)

	// Try to create user with empty Gmail ID
	err = storage.CreateUser(ctx, 1, "", time.Hour)
	assert.Error(t, err)

	// Try to create user with zero digest interval
	err = storage.CreateUser(ctx, 1, "test@example.com", 0)
	assert.Error(t, err)
}

func TestSQLiteStorage_TokenOperations(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Try to get non-existent token
	_, _, err = storage.GetToken(ctx, "nonexistent@example.com")
	assert.Error(t, err)

	// Try to store token with empty user ID
	err = storage.StoreToken(ctx, "", []byte("token"), []byte("nonce"))
	assert.Error(t, err)

	// Try to store token with nil token data
	err = storage.StoreToken(ctx, "test@example.com", nil, []byte("nonce"))
	assert.Error(t, err)

	// Try to store token with nil nonce
	err = storage.StoreToken(ctx, "test@example.com", []byte("token"), nil)
	assert.Error(t, err)
}

func TestSQLiteStorage_EmailProcessing(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Try to mark email as processed with empty message ID
	err = storage.MarkEmailProcessed(ctx, "", "test@example.com")
	assert.Error(t, err)

	// Try to mark email as processed with empty user ID
	err = storage.MarkEmailProcessed(ctx, "msg123", "")
	assert.Error(t, err)

	// Try to check status with empty message ID
	_, err = storage.IsEmailProcessed(ctx, "", "test@example.com")
	assert.Error(t, err)

	// Try to check status with empty user ID
	_, err = storage.IsEmailProcessed(ctx, "msg123", "")
	assert.Error(t, err)
}

func TestSQLiteStorage_ContextCancellation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try operations with cancelled context
	_, err = storage.GetUser(ctx, 1)
	assert.Error(t, err)

	err = storage.CreateUser(ctx, 1, "test@example.com", time.Hour)
	assert.Error(t, err)

	err = storage.UpdateUser(ctx, 1, time.Hour)
	assert.Error(t, err)

	err = storage.StoreToken(ctx, "test@example.com", []byte("token"), []byte("nonce"))
	assert.Error(t, err)

	_, _, err = storage.GetToken(ctx, "test@example.com")
	assert.Error(t, err)

	err = storage.MarkEmailProcessed(ctx, "msg123", "test@example.com")
	assert.Error(t, err)

	_, err = storage.IsEmailProcessed(ctx, "msg123", "test@example.com")
	assert.Error(t, err)
}

func TestSQLiteStorage_TransactionErrors(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()

	// Start a transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Commit the transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Try to use the transaction after it's closed
	err = tx.CreateUser(1, "test@example.com", time.Hour)
	assert.Error(t, err)

	_, err = tx.GetUser(1)
	assert.Error(t, err)

	err = tx.UpdateUser(1, time.Hour)
	assert.Error(t, err)

	err = tx.StoreToken("test@example.com", []byte("token"), []byte("nonce"))
	assert.Error(t, err)

	_, _, err = tx.GetToken("test@example.com")
	assert.Error(t, err)

	err = tx.MarkEmailProcessed("msg123", "test@example.com")
	assert.Error(t, err)

	_, err = tx.IsEmailProcessed("msg123", "test@example.com")
	assert.Error(t, err)
} 