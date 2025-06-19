package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_CreateUser(t *testing.T) {
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

	// Create user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Verify user exists
	user, err := storage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)
	assert.Equal(t, gmailUserID, user.GmailUserID)
	assert.Equal(t, digestInterval, user.DigestInterval)
	assert.False(t, user.TokenValid)
	assert.Nil(t, user.LastDigestSent)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)
}

func TestSQLiteStorage_CreateDuplicateUser(t *testing.T) {
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

	// Try to create duplicate user with same Gmail ID
	err = storage.CreateUser(ctx, 654321, gmailUserID, digestInterval)
	assert.Error(t, err)
}

func TestSQLiteStorage_UpdateUser(t *testing.T) {
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

	// Create user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Update user
	newDigestInterval := time.Hour * 4
	lastDigestSent := time.Now().UTC().Truncate(time.Second)
	err = storage.UpdateUser(ctx, telegramID, newDigestInterval, &lastDigestSent, true)
	require.NoError(t, err)

	// Verify updates
	user, err := storage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)
	assert.Equal(t, gmailUserID, user.GmailUserID)
	assert.Equal(t, newDigestInterval, user.DigestInterval)
	assert.True(t, user.TokenValid)
	assert.Equal(t, lastDigestSent, *user.LastDigestSent)
}

func TestSQLiteStorage_DeleteUser(t *testing.T) {
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

	// Create user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Store token
	err = storage.StoreToken(ctx, gmailUserID, []byte("token"), []byte("nonce"))
	require.NoError(t, err)

	// Mark email as processed
	err = storage.MarkEmailProcessed(ctx, "msg1", gmailUserID)
	require.NoError(t, err)

	// Delete user
	err = storage.DeleteUser(ctx, telegramID)
	require.NoError(t, err)

	// Verify user is deleted
	_, err = storage.GetUser(ctx, telegramID)
	assert.Error(t, err)

	// Verify token is deleted (due to foreign key cascade)
	_, _, err = storage.GetToken(ctx, gmailUserID)
	assert.Error(t, err)

	// Verify processed emails are deleted (due to foreign key cascade)
	processed, err := storage.IsEmailProcessed(ctx, "msg1", gmailUserID)
	require.NoError(t, err)
	assert.False(t, processed)
}

func TestSQLiteStorage_ListUsers(t *testing.T) {
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

	// List all users
	allUsers, err := storage.ListUsers(ctx)
	require.NoError(t, err)
	assert.Len(t, allUsers, len(users))

	// List users with valid tokens
	validUsers, err := storage.ListUsersWithValidTokens(ctx)
	require.NoError(t, err)
	assert.Len(t, validUsers, 2)

	// List users due for digest
	now := time.Now().UTC()
	dueUsers, err := storage.ListUsersDueForDigest(ctx, now)
	require.NoError(t, err)
	assert.Len(t, dueUsers, 2) // Only users with valid tokens should be due
}

func TestSQLiteStorage_UserOperationsInTransaction(t *testing.T) {
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

	// Start transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Create user in transaction
	err = tx.CreateUser(telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Verify user exists in transaction
	user, err := tx.GetUser(telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)

	// Verify user doesn't exist outside transaction
	_, err = storage.GetUser(ctx, telegramID)
	assert.Error(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify user exists after commit
	user, err = storage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)
}

func TestSQLiteStorage_UserOperationsWithRollback(t *testing.T) {
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

	// Start transaction
	tx, err := storage.BeginTx(ctx)
	require.NoError(t, err)

	// Create user in transaction
	err = tx.CreateUser(telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify user doesn't exist after rollback
	_, err = storage.GetUser(ctx, telegramID)
	assert.Error(t, err)
} 