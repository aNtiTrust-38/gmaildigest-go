package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OAuthManager handles OAuth2 authentication flow with Google
type OAuthManager struct {
	config      *oauth2.Config
	storage     Storage
	pkceStore   PKCEStore
	stateStore  StateStore
	tokenSource oauth2.TokenSource // For testing purposes
}

// Storage interface for token persistence
type Storage interface {
	StoreToken(ctx context.Context, userID string, token []byte, nonce []byte) error
	GetToken(ctx context.Context, userID string) ([]byte, []byte, error)
}

// StateStore manages OAuth state parameter
type StateStore interface {
	StoreState(userID, state string) error
	ValidateState(userID, state string) bool
	DeleteState(userID string)
}

// NewOAuthManager creates a new OAuthManager instance
func NewOAuthManager(storage Storage, pkceStore PKCEStore, stateStore StateStore) *OAuthManager {
	return &OAuthManager{
		storage:    storage,
		pkceStore:  pkceStore,
		stateStore: stateStore,
	}
}

// LoadCredentials loads Google OAuth credentials from a JSON file
func (m *OAuthManager) LoadCredentials(credPath string) error {
	if credPath == "" {
		return fmt.Errorf("credentials path cannot be empty")
	}

	data, err := os.ReadFile(credPath)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	var credConfig struct {
		Web struct {
			ClientID     string   `json:"client_id"`
			ClientSecret string   `json:"client_secret"`
			RedirectURIs []string `json:"redirect_uris"`
		} `json:"web"`
	}

	if err := json.Unmarshal(data, &credConfig); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	m.config = &oauth2.Config{
		ClientID:     credConfig.Web.ClientID,
		ClientSecret: credConfig.Web.ClientSecret,
		RedirectURL:  credConfig.Web.RedirectURIs[0],
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.modify",
		},
		Endpoint: google.Endpoint,
	}

	return nil
}

// GetAuthURL generates the OAuth authorization URL with PKCE
func (m *OAuthManager) GetAuthURL(userID string) (string, string, error) {
	if userID == "" {
		return "", "", fmt.Errorf("user ID cannot be empty")
	}

	// Generate PKCE verifier and challenge
	verifier, err := m.pkceStore.GenerateCodeVerifier(128)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate code verifier: %w", err)
	}

	challenge, err := m.pkceStore.GenerateCodeChallenge(verifier)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate code challenge: %w", err)
	}

	// Generate and store state
	state := generateRandomState()
	if err := m.stateStore.StoreState(userID, state); err != nil {
		return "", "", fmt.Errorf("failed to store state: %w", err)
	}

	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}

	authURL := m.config.AuthCodeURL(state, opts...)
	return authURL, state, nil
}

// ValidateToken checks if a token is valid and not expired
func (m *OAuthManager) ValidateToken(token *oauth2.Token) bool {
	if token == nil {
		return false
	}
	return token.Valid()
}

// RefreshToken refreshes the OAuth token for a given user
func (m *OAuthManager) RefreshToken(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	token, err := m.getToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if token == nil {
		return fmt.Errorf("no token found for user")
	}

	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	var tokenSource oauth2.TokenSource
	if m.tokenSource != nil {
		tokenSource = m.tokenSource
	} else {
		tokenSource = m.config.TokenSource(ctx, token)
	}

	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Preserve the refresh token if the new token doesn't have one
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = token.RefreshToken
	}

	tokenBytes, err := json.Marshal(newToken)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Get the existing nonce
	_, nonce, err := m.storage.GetToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	if err := m.storage.StoreToken(ctx, userID, tokenBytes, nonce); err != nil {
		return fmt.Errorf("failed to store refreshed token: %w", err)
	}

	return nil
}

// getToken retrieves and unmarshals the OAuth token for a user
func (m *OAuthManager) getToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	tokenBytes, _, err := m.storage.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from storage: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// generateRandomState generates a random state parameter for OAuth flow
func generateRandomState() string {
	// Implementation will be added in a separate PR
	return "temporary-state"
}

// SetTokenSource sets a custom TokenSource for testing
func (m *OAuthManager) SetTokenSource(ts oauth2.TokenSource) {
	m.tokenSource = ts
}

// HandleCallback processes the OAuth callback and stores the token
func (m *OAuthManager) HandleCallback(ctx context.Context, code, state, userID string) error {
	if code == "" {
		return fmt.Errorf("authorization code cannot be empty")
	}
	if state == "" {
		return fmt.Errorf("state parameter cannot be empty")
	}
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if !m.stateStore.ValidateState(userID, state) {
		return fmt.Errorf("invalid state parameter")
	}
	defer m.stateStore.DeleteState(userID)

	token, err := m.config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	nonce := []byte(generateRandomState()) // Using state generator for nonce temporarily
	if err := m.storage.StoreToken(ctx, userID, tokenBytes, nonce); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
} 