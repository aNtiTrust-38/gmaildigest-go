package session

import (
	"context"
	"time"
)

// Store defines the interface for session management.
type Store interface {
	// Create creates a new session for a user and returns the session ID.
	Create(ctx context.Context, userID string, duration time.Duration) (string, error)
	// Get retrieves the user ID for a given session ID.
	Get(ctx context.Context, sessionID string) (string, error)
	// Delete removes a session.
	Delete(ctx context.Context, sessionID string) error
} 