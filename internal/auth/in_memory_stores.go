package auth

import (
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
	generator PKCEStore
}

// NewInMemoryPKCEStore creates a new InMemoryPKCEStore.
func NewInMemoryPKCEStore() *InMemoryPKCEStore {
	return &InMemoryPKCEStore{
		verifiers: make(map[string]string),
		generator: NewPKCEGenerator(),
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
	return s.generator.GenerateCodeVerifier(length)
}

// GenerateCodeChallenge creates a code challenge from a verifier.
func (s *InMemoryPKCEStore) GenerateCodeChallenge(verifier string) (string, error) {
	return s.generator.GenerateCodeChallenge(verifier)
}

// ValidateChallenge validates a code challenge against a verifier.
func (s *InMemoryPKCEStore) ValidateChallenge(challenge, verifier string) bool {
	return s.generator.ValidateChallenge(challenge, verifier)
} 