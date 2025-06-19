package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_TransactionRollback(t *testing.T) {
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

	// Start a transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Create user within transaction
	err = tx.CreateUser(telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Verify user exists within transaction
	user, err := tx.GetUser(telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify user does not exist after rollback
	_, err = storage.GetUser(ctx, telegramID)
	assert.Error(t, err)
}

func TestSQLiteStorage_TransactionCommit(t *testing.T) {
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

	// Start a transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Create user within transaction
	err = tx.CreateUser(telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify user exists after commit
	user, err := storage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)
}

func TestSQLiteStorage_TransactionIsolation(t *testing.T) {
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

	// Start first transaction
	tx1, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Create user in first transaction
	err = tx1.CreateUser(telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Start second transaction
	tx2, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Verify user is not visible in second transaction
	_, err = tx2.GetUser(telegramID)
	assert.Error(t, err)

	// Commit first transaction
	err = tx1.Commit()
	require.NoError(t, err)

	// Now user should be visible in second transaction
	user, err := tx2.GetUser(telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)

	err = tx2.Rollback()
	require.NoError(t, err)
}

func TestSQLiteStorage_TransactionDoubleCommit(t *testing.T) {
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

	// First commit should succeed
	err = tx.Commit()
	require.NoError(t, err)

	// Second commit should fail
	err = tx.Commit()
	assert.Error(t, err)
}

func TestSQLiteStorage_TransactionDoubleRollback(t *testing.T) {
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

	// First rollback should succeed
	err = tx.Rollback()
	require.NoError(t, err)

	// Second rollback should fail
	err = tx.Rollback()
	assert.Error(t, err)
} 