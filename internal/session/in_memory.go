package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

// InMemoryStore is an in-memory implementation of the Store interface.
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]sessionData
}

type sessionData struct {
	userID  string
	expires time.Time
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]sessionData),
	}
}

// Create creates a new session for a user and returns the session ID.
func (s *InMemoryStore) Create(ctx context.Context, userID string, duration time.Duration) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = sessionData{
		userID:  userID,
		expires: time.Now().Add(duration),
	}

	return sessionID, nil
}

// Get retrieves the user ID for a given session ID.
func (s *InMemoryStore) Get(ctx context.Context, sessionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.sessions[sessionID]
	if !ok {
		return "", errors.New("session not found")
	}

	if time.Now().After(data.expires) {
		// The session has expired, but we'll delete it lazily.
		// A separate cleanup routine would handle proactive deletion.
		return "", errors.New("session expired")
	}

	return data.userID, nil
}

// Delete removes a session.
func (s *InMemoryStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

// generateSessionID creates a new random session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
} 