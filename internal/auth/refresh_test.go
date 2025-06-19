package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

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

// mockConfig is a custom oauth2.Config that allows setting a TokenSource
type mockConfig struct {
	*oauth2.Config
	tokenSource oauth2.TokenSource
}

func (c *mockConfig) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	return c.tokenSource
}

func TestOAuthManager_RefreshToken(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	pkceStore := &mockPKCEStore{}
	stateStore := newMockStateStore()

	tests := []struct {
		name    string
		token   *oauth2.Token
		wantErr bool
	}{
		{
			name: "successful refresh",
			token: &oauth2.Token{
				AccessToken:  "old-token",
				TokenType:    "Bearer",
				Expiry:      time.Now().Add(-time.Hour),
				RefreshToken: "refresh-token",
			},
			wantErr: false,
		},
		{
			name: "missing refresh token",
			token: &oauth2.Token{
				AccessToken: "old-token",
				TokenType:  "Bearer",
				Expiry:     time.Now().Add(-time.Hour),
			},
			wantErr: true,
		},
		{
			name:    "nil token",
			token:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &OAuthManager{
				config: &oauth2.Config{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					RedirectURL:  "http://localhost:8080/callback",
					Endpoint:     google.Endpoint,
				},
				storage:    storage,
				pkceStore:  pkceStore,
				stateStore: stateStore,
			}

			userID := "test-user"

			if tt.token != nil {
				tokenBytes, err := json.Marshal(tt.token)
				require.NoError(t, err)
				err = storage.StoreToken(ctx, userID, tokenBytes, []byte("test-nonce"))
				require.NoError(t, err)
				manager.SetTokenSource(&mockTokenSource{token: tt.token})
			}

			err := manager.RefreshToken(ctx, userID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			token, err := manager.getToken(ctx, userID)
			require.NoError(t, err)
			assert.NotEqual(t, tt.token.AccessToken, token.AccessToken)
			assert.True(t, token.Expiry.After(time.Now()))
		})
	}
}

func TestOAuthManager_HandleCallback(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	pkceStore := &mockPKCEStore{}
	stateStore := newMockStateStore()

	manager := &OAuthManager{
		config: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
			Endpoint:     google.Endpoint,
		},
		storage:    storage,
		pkceStore:  pkceStore,
		stateStore: stateStore,
	}

	tests := []struct {
		name      string
		code      string
		state     string
		userID    string
		setupFunc func()
		wantErr   bool
	}{
		{
			name:   "successful exchange",
			code:   "valid-code",
			state:  "valid-state",
			userID: "test-user",
			setupFunc: func() {
				stateStore.StoreState("test-user", "valid-state")
			},
			wantErr: false,
		},
		{
			name:    "invalid state",
			code:    "valid-code",
			state:   "invalid-state",
			userID:  "test-user",
			wantErr: true,
		},
		{
			name:    "empty code",
			code:    "",
			state:   "valid-state",
			userID:  "test-user",
			wantErr: true,
		},
		{
			name:    "empty user ID",
			code:    "valid-code",
			state:   "valid-state",
			userID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			err := manager.HandleCallback(ctx, tt.code, tt.state, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
} 