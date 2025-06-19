package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_Migrate(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	// Verify schema_migrations table exists and has the latest version
	var version int64
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	require.NoError(t, err)
	assert.Greater(t, version, int64(0), "Expected at least one migration to be applied")

	// Verify other tables exist
	tables := []string{"tokens", "users", "processed_emails"}
	for _, table := range tables {
		var exists bool
		err = db.QueryRow(`SELECT EXISTS (
			SELECT 1 FROM sqlite_master WHERE type='table' AND name=?
		)`, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Table %s should exist", table)
	}
}

func TestSQLiteStorage_StoreToken(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	userID := "test@example.com"
	token := []byte("encrypted_token_data")
	nonce := []byte("test_nonce")

	// Test storing token
	err = storage.StoreToken(ctx, userID, token, nonce)
	require.NoError(t, err)

	// Verify token was stored
	var storedToken []byte
	var storedNonce []byte
	err = db.QueryRow("SELECT encrypted_token, nonce FROM tokens WHERE user_id = ?", userID).
		Scan(&storedToken, &storedNonce)
	require.NoError(t, err)
	assert.Equal(t, token, storedToken)
	assert.Equal(t, nonce, storedNonce)

	// Test updating existing token
	newToken := []byte("new_encrypted_token")
	newNonce := []byte("new_nonce")
	err = storage.StoreToken(ctx, userID, newToken, newNonce)
	require.NoError(t, err)

	err = db.QueryRow("SELECT encrypted_token, nonce FROM tokens WHERE user_id = ?", userID).
		Scan(&storedToken, &storedNonce)
	require.NoError(t, err)
	assert.Equal(t, newToken, storedToken)
	assert.Equal(t, newNonce, storedNonce)
}

func TestSQLiteStorage_GetToken(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	userID := "test@example.com"
	token := []byte("encrypted_token_data")
	nonce := []byte("test_nonce")

	// Test getting non-existent token
	_, _, err = storage.GetToken(ctx, userID)
	assert.Error(t, err)

	// Store token
	err = storage.StoreToken(ctx, userID, token, nonce)
	require.NoError(t, err)

	// Test getting existing token
	retrievedToken, retrievedNonce, err := storage.GetToken(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, token, retrievedToken)
	assert.Equal(t, nonce, retrievedNonce)
}

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

	// Test creating new user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Verify user was created
	var storedGmailUserID string
	var storedDigestInterval int64
	err = db.QueryRow(`
		SELECT gmail_user_id, digest_interval 
		FROM users 
		WHERE telegram_id = ?`, telegramID).
		Scan(&storedGmailUserID, &storedDigestInterval)
	require.NoError(t, err)
	assert.Equal(t, gmailUserID, storedGmailUserID)
	assert.Equal(t, int64(digestInterval.Seconds()), storedDigestInterval)

	// Test duplicate user creation
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	assert.Error(t, err, "Creating duplicate user should fail")
}

func TestSQLiteStorage_GetUser(t *testing.T) {
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

	// Test getting non-existent user
	_, err = storage.GetUser(ctx, telegramID)
	assert.Error(t, err)

	// Create user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Test getting existing user
	user, err := storage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, telegramID, user.TelegramID)
	assert.Equal(t, gmailUserID, user.GmailUserID)
	assert.Equal(t, digestInterval, user.DigestInterval)
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

	// Create initial user
	err = storage.CreateUser(ctx, telegramID, gmailUserID, digestInterval)
	require.NoError(t, err)

	// Update user
	newDigestInterval := time.Hour * 4
	err = storage.UpdateUser(ctx, telegramID, newDigestInterval)
	require.NoError(t, err)

	// Verify update
	user, err := storage.GetUser(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, newDigestInterval, user.DigestInterval)
}

func TestSQLiteStorage_MarkEmailProcessed(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	messageID := "test_message_id"
	userID := "test@example.com"

	// Test marking email as processed
	err = storage.MarkEmailProcessed(ctx, messageID, userID)
	require.NoError(t, err)

	// Verify email was marked as processed
	processed, err := storage.IsEmailProcessed(ctx, messageID, userID)
	require.NoError(t, err)
	assert.True(t, processed)

	// Test duplicate marking
	err = storage.MarkEmailProcessed(ctx, messageID, userID)
	assert.NoError(t, err, "Marking already processed email should not error")
}

func TestSQLiteStorage_IsEmailProcessed(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	storage := NewSQLiteStorage(db)
	err = storage.Migrate(context.Background())
	require.NoError(t, err)

	ctx := context.Background()
	messageID := "test_message_id"
	userID := "test@example.com"

	// Test unprocessed email
	processed, err := storage.IsEmailProcessed(ctx, messageID, userID)
	require.NoError(t, err)
	assert.False(t, processed)

	// Mark as processed
	err = storage.MarkEmailProcessed(ctx, messageID, userID)
	require.NoError(t, err)

	// Test processed email
	processed, err = storage.IsEmailProcessed(ctx, messageID, userID)
	require.NoError(t, err)
	assert.True(t, processed)
} 