package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_Create(t *testing.T) {
	store := NewInMemoryStore()
	userID := "user-123"

	sessionID, err := store.Create(context.Background(), userID, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, sessionID)
}

func TestInMemoryStore_Get(t *testing.T) {
	store := NewInMemoryStore()
	userID := "user-123"
	ctx := context.Background()

	t.Run("gets a valid session", func(t *testing.T) {
		sessionID, err := store.Create(ctx, userID, time.Hour)
		require.NoError(t, err)

		retrievedUserID, err := store.Get(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, userID, retrievedUserID)
	})

	t.Run("returns error for non-existent session", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent-session-id")
		assert.Error(t, err)
	})

	t.Run("returns error for expired session", func(t *testing.T) {
		sessionID, err := store.Create(ctx, userID, -time.Hour) // Expired an hour ago
		require.NoError(t, err)

		_, err = store.Get(ctx, sessionID)
		assert.Error(t, err)
		assert.EqualError(t, err, "session expired")
	})
}

func TestInMemoryStore_Delete(t *testing.T) {
	store := NewInMemoryStore()
	userID := "user-123"
	ctx := context.Background()

	sessionID, err := store.Create(ctx, userID, time.Hour)
	require.NoError(t, err)

	err = store.Delete(ctx, sessionID)
	require.NoError(t, err)

	_, err = store.Get(ctx, sessionID)
	assert.Error(t, err, "should not be able to get a deleted session")
} 