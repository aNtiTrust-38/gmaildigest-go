package auth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// Mock Storage
type mockStorage struct {
	tokens map[string][]byte
	nonces map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		tokens: make(map[string][]byte),
		nonces: make(map[string][]byte),
	}
}

func (m *mockStorage) StoreToken(ctx context.Context, userID string, token []byte, nonce []byte) error {
	m.tokens[userID] = token
	m.nonces[userID] = nonce
	return nil
}

func (m *mockStorage) GetToken(ctx context.Context, userID string) ([]byte, []byte, error) {
	token, ok := m.tokens[userID]
	if !ok {
		return nil, nil, fmt.Errorf("token not found")
	}
	nonce := m.nonces[userID]
	return token, nonce, nil
}

// Mock PKCE Store
type mockPKCEStore struct{}

func (m *mockPKCEStore) GenerateCodeVerifier(length int) (string, error) {
	return "test-verifier", nil
}

func (m *mockPKCEStore) GenerateCodeChallenge(verifier string) (string, error) {
	return "test-challenge", nil
}

func (m *mockPKCEStore) ValidateChallenge(challenge, verifier string) bool {
	return true
}

// Mock State Store
type mockStateStore struct {
	states map[string]string
}

func newMockStateStore() *mockStateStore {
	return &mockStateStore{
		states: make(map[string]string),
	}
}

func (m *mockStateStore) StoreState(userID, state string) error {
	m.states[userID] = state
	return nil
}

func (m *mockStateStore) ValidateState(userID, state string) bool {
	storedState, exists := m.states[userID]
	return exists && storedState == state
}

func (m *mockStateStore) DeleteState(userID string) {
	delete(m.states, userID)
}

// Mock Token Source
type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &oauth2.Token{
		AccessToken:  "new-token",
		TokenType:    "Bearer",
		Expiry:      time.Now().Add(time.Hour),
		RefreshToken: m.token.RefreshToken,
	}, nil
} 