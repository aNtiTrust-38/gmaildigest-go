package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
)

// InMemoryStateStore provides an in-memory implementation of the StateStore interface.
type InMemoryStateStore struct {
	mu     sync.Mutex
	states map[string]string
}

// NewInMemoryStateStore creates a new InMemoryStateStore.
func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		states: make(map[string]string),
	}
}

// StoreState stores the state for a given user ID.
func (s *InMemoryStateStore) StoreState(userID, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[userID] = state
	return nil
}

// ValidateState validates and then deletes the state for a given user ID.
func (s *InMemoryStateStore) ValidateState(userID, state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if storedState, ok := s.states[userID]; ok && storedState == state {
		delete(s.states, userID)
		return true
	}
	return false
}

// DeleteState removes the state for a given user ID.
func (s *InMemoryStateStore) DeleteState(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, userID)
}

// InMemoryPKCEStore provides an in-memory implementation of the PKCEStore interface.
type InMemoryPKCEStore struct {
	mu        sync.Mutex
	verifiers map[string]string
}

// NewInMemoryPKCEStore creates a new InMemoryPKCEStore.
func NewInMemoryPKCEStore() *InMemoryPKCEStore {
	return &InMemoryPKCEStore{
		verifiers: make(map[string]string),
	}
}

// StoreVerifier stores the code verifier for a given state.
func (s *InMemoryPKCEStore) StoreVerifier(state, verifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verifiers[state] = verifier
	return nil
}

// GetVerifier retrieves and deletes the code verifier for a given state.
func (s *InMemoryPKCEStore) GetVerifier(state string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	verifier, ok := s.verifiers[state]
	if !ok {
		return "", fmt.Errorf("no verifier found for state: %s", state)
	}
	delete(s.verifiers, state)
	return verifier, nil
}

// GenerateCodeVerifier creates a new code verifier.
func (s *InMemoryPKCEStore) GenerateCodeVerifier(length int) (string, error) {
	if length < 43 || length > 128 {
		return "", fmt.Errorf("code verifier length must be between 43 and 128 characters")
	}
	p := make([]byte, length)
	if _, err := rand.Read(p); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(p), nil
}

// GenerateCodeChallenge creates a code challenge from a verifier.
func (s *InMemoryPKCEStore) GenerateCodeChallenge(verifier string) (string, error) {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil)), nil
}

// ValidateChallenge validates a code challenge against a verifier.
func (s *InMemoryPKCEStore) ValidateChallenge(challenge, verifier string) bool {
	calculatedChallenge, _ := s.GenerateCodeChallenge(verifier)
	return challenge == calculatedChallenge
} 