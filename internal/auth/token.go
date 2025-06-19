package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
)

// storeToken stores a token for a user
func (m *OAuthManager) storeToken(ctx context.Context, userID string, token *oauth2.Token) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	// Serialize token
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	// Generate nonce for encryption
	nonce, err := generateNonce()
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Store token
	if err := m.storage.StoreToken(ctx, userID, tokenBytes, nonce); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// getToken retrieves a token for a user
func (m *OAuthManager) getToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Get encrypted token and nonce
	tokenBytes, _, err := m.storage.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Deserialize token
	var token oauth2.Token
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("failed to deserialize token: %w", err)
	}

	return &token, nil
}

// RefreshToken refreshes an expired token
func (m *OAuthManager) RefreshToken(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Get existing token
	token, err := m.getToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if token.RefreshToken == "" {
		return fmt.Errorf("token has no refresh token")
	}

	// Use custom TokenSource if available (for testing)
	var tokenSource oauth2.TokenSource
	if m.tokenSource != nil {
		tokenSource = m.tokenSource
	} else {
		tokenSource = m.config.TokenSource(ctx, token)
	}

	// Get new token
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Store new token
	if err := m.storeToken(ctx, userID, newToken); err != nil {
		return fmt.Errorf("failed to store refreshed token: %w", err)
	}

	return nil
}

// HandleCallback processes the OAuth callback and exchanges the code for a token
func (m *OAuthManager) HandleCallback(ctx context.Context, code, state, userID string) error {
	if code == "" {
		return fmt.Errorf("authorization code cannot be empty")
	}
	if state == "" {
		return fmt.Errorf("state cannot be empty")
	}
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Validate state
	if !m.stateStore.ValidateState(userID, state) {
		return fmt.Errorf("invalid state parameter")
	}
	defer m.stateStore.DeleteState(userID)

	// Exchange code for token
	token, err := m.config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Store token
	if err := m.storeToken(ctx, userID, token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// generateNonce generates a random nonce for token encryption
func generateNonce() ([]byte, error) {
	// Implementation will be added in a separate PR
	return []byte("temporary-nonce"), nil
} 